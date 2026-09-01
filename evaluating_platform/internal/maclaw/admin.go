package maclaw

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type AdminConfig struct {
	BaseURL        string
	AdminSecret    string
	TimeoutSeconds int
}

type AdminClient struct {
	baseURL     string
	adminSecret string
	httpClient  *http.Client
}

type AdminCreateTenantInput struct {
	Name                   string `json:"name"`
	DeleteProtected        bool   `json:"delete_protected,omitempty"`
	DeleteProtectionReason string `json:"delete_protection_reason,omitempty"`
}

type AdminTenant struct {
	ID   string `json:"id"`
	Name string `json:"name,omitempty"`
}

type AdminCreateUserInput struct {
	Name                   string `json:"name"`
	Email                  string `json:"email,omitempty"`
	DeleteProtected        bool   `json:"delete_protected,omitempty"`
	DeleteProtectionReason string `json:"delete_protection_reason,omitempty"`
}

type AdminUser struct {
	ID       string `json:"id"`
	TenantID string `json:"tenant_id,omitempty"`
	Name     string `json:"name,omitempty"`
	Email    string `json:"email,omitempty"`
}

type AdminCreateCredentialInput struct {
	Name string `json:"name"`
}

type AdminCredential struct {
	ID        string `json:"id"`
	TenantID  string `json:"tenant_id,omitempty"`
	UserID    string `json:"user_id,omitempty"`
	APIKey    string `json:"api_key,omitempty"`
	APISecret string `json:"api_secret,omitempty"`
}

type TokenIssueInput struct {
	APIKey    string `json:"api_key,omitempty"`
	APISecret string `json:"api_secret,omitempty"`
}

type TokenIssueResult struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type RuntimeInstanceInput struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type RuntimeInstance struct {
	ID       string            `json:"id"`
	TenantID string            `json:"tenant_id,omitempty"`
	UserID   string            `json:"user_id,omitempty"`
	Name     string            `json:"name,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

func NewAdminClient(cfg AdminConfig) (*AdminClient, error) {
	baseURL := strings.TrimRight(strings.TrimSpace(cfg.BaseURL), "/")
	adminSecret := strings.TrimSpace(cfg.AdminSecret)
	if baseURL == "" || adminSecret == "" {
		return nil, ErrNotConfigured
	}
	timeout := time.Duration(cfg.TimeoutSeconds) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &AdminClient{
		baseURL:     baseURL,
		adminSecret: adminSecret,
		httpClient:  &http.Client{Timeout: timeout},
	}, nil
}

func (c *AdminClient) CreateTenant(ctx context.Context, in AdminCreateTenantInput) (*AdminTenant, error) {
	var out AdminTenant
	if err := c.doAdminJSON(ctx, http.MethodPost, "/api/v1/admin/tenants", in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *AdminClient) CreateUser(ctx context.Context, tenantID string, in AdminCreateUserInput) (*AdminUser, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, errors.New("tenant id is required")
	}
	var out AdminUser
	path := "/api/v1/admin/tenants/" + url.PathEscape(tenantID) + "/users"
	if err := c.doAdminJSON(ctx, http.MethodPost, path, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *AdminClient) CreateCredential(ctx context.Context, tenantID, userID string, in AdminCreateCredentialInput) (*AdminCredential, error) {
	tenantID = strings.TrimSpace(tenantID)
	userID = strings.TrimSpace(userID)
	if tenantID == "" || userID == "" {
		return nil, errors.New("tenant id and user id are required")
	}
	var out AdminCredential
	path := "/api/v1/admin/tenants/" + url.PathEscape(tenantID) + "/users/" + url.PathEscape(userID) + "/credentials"
	if err := c.doAdminJSON(ctx, http.MethodPost, path, in, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *AdminClient) IssueToken(ctx context.Context, in TokenIssueInput) (*TokenIssueResult, error) {
	var out TokenIssueResult
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/auth/token", in, &out, ""); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *AdminClient) CreateInstance(ctx context.Context, accessToken string, in RuntimeInstanceInput) (*RuntimeInstance, error) {
	var out RuntimeInstance
	if err := c.doJSON(ctx, http.MethodPost, "/api/v1/instances", in, &out, strings.TrimSpace(accessToken)); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *AdminClient) UpdateUserConfig(ctx context.Context, accessToken string, in RuntimeAppConfig) (*RuntimeUserConfig, error) {
	var out RuntimeUserConfig
	if err := c.doJSON(ctx, http.MethodPut, "/api/v1/config", in, &out, strings.TrimSpace(accessToken)); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *AdminClient) doAdminJSON(ctx context.Context, method, path string, in, out any) error {
	return c.doJSON(ctx, method, path, in, out, "")
}

func (c *AdminClient) doJSON(ctx context.Context, method, path string, in, out any, bearerToken string) error {
	if c == nil || c.baseURL == "" {
		return ErrNotConfigured
	}
	var body io.Reader
	if in != nil {
		buf, err := json.Marshal(in)
		if err != nil {
			return fmt.Errorf("marshal maclaw admin request: %w", err)
		}
		body = bytes.NewReader(buf)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("build maclaw admin request: %w", err)
	}
	if in != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+bearerToken)
	} else if c.adminSecret != "" && strings.HasPrefix(path, "/api/v1/admin/") {
		req.Header.Set("X-MaClaw-Admin-Secret", c.adminSecret)
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("call maclaw admin: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		message := strings.TrimSpace(string(data))
		return NewUpstreamError(method, path, resp.StatusCode, message)
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode maclaw admin response: %w", err)
	}
	return nil
}
