package main

import (
	"encoding/json"
	"net/url"
	"time"
)

const (
	pluginID      = "antigravity-daily"
	providerID    = "antigravity-daily"
	pluginVersion = "0.1.1"
	resourcePath  = "/login"
)

type envelope struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *envelopeError  `json:"error,omitempty"`
}

type envelopeError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Retryable  bool   `json:"retryable,omitempty"`
	HTTPStatus int    `json:"http_status,omitempty"`
}

type registration struct {
	SchemaVersion uint32                   `json:"schema_version"`
	Metadata      metadata                 `json:"metadata"`
	Capabilities  registrationCapabilities `json:"capabilities"`
}

type metadata struct {
	Name             string `json:"Name"`
	Version          string `json:"Version"`
	Author           string `json:"Author"`
	GitHubRepository string `json:"GitHubRepository"`
	Logo             string `json:"Logo,omitempty"`
	ConfigFields     []any  `json:"ConfigFields"`
}

type configField struct {
	Name        string `json:"Name"`
	Type        string `json:"Type"`
	Description string `json:"Description"`
}

type lifecycleRequest struct {
	ConfigYAML []byte `json:"config_yaml"`
}

type registrationCapabilities struct {
	AuthProvider  bool `json:"auth_provider"`
	ManagementAPI bool `json:"management_api"`
}

type hostConfigSummary struct {
	AuthDir  string `json:"AuthDir"`
	ProxyURL string `json:"ProxyURL"`
}

type authLoginStartRequest struct {
	Provider       string            `json:"Provider"`
	BaseURL        string            `json:"BaseURL"`
	Host           hostConfigSummary `json:"Host"`
	Metadata       map[string]any    `json:"Metadata"`
	HostCallbackID string            `json:"host_callback_id,omitempty"`
}

type authLoginStartResponse struct {
	Provider  string         `json:"Provider"`
	URL       string         `json:"URL"`
	State     string         `json:"State"`
	ExpiresAt time.Time      `json:"ExpiresAt"`
	Metadata  map[string]any `json:"Metadata,omitempty"`
}

type authLoginPollRequest struct {
	Provider       string            `json:"Provider"`
	State          string            `json:"State"`
	Host           hostConfigSummary `json:"Host"`
	Metadata       map[string]any    `json:"Metadata"`
	HostCallbackID string            `json:"host_callback_id,omitempty"`
}

type authLoginPollResponse struct {
	Status  string   `json:"Status"`
	Message string   `json:"Message,omitempty"`
	Auth    authData `json:"Auth,omitempty"`
}

type authParseResponse struct {
	Handled bool `json:"Handled"`
}

type authData struct {
	Provider         string            `json:"Provider"`
	ID               string            `json:"ID"`
	FileName         string            `json:"FileName"`
	Label            string            `json:"Label"`
	StorageJSON      []byte            `json:"StorageJSON"`
	Metadata         map[string]any    `json:"Metadata"`
	Attributes       map[string]string `json:"Attributes,omitempty"`
	NextRefreshAfter time.Time         `json:"NextRefreshAfter,omitempty"`
}

type managementRegistration struct {
	Resources []managementResource `json:"resources,omitempty"`
}

type managementResource struct {
	Path        string `json:"Path"`
	Menu        string `json:"Menu"`
	Description string `json:"Description"`
}

type managementResponse struct {
	StatusCode int                 `json:"StatusCode"`
	Headers    map[string][]string `json:"Headers"`
	Body       []byte              `json:"Body"`
}

type upstreamRequest struct {
	Method  string
	URL     string
	Headers map[string][]string
	Body    []byte
}

type hostHTTPRequest struct {
	HostCallbackID string              `json:"host_callback_id,omitempty"`
	Method         string              `json:"method,omitempty"`
	URL            string              `json:"url,omitempty"`
	Headers        map[string][]string `json:"headers,omitempty"`
	Body           []byte              `json:"body,omitempty"`
}

type upstreamResponse struct {
	StatusCode int
	Headers    map[string][]string
	Body       []byte
}

type httpDoFunc func(callbackID string, req upstreamRequest) (upstreamResponse, error)

type oauthCallback struct {
	Code  string `json:"code"`
	State string `json:"state"`
	Error string `json:"error"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	TokenType    string `json:"token_type"`
}

type userInfo struct {
	Email string `json:"email"`
}

func cloneValues(in url.Values) url.Values {
	if in == nil {
		return nil
	}
	out := make(url.Values, len(in))
	for key, values := range in {
		out[key] = append([]string(nil), values...)
	}
	return out
}
