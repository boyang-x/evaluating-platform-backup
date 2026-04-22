package skill

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/google/uuid"
)

type LaunchDocument struct {
	SkillID       uuid.UUID `json:"skill_id"`
	SkillName     string    `json:"skill_name"`
	SkillSlug     string    `json:"skill_slug"`
	VersionID     uuid.UUID `json:"version_id"`
	Version       string    `json:"version"`
	DocumentTitle string    `json:"document_title"`
	HTML          string    `json:"html"`
	LaunchMode    string    `json:"launch_mode"`
}

type LaunchSkillCandidate struct {
	SkillID           uuid.UUID `json:"skill_id"`
	SkillName         string    `json:"skill_name"`
	SkillSlug         string    `json:"skill_slug"`
	Description       string    `json:"description"`
	PlannerSummary    string    `json:"planner_summary"`
	IntentExamples    []string  `json:"intent_examples"`
	DeliveryMode      string    `json:"delivery_mode"`
	CapabilityProfile string    `json:"capability_profile"`
}

type interactiveLaunchValidation struct {
	SkillType         string   `json:"skill_type"`
	Entrypoint        string   `json:"entrypoint"`
	DocumentTitle     string   `json:"document_title,omitempty"`
	DeliveryMode      string   `json:"delivery_mode,omitempty"`
	AssetRefCount     int      `json:"asset_ref_count"`
	RemoteAssetRefs   []string `json:"remote_asset_refs,omitempty"`
	RelativeAssetRefs []string `json:"relative_asset_refs,omitempty"`
}

var launchHTMLAssetRefRE = regexp.MustCompile(`(?is)<(?:script|img|iframe|source)\b[^>]*\bsrc\s*=\s*["']([^"']+)["']|<(?:link)\b[^>]*\bhref\s*=\s*["']([^"']+)["']`)
var launchHTMLTitleRE = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)

func buildSkillVersionMetadata(manifest Manifest) map[string]interface{} {
	metadata := map[string]interface{}{}
	for key, value := range manifest.Metadata {
		metadata[key] = value
	}
	metadata["skill_name"] = strings.TrimSpace(manifest.Name)
	metadata["skill_type"] = strings.TrimSpace(manifest.SkillType)
	metadata["config_schema"] = buildConfigSchemaEnvelope(manifest)
	if strings.TrimSpace(manifest.CapabilityProfile) != "" {
		metadata["capability_profile"] = strings.TrimSpace(manifest.CapabilityProfile)
	}
	if manifest.SkillType == "interactive_web_skill" {
		metadata["planner"] = manifest.Planner
		metadata["web"] = manifest.Web
	}
	return metadata
}

func metadataPlanner(metadata map[string]interface{}) PlannerManifest {
	var planner PlannerManifest
	if metadata == nil {
		planner.DeliveryMode = "open_url"
		planner.EnterpriseVisible = boolPtr(true)
		return planner
	}
	if raw, ok := metadata["planner"]; ok {
		data, _ := json.Marshal(raw)
		_ = json.Unmarshal(data, &planner)
	}
	planner.DeliveryMode = strings.TrimSpace(defaultString(planner.DeliveryMode, "open_url"))
	if planner.EnterpriseVisible == nil {
		planner.EnterpriseVisible = boolPtr(true)
	}
	return planner
}

func metadataWeb(metadata map[string]interface{}) WebManifest {
	var web WebManifest
	if metadata == nil {
		return web
	}
	if raw, ok := metadata["web"]; ok {
		data, _ := json.Marshal(raw)
		_ = json.Unmarshal(data, &web)
	}
	web.Entrypoint = strings.TrimSpace(web.Entrypoint)
	return web
}

func plannerVisible(planner PlannerManifest) bool {
	return planner.EnterpriseVisible == nil || *planner.EnterpriseVisible
}

func prepareInteractiveLaunch(pkg *ImportedPackage) (string, string, json.RawMessage, error) {
	if pkg == nil {
		return "", "", json.RawMessage(`{}`), fmt.Errorf("interactive_web_skill package is empty")
	}
	if pkg.Manifest.SkillType != "interactive_web_skill" {
		return "", "", json.RawMessage(`{}`), fmt.Errorf("skill package is not interactive_web_skill")
	}
	entrypoint := strings.TrimSpace(pkg.Manifest.Web.Entrypoint)
	if entrypoint == "" {
		entrypoint = strings.TrimSpace(pkg.Manifest.Execution.Entrypoint)
	}
	if entrypoint == "" {
		return "", "", json.RawMessage(`{}`), fmt.Errorf("interactive_web_skill is missing web.entrypoint")
	}

	data, ok := pkg.Files[entrypoint]
	if !ok {
		return "", "", json.RawMessage(`{}`), fmt.Errorf("interactive_web_skill entrypoint not found in archive: %s", entrypoint)
	}

	htmlText := strings.TrimPrefix(string(data), "\uFEFF")
	relativeRefs, remoteRefs := collectInteractiveAssetRefs(htmlText)
	title := extractLaunchHTMLTitle(htmlText, pkg.Manifest.DisplayName)
	report := mustJSON(interactiveLaunchValidation{
		SkillType:         pkg.Manifest.SkillType,
		Entrypoint:        entrypoint,
		DocumentTitle:     title,
		DeliveryMode:      pkg.Manifest.Planner.DeliveryMode,
		AssetRefCount:     len(relativeRefs) + len(remoteRefs),
		RemoteAssetRefs:   remoteRefs,
		RelativeAssetRefs: relativeRefs,
	}, `{}`)

	if len(relativeRefs) > 0 {
		return "", "", report, fmt.Errorf("interactive_web_skill entrypoint must be self-contained or use absolute asset URLs: %s", strings.Join(relativeRefs, ", "))
	}

	return title, htmlText, report, nil
}

func collectInteractiveAssetRefs(htmlText string) ([]string, []string) {
	relativeRefs := make([]string, 0)
	remoteRefs := make([]string, 0)
	matches := launchHTMLAssetRefRE.FindAllStringSubmatch(htmlText, -1)
	for _, match := range matches {
		ref := ""
		for i := 1; i < len(match); i++ {
			if strings.TrimSpace(match[i]) != "" {
				ref = strings.TrimSpace(match[i])
				break
			}
		}
		if ref == "" {
			continue
		}
		if isRemoteLaunchAssetRef(ref) {
			remoteRefs = append(remoteRefs, ref)
			continue
		}
		relativeRefs = append(relativeRefs, ref)
	}
	sort.Strings(relativeRefs)
	sort.Strings(remoteRefs)
	return uniqueNonEmptyStrings(relativeRefs), uniqueNonEmptyStrings(remoteRefs)
}

func isRemoteLaunchAssetRef(ref string) bool {
	lower := strings.ToLower(strings.TrimSpace(ref))
	switch {
	case lower == "":
		return false
	case strings.HasPrefix(lower, "https://"),
		strings.HasPrefix(lower, "http://"),
		strings.HasPrefix(lower, "//"),
		strings.HasPrefix(lower, "data:"),
		strings.HasPrefix(lower, "blob:"),
		strings.HasPrefix(lower, "about:"):
		return true
	default:
		return false
	}
}

func extractLaunchHTMLTitle(htmlText, fallback string) string {
	matches := launchHTMLTitleRE.FindStringSubmatch(htmlText)
	if len(matches) < 2 {
		return strings.TrimSpace(defaultString(fallback, "Interactive Skill"))
	}
	title := strings.TrimSpace(stripHTMLWhitespace(matches[1]))
	if title == "" {
		return strings.TrimSpace(defaultString(fallback, "Interactive Skill"))
	}
	return title
}

func stripHTMLWhitespace(value string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
}

func uniqueNonEmptyStrings(items []string) []string {
	if len(items) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(items))
	result := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		if _, ok := seen[trimmed]; ok {
			continue
		}
		seen[trimmed] = struct{}{}
		result = append(result, trimmed)
	}
	return result
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
