package maclaw

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"

	"evaluating_platform/internal/model"
)

type GatewaySession struct {
	Client     GatewayClient
	InstanceID string
	Mapping    *model.MaclawAccountMapping
}

type GatewayClient interface {
	EvaluationGateway
	SkillGateway
	RuntimeGateway
	GetRuntimeConfig(context.Context) (*RuntimeUserConfig, error)
	UpdateRuntimeConfig(context.Context, RuntimeAppConfig) (*RuntimeUserConfig, error)
	ValidateRuntimeConfig(context.Context, *RuntimeAppConfig) (*RuntimeConfigValidation, error)
	TestRuntimeConfig(context.Context, *RuntimeAppConfig) (*RuntimeConfigTestResult, error)
	RefreshRuntimeInstanceReadiness(context.Context, string) error
	MaterializeEvaluationResource(context.Context, string) (*EvaluationResourceMaterialization, error)
}

type GatewayProvider interface {
	Enabled() bool
	Resolve(context.Context, RuntimeIdentity) (*GatewaySession, error)
}

type ProvisioningGatewayProvider struct {
	provisioner         *Provisioner
	targetConfigService *TargetConfigService
	artifactService     *RedteamArtifactService
	resourceService     *ResourceStoreService
}

func NewProvisioningGatewayProvider(provisioner *Provisioner) *ProvisioningGatewayProvider {
	return &ProvisioningGatewayProvider{provisioner: provisioner}
}

func (p *ProvisioningGatewayProvider) SetTargetConfigService(service *TargetConfigService) {
	if p != nil {
		p.targetConfigService = service
	}
}

func (p *ProvisioningGatewayProvider) SetArtifactService(service *RedteamArtifactService) {
	if p != nil {
		p.artifactService = service
	}
}

func (p *ProvisioningGatewayProvider) SetResourceStoreService(service *ResourceStoreService) {
	if p != nil {
		p.resourceService = service
	}
}

func (p *ProvisioningGatewayProvider) Enabled() bool {
	return p != nil && p.provisioner != nil
}

func (p *ProvisioningGatewayProvider) Resolve(ctx context.Context, identity RuntimeIdentity) (*GatewaySession, error) {
	if !p.Enabled() {
		return nil, ErrNotConfigured
	}
	userID, err := uuid.Parse(strings.TrimSpace(identity.UserID))
	if err != nil {
		return nil, errors.New("platform user id is required")
	}
	resolveCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), p.resolveTimeout())
	defer cancel()
	runtime, err := p.provisioner.EnsureRuntime(resolveCtx, userID)
	if err != nil {
		return nil, err
	}
	client := runtime.Client
	if runtime.Mapping != nil && p.targetConfigService != nil && p.targetConfigService.Enabled() {
		client = NewPlatformTargetGateway(client, p.targetConfigService, runtime.Mapping.PlatformUserID)
	}
	if runtime.Mapping != nil && p.artifactService != nil && p.artifactService.Enabled() {
		client = NewPlatformArtifactGateway(client, p.artifactService, runtime.Mapping.PlatformUserID)
	}
	if runtime.Mapping != nil && p.resourceService != nil && p.resourceService.Enabled() {
		client = NewPlatformResourceGateway(client, p.resourceService, runtime.Mapping.PlatformUserID, runtime.Mapping.MaclawTenantID)
	}
	return &GatewaySession{Client: client, InstanceID: runtime.InstanceID, Mapping: runtime.Mapping}, nil
}

func (p *ProvisioningGatewayProvider) resolveTimeout() time.Duration {
	timeout := 60 * time.Second
	if p != nil && p.provisioner != nil && p.provisioner.cfg.TimeoutSeconds > 0 {
		configured := time.Duration(p.provisioner.cfg.TimeoutSeconds) * time.Second
		if configured > timeout {
			timeout = configured
		}
	}
	return timeout
}

func clientForGatewayInstance(client GatewayClient, instanceID string) GatewayClient {
	if c, ok := client.(*MaclawSrvClient); ok {
		return c.WithInstanceID(instanceID)
	}
	return client
}

func MCPGatewayFromClient(client GatewayClient) (MCPGateway, bool) {
	if client == nil {
		return nil, false
	}
	if gateway, ok := client.(MCPGateway); ok && gateway != nil {
		return gateway, true
	}
	switch wrapped := client.(type) {
	case *PlatformTargetGateway:
		return MCPGatewayFromClient(wrapped.GatewayClient)
	case *PlatformArtifactGateway:
		return MCPGatewayFromClient(wrapped.GatewayClient)
	case *PlatformResourceGateway:
		return MCPGatewayFromClient(wrapped.GatewayClient)
	default:
		return nil, false
	}
}
