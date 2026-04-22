package chat

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"unicode"

	"evaluating_platform/internal/model"
	"evaluating_platform/internal/repository"
	skillpkg "evaluating_platform/internal/skill"
)

const defaultWelcomeCapabilityLimit = 6

type WelcomeCapability struct {
	ID          string `json:"id"`
	Label       string `json:"label"`
	Prompt      string `json:"prompt"`
	Tone        string `json:"tone"`
	SourceKind  string `json:"source_kind"`
	SourceType  string `json:"source_type,omitempty"`
	Description string `json:"description,omitempty"`
}

type WelcomeCapabilitiesService struct {
	sampleRepo            *repository.AttackSampleRepository
	composedAttackRepo    *repository.ComposedAttackRepository
	templateRepo          *repository.TemplateRepository
	externalMCPServerRepo *repository.ExternalMCPServerRepository
	skillService          *skillpkg.Service
}

type welcomeCandidate struct {
	WelcomeCapability
	priority int
}

func NewWelcomeCapabilitiesService(
	sampleRepo *repository.AttackSampleRepository,
	composedAttackRepo *repository.ComposedAttackRepository,
	templateRepo *repository.TemplateRepository,
	externalMCPServerRepo *repository.ExternalMCPServerRepository,
	skillService *skillpkg.Service,
) *WelcomeCapabilitiesService {
	return &WelcomeCapabilitiesService{
		sampleRepo:            sampleRepo,
		composedAttackRepo:    composedAttackRepo,
		templateRepo:          templateRepo,
		externalMCPServerRepo: externalMCPServerRepo,
		skillService:          skillService,
	}
}

func (s *WelcomeCapabilitiesService) List(ctx context.Context, limit int) ([]WelcomeCapability, error) {
	if limit <= 0 || limit > defaultWelcomeCapabilityLimit {
		limit = defaultWelcomeCapabilityLimit
	}

	skillCaps, err := s.skillCapabilities(ctx)
	if err != nil {
		return nil, err
	}
	mcpCaps, err := s.mcpCapabilities(ctx)
	if err != nil {
		return nil, err
	}
	resourceCaps, err := s.resourceCapabilities(ctx)
	if err != nil {
		return nil, err
	}

	final := assembleWelcomeCapabilities(limit, skillCaps, mcpCaps, resourceCaps)
	items := make([]WelcomeCapability, 0, len(final))
	for _, item := range final {
		items = append(items, item.WelcomeCapability)
	}
	return items, nil
}

func (s *WelcomeCapabilitiesService) skillCapabilities(ctx context.Context) ([]welcomeCandidate, error) {
	if s.skillService == nil {
		return nil, nil
	}

	candidates := make([]welcomeCandidate, 0)

	generatorSkills, err := s.skillService.SearchPublished(ctx, nil, "")
	if err != nil {
		return nil, fmt.Errorf("list published generator skills: %w", err)
	}
	for _, item := range generatorSkills {
		candidates = append(candidates, buildGeneratorSkillCapability(item))
	}

	interactiveSkills, err := s.skillService.ListPublishedInteractiveSkills(ctx)
	if err != nil {
		return nil, fmt.Errorf("list published interactive skills: %w", err)
	}
	for _, item := range interactiveSkills {
		candidates = append(candidates, buildInteractiveSkillCapability(item))
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].priority > candidates[j].priority
	})
	return uniqueWelcomeCandidates(candidates), nil
}

func (s *WelcomeCapabilitiesService) mcpCapabilities(ctx context.Context) ([]welcomeCandidate, error) {
	if s.externalMCPServerRepo == nil {
		return nil, nil
	}

	items, err := s.externalMCPServerRepo.ListEnabled(ctx)
	if err != nil {
		return nil, fmt.Errorf("list enabled external mcp servers: %w", err)
	}

	candidates := make([]welcomeCandidate, 0, len(items))
	for _, item := range items {
		if !externalServerVisible(item) {
			continue
		}
		candidates = append(candidates, buildMCPCapability(item))
	}
	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].priority > candidates[j].priority
	})
	return uniqueWelcomeCandidates(candidates), nil
}

func (s *WelcomeCapabilitiesService) resourceCapabilities(ctx context.Context) ([]welcomeCandidate, error) {
	candidates := make([]welcomeCandidate, 0, 5)

	if s.templateRepo != nil {
		if total, err := publishedTemplateCount(ctx, s.templateRepo, "multilingual"); err != nil {
			return nil, err
		} else if total > 0 {
			candidates = append(candidates, welcomeCandidate{
				WelcomeCapability: WelcomeCapability{
					ID:          "resource-template-multilingual",
					Label:       "多语种绕过测试",
					Prompt:      "请设计一轮覆盖多语种输入的安全评估，重点检查跨语言绕过与规避风险。",
					Tone:        "engine",
					SourceKind:  "resource",
					SourceType:  "multilingual_template",
					Description: fmt.Sprintf("当前已接入 %d 份多语种增强模板。", total),
				},
				priority: 99,
			})
		}

		if total, err := publishedTemplateCount(ctx, s.templateRepo, "encoding_evasion"); err != nil {
			return nil, err
		} else if total > 0 {
			candidates = append(candidates, welcomeCandidate{
				WelcomeCapability: WelcomeCapability{
					ID:          "resource-template-encoding-evasion",
					Label:       "编码变形对抗",
					Prompt:      "请设计一轮覆盖编码、变形和混淆表达的安全评估，重点检查规避型攻击风险。",
					Tone:        "engine",
					SourceKind:  "resource",
					SourceType:  "encoding_evasion_template",
					Description: fmt.Sprintf("当前已接入 %d 份编码变形增强模板。", total),
				},
				priority: 97,
			})
		}

		if total, err := publishedTemplateCount(ctx, s.templateRepo, ""); err != nil {
			return nil, err
		} else if total > 0 {
			candidates = append(candidates, welcomeCandidate{
				WelcomeCapability: WelcomeCapability{
					ID:          "resource-template-overview",
					Label:       "策略编排能力",
					Prompt:      "请结合当前策略编排能力，为目标应用生成一轮系统化安全评估方案。",
					Tone:        "engine",
					SourceKind:  "resource",
					SourceType:  "template_overview",
					Description: fmt.Sprintf("当前可用于编排的模板能力共有 %d 份。", total),
				},
				priority: 84,
			})
		}
	}

	if s.sampleRepo != nil {
		if _, total, err := s.sampleRepo.ListPublished(ctx, "", 1, 0); err != nil {
			return nil, fmt.Errorf("list published samples: %w", err)
		} else if total > 0 {
			candidates = append(candidates, welcomeCandidate{
				WelcomeCapability: WelcomeCapability{
					ID:          "resource-sample-overview",
					Label:       "场景化攻击题库",
					Prompt:      "请基于当前场景化攻击题库，为目标应用制定一轮安全评估方案。",
					Tone:        "attack",
					SourceKind:  "resource",
					SourceType:  "sample_overview",
					Description: fmt.Sprintf("当前可用的场景化攻击样本共有 %d 份。", total),
				},
				priority: 90,
			})
		}
	}

	if s.composedAttackRepo != nil {
		if _, total, err := s.composedAttackRepo.ListPublished(ctx, "", 1, 0); err != nil {
			return nil, fmt.Errorf("list published composed attacks: %w", err)
		} else if total > 0 {
			candidates = append(candidates, welcomeCandidate{
				WelcomeCapability: WelcomeCapability{
					ID:          "resource-composed-attack-overview",
					Label:       "即用型攻击链",
					Prompt:      "请基于当前可直接执行的攻击链，对目标应用开展一轮安全评估。",
					Tone:        "attack",
					SourceKind:  "resource",
					SourceType:  "composed_attack_overview",
					Description: fmt.Sprintf("当前可直接执行的攻击链共有 %d 份。", total),
				},
				priority: 88,
			})
		}
	}

	sort.SliceStable(candidates, func(i, j int) bool {
		return candidates[i].priority > candidates[j].priority
	})
	return uniqueWelcomeCandidates(candidates), nil
}

func assembleWelcomeCapabilities(limit int, skills []welcomeCandidate, mcps []welcomeCandidate, resources []welcomeCandidate) []welcomeCandidate {
	if limit <= 0 {
		return nil
	}

	chosen := make([]welcomeCandidate, 0, limit)
	seen := make(map[string]struct{})

	chosen = appendLimitedUnique(chosen, seen, skills, minInt(limit, 2))
	chosen = appendLimitedUnique(chosen, seen, mcps, minInt(limit-len(chosen), 2))
	chosen = appendLimitedUnique(chosen, seen, resources, minInt(limit-len(chosen), 2))

	remaining := make([]welcomeCandidate, 0, len(skills)+len(mcps)+len(resources))
	remaining = appendRemainingUnique(remaining, seen, skills)
	remaining = appendRemainingUnique(remaining, seen, mcps)
	remaining = appendRemainingUnique(remaining, seen, resources)

	sort.SliceStable(remaining, func(i, j int) bool {
		return remaining[i].priority > remaining[j].priority
	})

	for _, item := range remaining {
		if len(chosen) >= limit {
			break
		}
		key := welcomeDedupKey(item.WelcomeCapability)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		chosen = append(chosen, item)
	}

	if len(chosen) > limit {
		return chosen[:limit]
	}
	return chosen
}

func appendLimitedUnique(dst []welcomeCandidate, seen map[string]struct{}, src []welcomeCandidate, count int) []welcomeCandidate {
	for _, item := range src {
		if count <= 0 {
			break
		}
		key := welcomeDedupKey(item.WelcomeCapability)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		dst = append(dst, item)
		count--
	}
	return dst
}

func appendRemainingUnique(dst []welcomeCandidate, seen map[string]struct{}, src []welcomeCandidate) []welcomeCandidate {
	for _, item := range src {
		key := welcomeDedupKey(item.WelcomeCapability)
		if _, ok := seen[key]; ok {
			continue
		}
		dst = append(dst, item)
	}
	return dst
}

func uniqueWelcomeCandidates(items []welcomeCandidate) []welcomeCandidate {
	result := make([]welcomeCandidate, 0, len(items))
	seen := make(map[string]struct{})
	for _, item := range items {
		key := welcomeDedupKey(item.WelcomeCapability)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, item)
	}
	return result
}

func buildGeneratorSkillCapability(bundle skillpkg.SkillBundle) welcomeCandidate {
	skillName := ""
	skillDescription := ""
	skillCapability := ""
	skillID := ""
	versionName := ""
	versionSummary := ""
	if bundle.Skill != nil {
		skillName = bundle.Skill.Name
		skillDescription = bundle.Skill.Description
		skillCapability = bundle.Skill.CapabilityProfile
		skillID = bundle.Skill.ID.String()
	}
	if bundle.Version != nil {
		versionName = bundle.Version.DisplayName
		versionSummary = bundle.Version.Summary
	}
	label := friendlyCapabilityLabel(
		skillName,
		versionName,
		skillDescription,
		skillCapability,
		bundle.Manifest.Description,
	)
	prompt := generatorSkillPrompt(label, bundle)
	return welcomeCandidate{
		WelcomeCapability: WelcomeCapability{
			ID:          "skill-generator-" + skillID,
			Label:       label,
			Prompt:      prompt,
			Tone:        toneForKeyword(label + " " + skillCapability + " " + skillDescription),
			SourceKind:  "skill",
			SourceType:  "generator_skill",
			Description: firstNonEmptyTrimmed(bundle.Manifest.Description, versionSummary, skillDescription),
		},
		priority: 115,
	}
}

func buildInteractiveSkillCapability(item skillpkg.LaunchSkillCandidate) welcomeCandidate {
	label := friendlyCapabilityLabel(item.SkillName, item.Description, item.PlannerSummary, item.CapabilityProfile)
	prompt := interactiveSkillPrompt(label, item)
	return welcomeCandidate{
		WelcomeCapability: WelcomeCapability{
			ID:          "skill-interactive-" + item.SkillID.String(),
			Label:       label,
			Prompt:      prompt,
			Tone:        "tool",
			SourceKind:  "skill",
			SourceType:  "interactive_web_skill",
			Description: firstNonEmptyTrimmed(item.PlannerSummary, item.Description, item.CapabilityProfile),
		},
		priority: 104,
	}
}

func buildMCPCapability(item model.ExternalMCPServer) welcomeCandidate {
	label := friendlyMCPLabel(item)
	prompt := mcpPrompt(label, item)
	return welcomeCandidate{
		WelcomeCapability: WelcomeCapability{
			ID:          "mcp-" + item.ID.String(),
			Label:       label,
			Prompt:      prompt,
			Tone:        toneForKeyword(label + " " + item.Description + " " + item.SkillPrompt),
			SourceKind:  "mcp",
			SourceType:  "external_mcp_server",
			Description: firstNonEmptyTrimmed(item.Description, item.SkillPrompt),
		},
		priority: 110,
	}
}

func friendlyMCPLabel(item model.ExternalMCPServer) string {
	joined := strings.ToLower(strings.Join([]string{item.Name, item.Namespace, item.Description, item.SkillPrompt}, " "))
	switch {
	case welcomeContainsAny(joined, "文言文", "古文", "rewrite", "ccbos", "改写"):
		return "文言文改写引擎"
	case welcomeContainsAny(joined, "translate", "翻译", "multilingual", "多语言", "多语种"):
		return "多语种增强引擎"
	case welcomeContainsAny(joined, "encoding", "加密", "编码", "混淆"):
		return "编码变形引擎"
	}
	return friendlyCapabilityLabel(item.Name, item.Namespace, item.Description, item.SkillPrompt)
}

func generatorSkillPrompt(label string, bundle skillpkg.SkillBundle) string {
	skillName := ""
	skillDescription := ""
	skillCapability := ""
	if bundle.Skill != nil {
		skillName = bundle.Skill.Name
		skillDescription = bundle.Skill.Description
		skillCapability = bundle.Skill.CapabilityProfile
	}
	combined := strings.ToLower(strings.Join([]string{
		label,
		skillName,
		skillDescription,
		skillCapability,
		bundle.Manifest.Description,
	}, " "))
	if welcomeContainsAny(combined, "文言文", "古文", "rewrite", "rewriter", "改写") {
		return "请使用文言文改写能力，为目标应用设计一轮越狱安全评估。"
	}
	if welcomeContainsAny(combined, "合规", "compliance") {
		return "请使用当前合规评估能力，对目标应用开展一轮内容安全与合规测试。"
	}
	if welcomeContainsAny(combined, "agent", "智能体") {
		return "请围绕 Agent 安全场景，利用当前能力为目标应用设计一轮安全评估。"
	}
	return fmt.Sprintf("请使用「%s」能力，为目标应用制定一轮安全评估方案。", label)
}

func interactiveSkillPrompt(label string, item skillpkg.LaunchSkillCandidate) string {
	combined := strings.ToLower(strings.Join([]string{
		label,
		item.SkillName,
		item.Description,
		item.PlannerSummary,
		item.CapabilityProfile,
	}, " "))
	if welcomeContainsAny(combined, "可视", "visual", "看板", "dashboard") {
		return fmt.Sprintf("请打开「%s」页面工具，帮助我查看当前风险态势。", label)
	}
	return fmt.Sprintf("请打开「%s」页面工具，帮助我完成下一步评估。", label)
}

func mcpPrompt(label string, item model.ExternalMCPServer) string {
	combined := strings.ToLower(strings.Join([]string{
		label,
		item.Name,
		item.Namespace,
		item.Description,
		item.SkillPrompt,
	}, " "))
	if welcomeContainsAny(combined, "文言文", "古文", "rewrite", "改写") {
		return "请使用文言文改写引擎，设计一轮文言文绕过安全评估。"
	}
	if welcomeContainsAny(combined, "translate", "翻译", "multilingual", "多语言") {
		return fmt.Sprintf("请结合「%s」能力，设计一轮覆盖多语种输入的安全评估。", label)
	}
	return fmt.Sprintf("请结合「%s」能力，为目标应用生成一轮安全评估方案。", label)
}

func friendlyCapabilityLabel(parts ...string) string {
	joined := strings.ToLower(strings.Join(parts, " "))
	switch {
	case welcomeContainsAny(joined, "文言文", "古文", "rewrite", "ccbos", "改写"):
		return "文言文绕过测试"
	case welcomeContainsAny(joined, "compliance", "合规"):
		return "内容合规检查"
	case welcomeContainsAny(joined, "prompt injection", "注入", "injection"):
		return "提示注入检验"
	case welcomeContainsAny(joined, "agent", "智能体"):
		return "Agent 安全评估"
	case welcomeContainsAny(joined, "visual", "dashboard", "看板", "可视"):
		return "风险态势看板"
	case welcomeContainsAny(joined, "multilingual", "多语言", "多语种"):
		return "多语种安全测试"
	case welcomeContainsAny(joined, "encoding", "加密", "编码", "混淆"):
		return "编码变形对抗"
	}

	for _, raw := range parts {
		if label := cleanupCapabilityLabel(raw); label != "" {
			return label
		}
	}
	return "智能评估能力"
}

func cleanupCapabilityLabel(raw string) string {
	value := strings.TrimSpace(raw)
	if value == "" {
		return ""
	}

	replacer := strings.NewReplacer("_", " ", "-", " ", "  ", " ")
	value = replacer.Replace(value)
	for _, suffix := range []string{
		" skill",
		" Skill",
		" mcp",
		" MCP",
		" server",
		" Server",
		" engine",
		" Engine",
		" 服务",
		" 能力",
		" 工具",
		" 平台",
	} {
		value = strings.TrimSuffix(value, suffix)
	}
	value = strings.TrimSpace(value)
	value = strings.TrimFunc(value, func(r rune) bool {
		return unicode.IsSpace(r) || r == ':' || r == '：' || r == '|' || r == ',' || r == '，'
	})
	if value == "" {
		return ""
	}
	return clipRunes(value, 14)
}

func publishedTemplateCount(ctx context.Context, repo *repository.TemplateRepository, subType string) (int, error) {
	_, total, err := repo.ListPublished(ctx, subType, 1, 0)
	if err != nil {
		if subType == "" {
			return 0, fmt.Errorf("list published templates: %w", err)
		}
		return 0, fmt.Errorf("list published templates for subtype %s: %w", subType, err)
	}
	return total, nil
}

func externalServerVisible(item model.ExternalMCPServer) bool {
	if !item.Enabled {
		return false
	}
	status := strings.ToLower(strings.TrimSpace(item.Status))
	if strings.Contains(status, "disabled") || strings.Contains(status, "error") || strings.Contains(status, "fail") {
		return false
	}
	return true
}

func welcomeDedupKey(item WelcomeCapability) string {
	return strings.ToLower(strings.TrimSpace(item.Label + "|" + item.SourceKind + "|" + item.SourceType))
}

func toneForKeyword(value string) string {
	normalized := strings.ToLower(strings.TrimSpace(value))
	switch {
	case welcomeContainsAny(normalized, "合规", "compliance", "治理", "policy"):
		return "governance"
	case welcomeContainsAny(normalized, "文言文", "多语种", "multilingual", "rewrite", "改写", "编码", "加密", "混淆"):
		return "engine"
	case welcomeContainsAny(normalized, "tool", "页面", "可视", "dashboard", "mcp", "集成"):
		return "tool"
	default:
		return "attack"
	}
}

func firstNonEmptyTrimmed(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func welcomeContainsAny(value string, terms ...string) bool {
	for _, term := range terms {
		if strings.Contains(value, strings.ToLower(term)) {
			return true
		}
	}
	return false
}

func clipRunes(value string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(strings.TrimSpace(value))
	if len(runes) <= max {
		return string(runes)
	}
	return string(runes[:max])
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
