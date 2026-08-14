package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	googleAuthEndpoint     = "https://accounts.google.com/o/oauth2/v2/auth"
	googleTokenEndpoint    = "https://oauth2.googleapis.com/token"
	googleUserInfoEndpoint = "https://www.googleapis.com/oauth2/v2/userinfo?alt=json"
	loopbackRedirectURI    = "http://localhost:51121/oauth-callback"
	loginLifetime          = 5 * time.Minute
)

var googleScopes = []string{
	"https://www.googleapis.com/auth/cloud-platform",
	"https://www.googleapis.com/auth/userinfo.email",
	"https://www.googleapis.com/auth/userinfo.profile",
	"https://www.googleapis.com/auth/cclog",
	"https://www.googleapis.com/auth/experimentsandconfigs",
}

func startLogin(_ authLoginStartRequest) (authLoginStartResponse, error) {
	cfg, err := requirePluginConfig()
	if err != nil {
		return authLoginStartResponse{}, err
	}
	state := randomState()
	return authLoginStartResponse{
		Provider:  providerID,
		URL:       buildAuthURL(state, loopbackRedirectURI, cfg.ClientID),
		State:     state,
		ExpiresAt: time.Now().UTC().Add(loginLifetime),
		Metadata: map[string]any{
			"redirect_uri": loopbackRedirectURI,
		},
	}, nil
}

func randomState() string {
	raw := make([]byte, 24)
	if _, err := rand.Read(raw); err == nil {
		return base64.RawURLEncoding.EncodeToString(raw)
	}
	return fmt.Sprintf("antigravity-%d", time.Now().UnixNano())
}

func buildAuthURL(state, redirectURI, clientID string) string {
	query := url.Values{}
	query.Set("access_type", "offline")
	query.Set("client_id", clientID)
	query.Set("include_granted_scopes", "true")
	query.Set("prompt", "consent")
	query.Set("redirect_uri", redirectURI)
	query.Set("response_type", "code")
	query.Set("scope", strings.Join(googleScopes, " "))
	query.Set("state", state)
	return googleAuthEndpoint + "?" + query.Encode()
}

func pollLogin(req authLoginPollRequest, do httpDoFunc) (authLoginPollResponse, error) {
	cfg, err := requirePluginConfig()
	if err != nil {
		return authLoginPollResponse{Status: "error", Message: err.Error()}, nil
	}
	state := strings.TrimSpace(req.State)
	if !validOAuthState(state) {
		return authLoginPollResponse{Status: "error", Message: "invalid OAuth state"}, nil
	}
	authDir := strings.TrimSpace(req.Host.AuthDir)
	if authDir == "" {
		return authLoginPollResponse{Status: "error", Message: "CPA did not provide AuthDir"}, nil
	}
	waitFile := filepath.Join(authDir, ".oauth-"+providerID+"-"+state+".oauth")
	raw, err := os.ReadFile(waitFile)
	if errors.Is(err, os.ErrNotExist) {
		return authLoginPollResponse{Status: "pending", Message: "waiting for OAuth callback"}, nil
	}
	if err != nil {
		return authLoginPollResponse{}, fmt.Errorf("read OAuth callback: %w", err)
	}
	defer func() { _ = os.Remove(waitFile) }()

	var callback oauthCallback
	if err := json.Unmarshal(raw, &callback); err != nil {
		return authLoginPollResponse{Status: "error", Message: "invalid OAuth callback payload"}, nil
	}
	if callback.State != state {
		return authLoginPollResponse{Status: "error", Message: "OAuth state mismatch"}, nil
	}
	if strings.TrimSpace(callback.Error) != "" {
		return authLoginPollResponse{Status: "error", Message: "Google OAuth failed: " + callback.Error}, nil
	}
	if strings.TrimSpace(callback.Code) == "" {
		return authLoginPollResponse{Status: "error", Message: "OAuth callback did not contain a code"}, nil
	}

	redirectURI, _ := req.Metadata["redirect_uri"].(string)
	if strings.TrimSpace(redirectURI) == "" {
		redirectURI = loopbackRedirectURI
	}
	token, err := exchangeCode(req.HostCallbackID, callback.Code, redirectURI, cfg, do)
	if err != nil {
		return authLoginPollResponse{Status: "error", Message: err.Error()}, nil
	}
	email, err := fetchEmail(req.HostCallbackID, token.AccessToken, do)
	if err != nil {
		return authLoginPollResponse{Status: "error", Message: err.Error()}, nil
	}
	projectID, tier, err := discoverDailyProject(req.HostCallbackID, token.AccessToken, do)
	if err != nil {
		return authLoginPollResponse{Status: "error", Message: err.Error()}, nil
	}
	auth, err := buildCredential(email, projectID, tier, token, cfg)
	if err != nil {
		return authLoginPollResponse{Status: "error", Message: err.Error()}, nil
	}
	return authLoginPollResponse{
		Status:  "success",
		Message: "Antigravity credential created with project_id " + projectID,
		Auth:    auth,
	}, nil
}

func exchangeCode(callbackID, code, redirectURI string, cfg pluginConfig, do httpDoFunc) (tokenResponse, error) {
	form := url.Values{}
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)
	form.Set("code", strings.TrimSpace(code))
	form.Set("grant_type", "authorization_code")
	form.Set("redirect_uri", redirectURI)
	resp, err := do(callbackID, upstreamRequest{
		Method: http.MethodPost,
		URL:    googleTokenEndpoint,
		Headers: map[string][]string{
			"Content-Type": {"application/x-www-form-urlencoded"},
		},
		Body: []byte(form.Encode()),
	})
	if err != nil {
		return tokenResponse{}, fmt.Errorf("exchange OAuth code: %w", err)
	}
	if err := require2xx("OAuth token exchange", resp); err != nil {
		return tokenResponse{}, err
	}
	var token tokenResponse
	if err := json.Unmarshal(resp.Body, &token); err != nil {
		return tokenResponse{}, fmt.Errorf("decode OAuth token response: %w", err)
	}
	if strings.TrimSpace(token.AccessToken) == "" {
		return tokenResponse{}, fmt.Errorf("OAuth token response did not contain access_token")
	}
	if strings.TrimSpace(token.RefreshToken) == "" {
		return tokenResponse{}, fmt.Errorf("OAuth token response did not contain refresh_token; revoke the old grant and try again")
	}
	return token, nil
}

func fetchEmail(callbackID, accessToken string, do httpDoFunc) (string, error) {
	resp, err := do(callbackID, upstreamRequest{
		Method:  http.MethodGet,
		URL:     googleUserInfoEndpoint,
		Headers: bearerHeaders(accessToken),
	})
	if err != nil {
		return "", fmt.Errorf("fetch Google user info: %w", err)
	}
	if err := require2xx("Google user info", resp); err != nil {
		return "", err
	}
	var info userInfo
	if err := json.Unmarshal(resp.Body, &info); err != nil {
		return "", fmt.Errorf("decode Google user info: %w", err)
	}
	email := strings.TrimSpace(info.Email)
	if email == "" {
		return "", fmt.Errorf("Google user info did not contain email")
	}
	return email, nil
}

func buildCredential(email, projectID, tier string, token tokenResponse, cfg pluginConfig) (authData, error) {
	email = strings.TrimSpace(email)
	projectID = strings.TrimSpace(projectID)
	if email == "" || projectID == "" || strings.TrimSpace(token.AccessToken) == "" || strings.TrimSpace(token.RefreshToken) == "" {
		return authData{}, fmt.Errorf("refusing to save incomplete Antigravity credential")
	}
	now := time.Now().UTC()
	expiresIn := token.ExpiresIn
	if expiresIn <= 0 {
		expiresIn = 3600
	}
	expiry := now.Add(time.Duration(expiresIn) * time.Second)
	credential := map[string]any{
		"type":          "antigravity",
		"client_id":     cfg.ClientID,
		"client_secret": cfg.ClientSecret,
		"token":         token.AccessToken,
		"access_token":  token.AccessToken,
		"refresh_token": token.RefreshToken,
		"token_uri":     googleTokenEndpoint,
		"scopes":        append([]string(nil), googleScopes...),
		"project_id":    projectID,
		"email":         email,
		"expires_in":    expiresIn,
		"timestamp":     now.UnixMilli(),
		"expiry":        expiry.Format(time.RFC3339),
		"expired":       expiry.Format(time.RFC3339),
	}
	if strings.TrimSpace(tier) != "" {
		credential["subscription_tier"] = tier
	}
	storage, err := json.Marshal(credential)
	if err != nil {
		return authData{}, fmt.Errorf("encode credential: %w", err)
	}
	fileName := "antigravity-" + sanitizeEmailForFileName(email) + ".json"
	metadata := make(map[string]any, len(credential))
	for key, value := range credential {
		metadata[key] = value
	}
	return authData{
		Provider:    "antigravity",
		ID:          fileName,
		FileName:    fileName,
		Label:       email,
		StorageJSON: storage,
		Metadata:    metadata,
	}, nil
}

func sanitizeEmailForFileName(email string) string {
	email = strings.TrimSpace(email)
	replacer := strings.NewReplacer("/", "_", "\\", "_", "..", "_", "\x00", "_")
	return replacer.Replace(email)
}

func validOAuthState(state string) bool {
	if state == "" || len(state) > 128 {
		return false
	}
	for _, r := range state {
		switch {
		case r >= 'a' && r <= 'z':
		case r >= 'A' && r <= 'Z':
		case r >= '0' && r <= '9':
		case r == '-' || r == '_' || r == '.':
		default:
			return false
		}
	}
	return true
}

func bearerHeaders(accessToken string) map[string][]string {
	return map[string][]string{
		"Authorization": {"Bearer " + accessToken},
		"User-Agent":    {antigravityUserAgent},
	}
}

func require2xx(operation string, resp upstreamResponse) error {
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	detail := safeUpstreamError(resp.Body)
	if detail == "" {
		return fmt.Errorf("%s failed: HTTP %d", operation, resp.StatusCode)
	}
	return fmt.Errorf("%s failed: HTTP %d: %s", operation, resp.StatusCode, detail)
}

func safeUpstreamError(body []byte) string {
	var payload struct {
		Error            any    `json:"error"`
		ErrorDescription string `json:"error_description"`
		Message          string `json:"message"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	detail := strings.TrimSpace(payload.ErrorDescription)
	if detail == "" {
		detail = strings.TrimSpace(payload.Message)
	}
	if detail == "" {
		switch value := payload.Error.(type) {
		case string:
			detail = strings.TrimSpace(value)
		case map[string]any:
			if message, ok := value["message"].(string); ok {
				detail = strings.TrimSpace(message)
			}
		}
	}
	if len(detail) > 240 {
		detail = detail[:240]
	}
	return detail
}
