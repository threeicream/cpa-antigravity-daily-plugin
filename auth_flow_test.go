package main

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

func TestBuildAuthURL(t *testing.T) {
	raw := buildAuthURL("state_123", loopbackRedirectURI, "test-client-id")
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	if q.Get("state") != "state_123" || q.Get("redirect_uri") != loopbackRedirectURI || q.Get("access_type") != "offline" || q.Get("prompt") != "consent" {
		t.Fatalf("unexpected OAuth query: %#v", q)
	}
	if !strings.Contains(q.Get("scope"), "experimentsandconfigs") {
		t.Fatalf("missing Antigravity scopes: %q", q.Get("scope"))
	}
}

func TestSelectOnboardTier(t *testing.T) {
	data := map[string]any{"allowedTiers": []any{
		map[string]any{"id": "standard-tier"},
		map[string]any{"id": "free-tier", "isDefault": true},
	}}
	if got := selectOnboardTier(data); got != "free-tier" {
		t.Fatalf("selectOnboardTier()=%q", got)
	}
}

func TestExtractProjectIDObjectAndString(t *testing.T) {
	if got := extractProjectID(map[string]any{"cloudaicompanionProject": "project-a"}); got != "project-a" {
		t.Fatalf("string project=%q", got)
	}
	if got := extractProjectID(map[string]any{"cloudaicompanionProject": map[string]any{"id": "project-b"}}); got != "project-b" {
		t.Fatalf("object project=%q", got)
	}
}

func TestDiscoverDailyProjectOnboardsFreeTier(t *testing.T) {
	loadCalls := 0
	onboardCalls := 0
	do := func(_ string, req upstreamRequest) (upstreamResponse, error) {
		switch {
		case strings.HasSuffix(req.URL, ":loadCodeAssist"):
			loadCalls++
			body := `{"paidTier":{"id":"free-tier"},"allowedTiers":[{"id":"free-tier","isDefault":true}]}`
			return upstreamResponse{StatusCode: http.StatusOK, Body: []byte(body)}, nil
		case strings.HasSuffix(req.URL, ":onboardUser"):
			onboardCalls++
			var payload map[string]any
			if err := json.Unmarshal(req.Body, &payload); err != nil {
				t.Fatal(err)
			}
			if payload["tierId"] != "free-tier" {
				t.Fatalf("tierId=%v", payload["tierId"])
			}
			body := `{"done":true,"response":{"cloudaicompanionProject":{"id":"real-project"}}}`
			return upstreamResponse{StatusCode: http.StatusOK, Body: []byte(body)}, nil
		default:
			t.Fatalf("unexpected URL %s", req.URL)
			return upstreamResponse{}, nil
		}
	}
	project, tier, err := discoverDailyProject("callback", "token", do)
	if err != nil {
		t.Fatal(err)
	}
	if project != "real-project" || tier != "free-tier" || loadCalls != 2 || onboardCalls != 1 {
		t.Fatalf("project=%q tier=%q load=%d onboard=%d", project, tier, loadCalls, onboardCalls)
	}
}

func TestBuildCredentialIsNativeAntigravity(t *testing.T) {
	auth, err := buildCredential("user@example.com", "project-123", "free-tier", tokenResponse{
		AccessToken:  "access",
		RefreshToken: "refresh",
		ExpiresIn:    3600,
	}, pluginConfig{ClientID: "test-client-id", ClientSecret: "test-client-secret"})
	if err != nil {
		t.Fatal(err)
	}
	if auth.Provider != "antigravity" || auth.FileName != "antigravity-user@example.com.json" {
		t.Fatalf("unexpected auth identity: %#v", auth)
	}
	var stored map[string]any
	if err := json.Unmarshal(auth.StorageJSON, &stored); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"client_id", "client_secret", "token", "access_token", "refresh_token", "project_id", "email", "type"} {
		if stored[key] == nil || stored[key] == "" {
			t.Fatalf("missing %s in storage", key)
		}
	}
}

func TestRequire2xxLimitsErrorBody(t *testing.T) {
	err := require2xx("test", upstreamResponse{StatusCode: 500, Body: []byte(`{"error":{"message":"safe detail"},"token":"must-not-leak"}`)})
	if err == nil || len(err.Error()) > 550 {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(err.Error(), "must-not-leak") || !strings.Contains(err.Error(), "safe detail") {
		t.Fatalf("unsafe error: %v", err)
	}
}

func TestValidOAuthState(t *testing.T) {
	for _, state := range []string{"abc_DEF-123.xyz", randomState()} {
		if !validOAuthState(state) {
			t.Fatalf("valid state rejected: %q", state)
		}
	}
	for _, state := range []string{"", "../escape", "bad/state", strings.Repeat("a", 129)} {
		if validOAuthState(state) {
			t.Fatalf("invalid state accepted: %q", state)
		}
	}
}

func TestLoginPageFollowsCPATheme(t *testing.T) {
	for _, marker := range []string{"cli-proxy-theme", "data-theme", "prefers-color-scheme: dark"} {
		if !strings.Contains(loginPageHTML, marker) {
			t.Fatalf("login page missing CPA theme marker %q", marker)
		}
	}
}

func TestConfigureLoadsOAuthClient(t *testing.T) {
	raw, err := json.Marshal(lifecycleRequest{ConfigYAML: []byte("client_id: test-id\nclient_secret: test-secret\n")})
	if err != nil {
		t.Fatal(err)
	}
	if err := configure(raw); err != nil {
		t.Fatal(err)
	}
	cfg, err := requirePluginConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ClientID != "test-id" || cfg.ClientSecret != "test-secret" {
		t.Fatalf("unexpected config: %#v", cfg)
	}
}

func TestConfigureAllowsRegistrationWithoutOAuthClient(t *testing.T) {
	if err := configure(nil); err != nil {
		t.Fatal(err)
	}
	if _, err := requirePluginConfig(); err == nil {
		t.Fatal("missing OAuth client config was accepted")
	}
}
