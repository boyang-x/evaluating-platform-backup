package maclaw

import (
	"context"
	"errors"
	"net/http"
	"strings"
)

const (
	RuntimeCapabilityProfileShadow        = "shadow"
	RuntimeCapabilityProfileNativeCatalog = "native-catalog"
)

type RuntimeCapabilities struct {
	Profile                     string `json:"profile,omitempty"`
	AutonomousSkillExecution    bool   `json:"autonomous_skill_execution,omitempty"`
	NativeResourceCatalog       bool   `json:"native_resource_catalog,omitempty"`
	NativeSkillCatalog          bool   `json:"native_skill_catalog,omitempty"`
	NativeCrossTenantReferences bool   `json:"native_cross_tenant_references,omitempty"`
	NativeCatalogGrantSync      bool   `json:"native_catalog_grant_sync,omitempty"`
	RedteamDomainProfile        bool   `json:"redteam_domain_profile,omitempty"`
	CapabilityCardContext       bool   `json:"capability_card_context,omitempty"`
	StructuredRedteamPlanner    bool   `json:"structured_redteam_planner,omitempty"`
	ReportSchemaV1              bool   `json:"report_schema_v1,omitempty"`
	JobRecovery                 bool   `json:"job_recovery,omitempty"`
	ReportExport                bool   `json:"report_export,omitempty"`
}

func ShadowRuntimeCapabilities() RuntimeCapabilities {
	return RuntimeCapabilities{
		Profile:        RuntimeCapabilityProfileShadow,
		ReportSchemaV1: true,
		JobRecovery:    true,
		ReportExport:   true,
	}
}

func NativeCatalogRuntimeCapabilities() RuntimeCapabilities {
	return RuntimeCapabilities{
		Profile:                     RuntimeCapabilityProfileNativeCatalog,
		AutonomousSkillExecution:    true,
		NativeResourceCatalog:       true,
		NativeSkillCatalog:          true,
		NativeCrossTenantReferences: true,
		RedteamDomainProfile:        true,
		CapabilityCardContext:       true,
		StructuredRedteamPlanner:    true,
		ReportSchemaV1:              true,
		JobRecovery:                 true,
		ReportExport:                true,
	}
}

func RuntimeCapabilitiesFromProfile(profile string) RuntimeCapabilities {
	switch strings.ToLower(strings.TrimSpace(profile)) {
	case RuntimeCapabilityProfileNativeCatalog, "native", "grant", "grants":
		return NativeCatalogRuntimeCapabilities()
	default:
		return ShadowRuntimeCapabilities()
	}
}

func (c RuntimeCapabilities) normalized() RuntimeCapabilities {
	if strings.TrimSpace(c.Profile) == "" {
		c.Profile = RuntimeCapabilityProfileShadow
	}
	return c
}

func (c RuntimeCapabilities) SupportsNativeResourceProjection() bool {
	return c.NativeResourceCatalog && c.NativeCrossTenantReferences && c.NativeCatalogGrantSync
}

func (c RuntimeCapabilities) SupportsNativeSkillProjection() bool {
	return c.NativeSkillCatalog && c.NativeCrossTenantReferences && c.NativeCatalogGrantSync
}

func (c *Client) GetRuntimeCapabilities(ctx context.Context) (RuntimeCapabilities, error) {
	if !c.Enabled() {
		return ShadowRuntimeCapabilities(), ErrNotConfigured
	}
	var out RuntimeCapabilities
	if err := c.doJSON(ctx, http.MethodGet, "/api/v1/capabilities", nil, &out); err != nil {
		var upstream *UpstreamError
		if errors.As(err, &upstream) && upstream.StatusCode == http.StatusNotFound {
			return ShadowRuntimeCapabilities(), nil
		}
		return ShadowRuntimeCapabilities(), err
	}
	return out.normalized(), nil
}
