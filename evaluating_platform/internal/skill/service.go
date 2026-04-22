package skill

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	cryptopkg "evaluating_platform/internal/crypto"
	"evaluating_platform/internal/model"
	"evaluating_platform/internal/repository"
	"evaluating_platform/pkg/logger"
	"evaluating_platform/pkg/storage"
)

type Runner interface {
	Run(ctx context.Context, req *RuntimeRequest) (*RuntimeResponse, error)
}

type Service struct {
	skillRepo   *repository.SkillRepository
	versionRepo *repository.SkillVersionRepository
	runRepo     *repository.SkillRunRepository
	configRepo  *repository.SkillConfigValueRepository
	targetRepo  *repository.TargetLLMRepository
	store       *storage.MinIOClient
	runner      Runner
	keyStore    *cryptopkg.KeyStore
}

func NewService(
	skillRepo *repository.SkillRepository,
	versionRepo *repository.SkillVersionRepository,
	runRepo *repository.SkillRunRepository,
	configRepo *repository.SkillConfigValueRepository,
	targetRepo *repository.TargetLLMRepository,
	store *storage.MinIOClient,
	runner Runner,
	keyStore *cryptopkg.KeyStore,
) *Service {
	return &Service{
		skillRepo:   skillRepo,
		versionRepo: versionRepo,
		runRepo:     runRepo,
		configRepo:  configRepo,
		targetRepo:  targetRepo,
		store:       store,
		runner:      runner,
		keyStore:    keyStore,
	}
}

type SkillBundle struct {
	Skill    *model.Skill         `json:"skill"`
	Version  *model.SkillVersion  `json:"version"`
	Versions []model.SkillVersion `json:"versions,omitempty"`
	Runs     []model.SkillRun     `json:"runs,omitempty"`
	Manifest Manifest             `json:"manifest"`
}

const staleSelfTestGracePeriod = 6 * time.Minute

func (s *Service) ImportArchive(ctx context.Context, expertID uuid.UUID, archive []byte) (*SkillBundle, error) {
	if s.store == nil {
		return nil, fmt.Errorf("minio storage is not configured")
	}

	pkg, err := ParsePackage(archive, DefaultMaxArchiveBytes)
	if err != nil {
		return nil, err
	}

	slug := sanitizeSlug(pkg.Manifest.Name)
	if slug == "" {
		return nil, fmt.Errorf("invalid skill name")
	}

	existing, err := s.skillRepo.GetBySlugForExpert(ctx, slug, expertID)
	if err != nil {
		return nil, err
	}

	var skillItem *model.Skill
	if existing == nil {
		skillItem = &model.Skill{
			ID:                uuid.New(),
			ExpertID:          expertID,
			Name:              strings.TrimSpace(pkg.Manifest.DisplayName),
			Slug:              slug,
			Description:       strings.TrimSpace(pkg.Manifest.Description),
			SkillType:         pkg.Manifest.SkillType,
			Category:          strings.TrimSpace(pkg.Manifest.Category),
			CapabilityProfile: strings.TrimSpace(pkg.Manifest.CapabilityProfile),
			Status:            model.SkillStatusDraft,
		}
		if err := s.skillRepo.Create(ctx, skillItem); err != nil {
			return nil, err
		}
	} else {
		skillItem = existing
		skillItem.Name = strings.TrimSpace(pkg.Manifest.DisplayName)
		skillItem.Description = strings.TrimSpace(pkg.Manifest.Description)
		skillItem.SkillType = pkg.Manifest.SkillType
		skillItem.Category = strings.TrimSpace(pkg.Manifest.Category)
		skillItem.CapabilityProfile = strings.TrimSpace(pkg.Manifest.CapabilityProfile)
		if err := s.skillRepo.UpdateMetadata(ctx, skillItem); err != nil {
			return nil, err
		}
	}
	if existing != nil {
		if err := s.ensureConfigSchemaCompatibility(ctx, skillItem.ID, pkg.Manifest); err != nil {
			return nil, err
		}
	}

	objectPath := fmt.Sprintf("skills/%s/%s/%s.zip", expertID.String(), skillItem.ID.String(), pkg.Manifest.Version)
	if err := s.store.Upload(ctx, objectPath, pkg.ArchiveBytes, "application/zip"); err != nil {
		return nil, fmt.Errorf("upload skill archive: %w", err)
	}

	version := &model.SkillVersion{
		ID:                     uuid.New(),
		SkillID:                skillItem.ID,
		Version:                pkg.Manifest.Version,
		ManifestVersion:        pkg.Manifest.ManifestVersion,
		DisplayName:            strings.TrimSpace(pkg.Manifest.DisplayName),
		Summary:                strings.TrimSpace(pkg.Manifest.Description),
		PackageObjectPath:      objectPath,
		PackageHash:            pkg.ArchiveHash,
		PackageSize:            int64(len(pkg.ArchiveBytes)),
		PromptText:             pkg.PromptText,
		InputSourceMode:        pkg.Manifest.InputSourceMode,
		ExecutionRuntime:       pkg.Manifest.Execution.Runtime,
		ExecutionEntrypoint:    pkg.Manifest.Execution.Entrypoint,
		SelfTestEntrypoint:     pkg.Manifest.Execution.SelfTestEntrypoint,
		Permissions:            mustJSON(pkg.Manifest.Permissions, `{}`),
		EmbeddedDatasetSummary: mustJSON(pkg.Manifest.EmbeddedResources.Summary, `{}`),
		AssessmentTypes:        mustJSON(pkg.Manifest.AssessmentTypes, `[]`),
		Metadata:               mustJSON(buildSkillVersionMetadata(pkg.Manifest), `{}`),
		ValidationReport:       pkg.ValidationReport,
		Examples:               mustJSON(map[string]json.RawMessage{"input": pkg.ExampleInput, "output": pkg.ExampleOutput}, `{}`),
		Status:                 model.SkillVersionStatusDraft,
	}
	if err := s.versionRepo.Create(ctx, version); err != nil {
		return nil, err
	}
	if err := s.skillRepo.SetLatestVersion(ctx, skillItem.ID, version.ID); err != nil {
		return nil, err
	}
	skillItem.LatestVersionID = &version.ID

	return &SkillBundle{
		Skill:    skillItem,
		Version:  version,
		Manifest: pkg.Manifest,
	}, nil
}

func (s *Service) ListByExpert(ctx context.Context, expertID uuid.UUID) ([]SkillBundle, error) {
	items, err := s.skillRepo.ListByExpert(ctx, expertID)
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if err := s.reconcileStaleSelfTests(ctx, item.ID); err != nil {
			return nil, err
		}
	}
	result := make([]SkillBundle, 0, len(items))
	for _, item := range items {
		bundle, bundleErr := s.GetByID(ctx, item.ID, expertID)
		if bundleErr != nil {
			return nil, bundleErr
		}
		if bundle != nil {
			result = append(result, *bundle)
		}
	}
	return result, nil
}

func (s *Service) GetByID(ctx context.Context, skillID, expertID uuid.UUID) (*SkillBundle, error) {
	item, err := s.skillRepo.GetByIDForExpert(ctx, skillID, expertID)
	if err != nil || item == nil {
		return nil, err
	}
	if err := s.reconcileStaleSelfTests(ctx, skillID); err != nil {
		return nil, err
	}
	versions, err := s.versionRepo.ListBySkill(ctx, skillID)
	if err != nil {
		return nil, err
	}
	runs, err := s.runRepo.ListBySkill(ctx, skillID)
	if err != nil {
		return nil, err
	}
	var latest *model.SkillVersion
	if item.LatestVersionID != nil {
		latest, _ = s.versionRepo.GetByID(ctx, *item.LatestVersionID)
	}
	var manifest Manifest
	if latest != nil {
		manifest = manifestFromVersion(item.SkillType, *latest)
	}
	return &SkillBundle{
		Skill:    item,
		Version:  latest,
		Versions: versions,
		Runs:     runs,
		Manifest: manifest,
	}, nil
}

func (s *Service) RunSelfTest(ctx context.Context, skillID, versionID, expertID uuid.UUID) (*model.SkillRun, error) {
	item, err := s.skillRepo.GetByIDForExpert(ctx, skillID, expertID)
	if err != nil || item == nil {
		return nil, fmt.Errorf("skill not found")
	}
	version, err := s.versionRepo.GetByID(ctx, versionID)
	if err != nil || version == nil || version.SkillID != skillID {
		return nil, fmt.Errorf("skill version not found")
	}
	if err := s.reconcileStaleSelfTests(ctx, skillID); err != nil {
		return nil, err
	}
	if item.SkillType == "interactive_web_skill" {
		return s.executeInteractiveSelfTest(ctx, item, version)
	}
	return s.executeRun(ctx, item, version, model.SkillRunTypeSelfTest, "expert_portal", nil, nil)
}

func (s *Service) Publish(ctx context.Context, skillID, versionID, expertID uuid.UUID) error {
	item, err := s.skillRepo.GetByIDForExpert(ctx, skillID, expertID)
	if err != nil || item == nil {
		return fmt.Errorf("skill not found")
	}
	var version *model.SkillVersion
	if versionID == uuid.Nil {
		if item.LatestVersionID != nil {
			version, err = s.versionRepo.GetByID(ctx, *item.LatestVersionID)
		} else {
			version, err = s.versionRepo.GetLatestBySkillID(ctx, skillID)
		}
	} else {
		version, err = s.versionRepo.GetByID(ctx, versionID)
	}
	if err != nil || version == nil || version.SkillID != skillID {
		return fmt.Errorf("skill version not found")
	}
	if version.Status != model.SkillVersionStatusSelfTestPassed && version.Status != model.SkillVersionStatusPublished {
		return fmt.Errorf("latest skill version must pass self-test before publish")
	}
	if err := s.validateSkillConfigForPublish(ctx, skillID, version); err != nil {
		return err
	}
	if err := s.versionRepo.ClearPublishedBySkill(ctx, skillID, &version.ID); err != nil {
		return err
	}
	if err := s.versionRepo.MarkPublished(ctx, version.ID); err != nil {
		return err
	}
	return s.skillRepo.PublishVersion(ctx, skillID, version.ID)
}

func (s *Service) Disable(ctx context.Context, skillID, expertID uuid.UUID) error {
	item, err := s.skillRepo.GetByIDForExpert(ctx, skillID, expertID)
	if err != nil || item == nil {
		return fmt.Errorf("skill not found")
	}
	if item.Status == model.SkillStatusDisabled {
		return nil
	}
	return s.skillRepo.SetStatus(ctx, skillID, expertID, model.SkillStatusDisabled)
}

func (s *Service) Enable(ctx context.Context, skillID, expertID uuid.UUID) error {
	item, err := s.skillRepo.GetByIDForExpert(ctx, skillID, expertID)
	if err != nil || item == nil {
		return fmt.Errorf("skill not found")
	}

	nextStatus := model.SkillStatusDraft
	if item.PublishedVersionID != nil {
		if err := s.versionRepo.ClearPublishedBySkill(ctx, skillID, item.PublishedVersionID); err != nil {
			return err
		}
		if err := s.versionRepo.MarkPublished(ctx, *item.PublishedVersionID); err != nil {
			return err
		}
		nextStatus = model.SkillStatusPublished
	}
	return s.skillRepo.SetStatus(ctx, skillID, expertID, nextStatus)
}

func (s *Service) Delete(ctx context.Context, skillID, expertID uuid.UUID) error {
	item, err := s.skillRepo.GetByIDForExpert(ctx, skillID, expertID)
	if err != nil || item == nil {
		return fmt.Errorf("skill not found")
	}

	versions, err := s.versionRepo.ListBySkill(ctx, skillID)
	if err != nil {
		return err
	}

	paths := make(map[string]struct{}, len(versions))
	for _, version := range versions {
		if path := strings.TrimSpace(version.PackageObjectPath); path != "" {
			paths[path] = struct{}{}
		}
	}

	if err := s.skillRepo.Delete(ctx, skillID, expertID); err != nil {
		return err
	}

	if s.store != nil {
		for path := range paths {
			if err := s.store.Delete(ctx, path); err != nil {
				logger.Warn("delete skill archive failed after skill removal", map[string]interface{}{
					"skill_id":     skillID.String(),
					"object_path":  path,
					"cleanup_err":  err.Error(),
					"expert_id":    expertID.String(),
					"skill_status": item.Status,
				})
			}
		}
	}

	return nil
}

func (s *Service) Deprecate(ctx context.Context, skillID, expertID uuid.UUID) error {
	return s.Disable(ctx, skillID, expertID)
}

func (s *Service) ListRuns(ctx context.Context, skillID, expertID uuid.UUID) ([]model.SkillRun, error) {
	item, err := s.skillRepo.GetByIDForExpert(ctx, skillID, expertID)
	if err != nil || item == nil {
		return nil, fmt.Errorf("skill not found")
	}
	if err := s.reconcileStaleSelfTests(ctx, skillID); err != nil {
		return nil, err
	}
	return s.runRepo.ListBySkill(ctx, skillID)
}

func (s *Service) reconcileStaleSelfTests(ctx context.Context, skillID uuid.UUID) error {
	if s.runRepo == nil {
		return nil
	}
	runs, err := s.runRepo.ListBySkill(ctx, skillID)
	if err != nil {
		return err
	}

	cutoff := time.Now().Add(-staleSelfTestGracePeriod)
	for i := range runs {
		run := runs[i]
		if run.RunType != model.SkillRunTypeSelfTest || run.Status != model.SkillRunStatusRunning {
			continue
		}
		if run.StartedAt == nil || !run.StartedAt.Before(cutoff) {
			continue
		}

		exitCode := 124
		completedAt := time.Now()
		run.Status = model.SkillRunStatusTimeout
		run.ExitCode = &exitCode
		run.ErrorMessage = "stale self-test was marked as timeout after the backend did not finish updating the run"
		run.CompletedAt = &completedAt
		run.ValidationReport = mustJSON(map[string]interface{}{
			"skill_id":      skillID.String(),
			"skill_version": run.SkillVersionID.String(),
			"run_id":        run.ID.String(),
			"run_type":      run.RunType,
			"all_passed":    false,
			"notes": []string{
				run.ErrorMessage,
			},
		}, `{}`)
		if err := s.runRepo.UpdateResult(ctx, &run); err != nil {
			return err
		}
		if s.versionRepo != nil {
			if err := s.versionRepo.UpdateRunResult(
				ctx,
				run.SkillVersionID,
				model.SkillVersionStatusSelfTestFailed,
				&run.ID,
				run.ValidationReport,
			); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) SearchPublished(ctx context.Context, assessmentTypes []string, goal string) ([]SkillBundle, error) {
	items, err := s.skillRepo.SearchPublishedSkills(ctx, assessmentTypes, goal)
	if err != nil {
		return nil, err
	}
	result := make([]SkillBundle, 0, len(items))
	for _, item := range items {
		if item.SkillType != "generator_skill" {
			continue
		}
		if item.PublishedVersionID == nil {
			continue
		}
		version, versionErr := s.versionRepo.GetByID(ctx, *item.PublishedVersionID)
		if versionErr != nil || version == nil {
			continue
		}
		if !s.publishedSkillConfigReady(ctx, item.ID, version) {
			continue
		}
		result = append(result, SkillBundle{
			Skill:    &item,
			Version:  version,
			Manifest: manifestFromVersion(item.SkillType, *version),
		})
	}
	return result, nil
}

func (s *Service) GetPublishedByID(ctx context.Context, skillID uuid.UUID) (*SkillBundle, error) {
	item, err := s.skillRepo.GetByID(ctx, skillID)
	if err != nil || item == nil || item.Status != model.SkillStatusPublished || item.PublishedVersionID == nil {
		return nil, err
	}
	version, err := s.versionRepo.GetByID(ctx, *item.PublishedVersionID)
	if err != nil || version == nil {
		return nil, err
	}
	if !s.publishedSkillConfigReady(ctx, item.ID, version) {
		return nil, fmt.Errorf("published skill not found")
	}
	return &SkillBundle{
		Skill:    item,
		Version:  version,
		Manifest: manifestFromVersion(item.SkillType, *version),
	}, nil
}

func (s *Service) GeneratePayloads(ctx context.Context, skillID uuid.UUID, req GenerateRequest) (*GenerateResult, *model.SkillRun, error) {
	item, err := s.skillRepo.GetByID(ctx, skillID)
	if err != nil || item == nil || item.Status != model.SkillStatusPublished || item.PublishedVersionID == nil {
		return nil, nil, fmt.Errorf("published skill not found")
	}
	if item.SkillType != "generator_skill" {
		return nil, nil, fmt.Errorf("skill %s is not a generator_skill", skillID.String())
	}
	version, err := s.versionRepo.GetByID(ctx, *item.PublishedVersionID)
	if err != nil || version == nil {
		return nil, nil, fmt.Errorf("published skill version not found")
	}
	if !s.publishedSkillConfigReady(ctx, item.ID, version) {
		return nil, nil, fmt.Errorf("published skill not found")
	}
	if version.InputSourceMode == "platform_resource_only" && len(req.SourceSamples) == 0 {
		return nil, nil, fmt.Errorf("skill requires platform samples but none were provided")
	}

	input := &GenerateInput{
		AssessmentID:    req.AssessmentID,
		UserID:          req.UserID,
		Goal:            req.Goal,
		AssessmentTypes: req.AssessmentTypes,
		RequestedCount:  req.RequestedCount,
		SourceSampleID:  req.SourceSampleID,
		SourceSamples:   req.SourceSamples,
		ExecutionMeta: map[string]interface{}{
			"input_source_mode":   version.InputSourceMode,
			"skill_id":            skillID.String(),
			"skill_version_id":    version.ID.String(),
			"source_sample_count": len(req.SourceSamples),
		},
	}
	if strings.TrimSpace(req.SourceSampleID) != "" {
		input.ExecutionMeta["source_sample_id"] = req.SourceSampleID
	}
	if s.targetRepo != nil && req.UserID != "" {
		if parsedUserID, parseErr := uuid.Parse(req.UserID); parseErr == nil {
			if cfg, cfgErr := s.targetRepo.GetByUserID(ctx, parsedUserID); cfgErr == nil && cfg != nil {
				input.TargetProfile = map[string]interface{}{
					"type":     cfg.ConnectorType,
					"base_url": cfg.BaseURL,
					"api_key":  cfg.APIKey,
					"model":    cfg.Model,
				}
			}
		}
	}

	run, err := s.executeRun(ctx, item, version, model.SkillRunTypeGenerate, "orchestrator", input, parseAssessmentID(req.AssessmentID))
	if err != nil {
		return nil, nil, err
	}

	var result GenerateResult
	if err := json.Unmarshal(run.ResultPayload, &result); err != nil {
		return nil, run, fmt.Errorf("parse skill generate result: %w", err)
	}
	if len(result.PayloadDataset.Payloads) == 0 {
		return nil, run, fmt.Errorf("skill generated empty payload dataset")
	}
	return &result, run, nil
}

func (s *Service) executeRun(ctx context.Context, item *model.Skill, version *model.SkillVersion, runType model.SkillRunType, triggerSource string, input any, assessmentID *uuid.UUID) (*model.SkillRun, error) {
	if s.runner == nil {
		return nil, fmt.Errorf("skill runner is not configured")
	}
	if s.store == nil {
		return nil, fmt.Errorf("minio storage is not configured")
	}

	archiveBytes, err := s.store.Download(ctx, version.PackageObjectPath)
	if err != nil {
		return nil, fmt.Errorf("download skill archive: %w", err)
	}

	var payload json.RawMessage
	if input != nil {
		payload = mustJSON(input, `{}`)
	}

	now := time.Now()
	run := &model.SkillRun{
		ID:             uuid.New(),
		SkillID:        item.ID,
		SkillVersionID: version.ID,
		AssessmentID:   assessmentID,
		RunType:        runType,
		TriggerSource:  triggerSource,
		Status:         model.SkillRunStatusRunning,
		StartedAt:      &now,
	}
	if err := s.runRepo.Create(ctx, run); err != nil {
		return nil, err
	}

	entrypoint := version.ExecutionEntrypoint
	if runType == model.SkillRunTypeSelfTest {
		entrypoint = version.SelfTestEntrypoint
	}
	req := &RuntimeRequest{
		RunType:        string(runType),
		ArchiveBase64:  base64.StdEncoding.EncodeToString(archiveBytes),
		Runtime:        version.ExecutionRuntime,
		Entrypoint:     entrypoint,
		TimeoutSeconds: 180,
		Input:          payload,
	}
	if item.SkillType == "generator_skill" {
		env, envErr := s.buildRuntimeEnv(ctx, item.ID, version)
		if envErr != nil {
			run.Status = model.SkillRunStatusFailed
			run.ErrorMessage = envErr.Error()
			_ = s.runRepo.UpdateResult(ctx, run)
			if runType == model.SkillRunTypeSelfTest {
				_ = s.versionRepo.UpdateRunResult(ctx, version.ID, model.SkillVersionStatusSelfTestFailed, &run.ID, nil)
			}
			return nil, envErr
		}
		req.Env = env
	}
	resp, err := s.runner.Run(ctx, req)
	completedAt := time.Now()
	run.CompletedAt = &completedAt
	if err != nil {
		run.Status = model.SkillRunStatusFailed
		run.ErrorMessage = err.Error()
		_ = s.runRepo.UpdateResult(ctx, run)
		if runType == model.SkillRunTypeSelfTest {
			_ = s.versionRepo.UpdateRunResult(ctx, version.ID, model.SkillVersionStatusSelfTestFailed, &run.ID, nil)
		}
		return nil, err
	}

	run.StdoutLog = resp.Stdout
	run.StderrLog = resp.Stderr
	run.ResultPayload = resp.Result
	run.ValidationReport = resp.ValidationReport
	run.ErrorMessage = resp.Error
	run.ExitCode = resp.ExitCode
	switch resp.Status {
	case "completed":
		run.Status = model.SkillRunStatusCompleted
	case "timeout":
		run.Status = model.SkillRunStatusTimeout
	default:
		run.Status = model.SkillRunStatusFailed
	}

	if runType == model.SkillRunTypeGenerate {
		var generated GenerateResult
		if err := json.Unmarshal(resp.Result, &generated); err == nil {
			run.PayloadDatasetSummary = mustJSON(map[string]interface{}{
				"count":   len(generated.PayloadDataset.Payloads),
				"summary": generated.PayloadDataset.Summary,
			}, `{}`)
		}
	}
	if err := s.runRepo.UpdateResult(ctx, run); err != nil {
		return nil, err
	}

	if runType == model.SkillRunTypeSelfTest {
		status := model.SkillVersionStatusSelfTestFailed
		if run.Status == model.SkillRunStatusCompleted {
			status = model.SkillVersionStatusSelfTestPassed
		}
		if err := s.versionRepo.UpdateRunResult(ctx, version.ID, status, &run.ID, resp.ValidationReport); err != nil {
			return nil, err
		}
	}

	return run, nil
}

func manifestFromVersion(skillType string, version model.SkillVersion) Manifest {
	var permissions map[string]interface{}
	var metadata map[string]interface{}
	var embedded map[string]interface{}
	var assessmentTypes []string
	_ = json.Unmarshal(version.Permissions, &permissions)
	_ = json.Unmarshal(version.Metadata, &metadata)
	_ = json.Unmarshal(version.EmbeddedDatasetSummary, &embedded)
	_ = json.Unmarshal(version.AssessmentTypes, &assessmentTypes)
	envelope, _ := configSchemaEnvelopeFromMetadata(metadata)
	planner := metadataPlanner(metadata)
	web := metadataWeb(metadata)
	return Manifest{
		ManifestVersion:   version.ManifestVersion,
		SkillType:         defaultString(skillType, metadataString(metadata, "skill_type", "generator_skill")),
		Name:              metadataString(metadata, "skill_name", version.DisplayName),
		DisplayName:       version.DisplayName,
		Version:           version.Version,
		Description:       version.Summary,
		CapabilityProfile: metadataString(metadata, "capability_profile", ""),
		InputSourceMode:   version.InputSourceMode,
		AssessmentTypes:   assessmentTypes,
		Permissions:       permissions,
		Metadata:          metadata,
		ConfigSchema:      envelope.Fields,
		Planner:           planner,
		Web:               web,
		Execution: ExecutionManifest{
			Runtime:            version.ExecutionRuntime,
			Entrypoint:         version.ExecutionEntrypoint,
			SelfTestEntrypoint: version.SelfTestEntrypoint,
		},
		EmbeddedResources:      EmbeddedResources{Summary: embedded},
		ConfigSchemaCompatMode: envelope.CompatMode,
		ConfigSchemaSource:     envelope.Source,
	}
}

func (s *Service) ListPublishedInteractiveSkills(ctx context.Context) ([]LaunchSkillCandidate, error) {
	items, err := s.skillRepo.ListPublished(ctx)
	if err != nil {
		return nil, err
	}

	candidates := make([]LaunchSkillCandidate, 0)
	for _, item := range items {
		if item.SkillType != "interactive_web_skill" || item.PublishedVersionID == nil {
			continue
		}

		version, versionErr := s.versionRepo.GetByID(ctx, *item.PublishedVersionID)
		if versionErr != nil || version == nil {
			continue
		}
		manifest := manifestFromVersion(item.SkillType, *version)
		if !plannerVisible(manifest.Planner) {
			continue
		}

		candidates = append(candidates, LaunchSkillCandidate{
			SkillID:           item.ID,
			SkillName:         firstNonEmptyString(strings.TrimSpace(version.DisplayName), strings.TrimSpace(item.Name), strings.TrimSpace(manifest.DisplayName)),
			SkillSlug:         item.Slug,
			Description:       firstNonEmptyString(strings.TrimSpace(version.Summary), strings.TrimSpace(item.Description), strings.TrimSpace(manifest.Description)),
			PlannerSummary:    firstNonEmptyString(strings.TrimSpace(manifest.Planner.Summary), strings.TrimSpace(version.Summary), strings.TrimSpace(item.Description)),
			IntentExamples:    manifest.Planner.IntentExamples,
			DeliveryMode:      firstNonEmptyString(strings.TrimSpace(manifest.Planner.DeliveryMode), "open_url"),
			CapabilityProfile: firstNonEmptyString(strings.TrimSpace(item.CapabilityProfile), strings.TrimSpace(manifest.CapabilityProfile)),
		})
	}
	return candidates, nil
}

func (s *Service) BuildLaunchDocument(ctx context.Context, skillID uuid.UUID) (*LaunchDocument, error) {
	if s.store == nil {
		return nil, fmt.Errorf("minio storage is not configured")
	}

	item, err := s.skillRepo.GetByID(ctx, skillID)
	if err != nil || item == nil || item.Status != model.SkillStatusPublished || item.PublishedVersionID == nil {
		return nil, fmt.Errorf("published interactive skill not found")
	}
	if item.SkillType != "interactive_web_skill" {
		return nil, fmt.Errorf("skill is not launchable")
	}

	version, err := s.versionRepo.GetByID(ctx, *item.PublishedVersionID)
	if err != nil || version == nil {
		return nil, fmt.Errorf("published skill version not found")
	}

	archiveBytes, err := s.store.Download(ctx, version.PackageObjectPath)
	if err != nil {
		return nil, fmt.Errorf("download skill archive: %w", err)
	}
	pkg, err := ParsePackage(archiveBytes, DefaultMaxArchiveBytes)
	if err != nil {
		return nil, fmt.Errorf("parse skill archive: %w", err)
	}
	title, htmlText, _, err := prepareInteractiveLaunch(pkg)
	if err != nil {
		return nil, err
	}

	return &LaunchDocument{
		SkillID:       item.ID,
		SkillName:     firstNonEmptyString(strings.TrimSpace(version.DisplayName), strings.TrimSpace(item.Name)),
		SkillSlug:     item.Slug,
		VersionID:     version.ID,
		Version:       version.Version,
		DocumentTitle: title,
		HTML:          htmlText,
		LaunchMode:    "open_url",
	}, nil
}

func (s *Service) executeInteractiveSelfTest(ctx context.Context, item *model.Skill, version *model.SkillVersion) (*model.SkillRun, error) {
	if s.store == nil {
		return nil, fmt.Errorf("minio storage is not configured")
	}

	now := time.Now()
	run := &model.SkillRun{
		ID:             uuid.New(),
		SkillID:        item.ID,
		SkillVersionID: version.ID,
		RunType:        model.SkillRunTypeSelfTest,
		TriggerSource:  "expert_portal",
		Status:         model.SkillRunStatusRunning,
		StartedAt:      &now,
	}
	if err := s.runRepo.Create(ctx, run); err != nil {
		return nil, err
	}

	archiveBytes, err := s.store.Download(ctx, version.PackageObjectPath)
	completedAt := time.Now()
	run.CompletedAt = &completedAt
	if err != nil {
		run.Status = model.SkillRunStatusFailed
		run.ErrorMessage = fmt.Sprintf("download skill archive: %v", err)
		_ = s.runRepo.UpdateResult(ctx, run)
		_ = s.versionRepo.UpdateRunResult(ctx, version.ID, model.SkillVersionStatusSelfTestFailed, &run.ID, nil)
		return nil, err
	}

	pkg, err := ParsePackage(archiveBytes, DefaultMaxArchiveBytes)
	if err != nil {
		run.Status = model.SkillRunStatusFailed
		run.ErrorMessage = fmt.Sprintf("parse skill archive: %v", err)
		_ = s.runRepo.UpdateResult(ctx, run)
		_ = s.versionRepo.UpdateRunResult(ctx, version.ID, model.SkillVersionStatusSelfTestFailed, &run.ID, nil)
		return nil, err
	}

	title, _, report, validateErr := prepareInteractiveLaunch(pkg)
	run.ValidationReport = report
	run.ResultPayload = mustJSON(map[string]interface{}{
		"document_title": title,
		"launch_mode":    "open_url",
	}, `{}`)
	if validateErr != nil {
		run.Status = model.SkillRunStatusFailed
		exitCode := 1
		run.ExitCode = &exitCode
		run.ErrorMessage = validateErr.Error()
		run.StdoutLog = "interactive_web_skill validation failed"
		run.StderrLog = validateErr.Error()
		if err := s.runRepo.UpdateResult(ctx, run); err != nil {
			return nil, err
		}
		if err := s.versionRepo.UpdateRunResult(ctx, version.ID, model.SkillVersionStatusSelfTestFailed, &run.ID, report); err != nil {
			return nil, err
		}
		return nil, validateErr
	}

	run.Status = model.SkillRunStatusCompleted
	exitCode := 0
	run.ExitCode = &exitCode
	run.StdoutLog = fmt.Sprintf("interactive_web_skill self-test passed; entrypoint=%s", strings.TrimSpace(pkg.Manifest.Web.Entrypoint))
	if err := s.runRepo.UpdateResult(ctx, run); err != nil {
		return nil, err
	}
	if err := s.versionRepo.UpdateRunResult(ctx, version.ID, model.SkillVersionStatusSelfTestPassed, &run.ID, report); err != nil {
		return nil, err
	}
	return run, nil
}

func metadataString(metadata map[string]interface{}, key string, fallback string) string {
	if value, ok := metadata[key].(string); ok && strings.TrimSpace(value) != "" {
		return value
	}
	return fallback
}

func mustJSON(value any, fallback string) json.RawMessage {
	data, err := json.Marshal(value)
	if err != nil || len(data) == 0 {
		return json.RawMessage(fallback)
	}
	return data
}

func sanitizeSlug(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	replacer := strings.NewReplacer("-", "_", " ", "_")
	value = replacer.Replace(value)
	var builder strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			builder.WriteRune(r)
		}
	}
	slug := strings.Trim(builder.String(), "_")
	for strings.Contains(slug, "__") {
		slug = strings.ReplaceAll(slug, "__", "_")
	}
	return slug
}

func parseAssessmentID(value string) *uuid.UUID {
	if parsed, err := uuid.Parse(strings.TrimSpace(value)); err == nil {
		return &parsed
	}
	return nil
}
