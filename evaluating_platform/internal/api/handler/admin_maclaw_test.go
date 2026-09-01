package handler

import (
	"testing"

	"evaluating_platform/internal/maclaw"
)

func TestNormalizeRuntimeConfigLocalhostURLsForDockerRuntime(t *testing.T) {
	cfg := maclaw.RuntimeAppConfig{
		MaclawLLMUrl: "http://localhost:48760/v1",
		MaclawLLMProviders: []maclaw.RuntimeLLMProvider{
			{Name: "default", URL: "http://127.0.0.1:48760/v1", Model: "gpt-5.5"},
		},
	}

	normalizeRuntimeConfigLocalhostURLs(&cfg)

	if cfg.MaclawLLMUrl != "http://host.docker.internal:48760/v1" {
		t.Fatalf("maclaw_llm_url = %q", cfg.MaclawLLMUrl)
	}
	if cfg.MaclawLLMProviders[0].URL != "http://host.docker.internal:48760/v1" {
		t.Fatalf("provider url = %q", cfg.MaclawLLMProviders[0].URL)
	}
}

func TestNormalizeRuntimeConfigLocalhostURLsPreservesLocalRuntime(t *testing.T) {
	cfg := maclaw.RuntimeAppConfig{
		MaclawLLMUrl: "http://localhost:48760/v1",
		MaclawLLMProviders: []maclaw.RuntimeLLMProvider{
			{Name: "default", URL: "http://127.0.0.1:48760/v1", Model: "gpt-5.5"},
		},
	}

	normalizeRuntimeConfigLocalhostURLsForRuntimeMode(&cfg, "local")

	if cfg.MaclawLLMUrl != "http://localhost:48760/v1" {
		t.Fatalf("maclaw_llm_url = %q", cfg.MaclawLLMUrl)
	}
	if cfg.MaclawLLMProviders[0].URL != "http://127.0.0.1:48760/v1" {
		t.Fatalf("provider url = %q", cfg.MaclawLLMProviders[0].URL)
	}
}

func TestMergeAccountRuntimeConfigSecretsPreservesMaskedAndEmptySecrets(t *testing.T) {
	current := &maclaw.RuntimeUserConfig{AppConfig: maclaw.RuntimeAppConfig{
		MaclawLLMKey: "sk-current-global",
		MaclawLLMProviders: []maclaw.RuntimeLLMProvider{{
			Name:             "openai-prod",
			URL:              "https://api.example/v1",
			Key:              "sk-current-provider",
			OAuthAccessToken: "oauth-current",
			RefreshToken:     "refresh-current",
			Model:            "gpt-old",
			ContextLength:    128000,
			SupportsVision:   true,
		}},
		MaclawLLMCurrentProvider: "openai-prod",
	}}
	next := maclaw.RuntimeAppConfig{
		MaclawLLMKey: "******",
		MaclawLLMProviders: []maclaw.RuntimeLLMProvider{{
			Name:             "openai-prod",
			URL:              "https://api.example/v1",
			Key:              "******",
			OAuthAccessToken: "",
			RefreshToken:     "******",
			Model:            "gpt-new",
			ContextLength:    256000,
		}},
		MaclawLLMCurrentProvider: "openai-prod",
	}

	got := mergeAccountRuntimeConfigSecrets(current, next)

	if got.MaclawLLMKey != "sk-current-global" {
		t.Fatalf("global key = %q, want preserved", got.MaclawLLMKey)
	}
	provider := got.MaclawLLMProviders[0]
	if provider.Key != "sk-current-provider" {
		t.Fatalf("provider key = %q, want preserved", provider.Key)
	}
	if provider.OAuthAccessToken != "oauth-current" {
		t.Fatalf("oauth token = %q, want preserved", provider.OAuthAccessToken)
	}
	if provider.RefreshToken != "refresh-current" {
		t.Fatalf("refresh token = %q, want preserved", provider.RefreshToken)
	}
	if provider.Model != "gpt-new" {
		t.Fatalf("provider model = %q, want submitted non-secret value", provider.Model)
	}
}
