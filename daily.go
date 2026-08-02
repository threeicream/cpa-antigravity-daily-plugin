package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	dailyAPIBase         = "https://daily-cloudcode-pa.googleapis.com"
	antigravityUserAgent = "antigravity/cli/1.1.9 windows/amd64"
)

func discoverDailyProject(callbackID, accessToken string, do httpDoFunc) (string, string, error) {
	load, err := loadCodeAssist(callbackID, accessToken, map[string]any{"ideType": "ANTIGRAVITY"}, do)
	if err != nil {
		return "", "", err
	}
	tier := extractTier(load)
	if projectID := extractProjectID(load); projectID != "" {
		return projectID, tier, nil
	}

	onboardMetadata := map[string]any{
		"ideType":    "ANTIGRAVITY",
		"platform":   "PLATFORM_UNSPECIFIED",
		"pluginType": "GEMINI",
	}
	tierLoad, err := loadCodeAssist(callbackID, accessToken, onboardMetadata, do)
	if err != nil {
		return "", tier, err
	}
	if tier == "" {
		tier = extractTier(tierLoad)
	}
	tierID := selectOnboardTier(tierLoad)
	if tierID == "" {
		return "", tier, fmt.Errorf("daily loadCodeAssist returned no usable onboarding tier")
	}

	for attempt := 1; attempt <= 5; attempt++ {
		projectID, done, err := onboardUser(callbackID, accessToken, tierID, onboardMetadata, do)
		if err != nil {
			return "", tier, err
		}
		if projectID != "" {
			return projectID, tier, nil
		}
		if done {
			return "", tier, fmt.Errorf("daily onboardUser completed without project_id")
		}
		if attempt < 5 {
			time.Sleep(2 * time.Second)
		}
	}
	return "", tier, fmt.Errorf("daily onboardUser did not complete within 5 attempts")
}

func loadCodeAssist(callbackID, accessToken string, metadata map[string]any, do httpDoFunc) (map[string]any, error) {
	body, err := json.Marshal(map[string]any{"metadata": metadata})
	if err != nil {
		return nil, err
	}
	resp, err := do(callbackID, upstreamRequest{
		Method:  http.MethodPost,
		URL:     dailyAPIBase + "/v1internal:loadCodeAssist",
		Headers: antigravityHeaders(accessToken),
		Body:    body,
	})
	if err != nil {
		return nil, fmt.Errorf("daily loadCodeAssist: %w", err)
	}
	if err := require2xx("daily loadCodeAssist", resp); err != nil {
		return nil, err
	}
	var result map[string]any
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return nil, fmt.Errorf("decode daily loadCodeAssist: %w", err)
	}
	return result, nil
}

func onboardUser(callbackID, accessToken, tierID string, metadata map[string]any, do httpDoFunc) (string, bool, error) {
	body, err := json.Marshal(map[string]any{
		"tierId":   tierID,
		"metadata": metadata,
	})
	if err != nil {
		return "", false, err
	}
	resp, err := do(callbackID, upstreamRequest{
		Method:  http.MethodPost,
		URL:     dailyAPIBase + "/v1internal:onboardUser",
		Headers: antigravityHeaders(accessToken),
		Body:    body,
	})
	if err != nil {
		return "", false, fmt.Errorf("daily onboardUser: %w", err)
	}
	if err := require2xx("daily onboardUser", resp); err != nil {
		return "", false, err
	}
	var result map[string]any
	if err := json.Unmarshal(resp.Body, &result); err != nil {
		return "", false, fmt.Errorf("decode daily onboardUser: %w", err)
	}
	done, _ := result["done"].(bool)
	if !done {
		return "", false, nil
	}
	if response, ok := result["response"].(map[string]any); ok {
		return extractProjectID(response), true, nil
	}
	return extractProjectID(result), true, nil
}

func antigravityHeaders(accessToken string) map[string][]string {
	return map[string][]string{
		"Authorization": {"Bearer " + accessToken},
		"Content-Type":  {"application/json"},
		"User-Agent":    {antigravityUserAgent},
	}
}

func extractProjectID(data map[string]any) string {
	for _, key := range []string{"cloudaicompanionProject", "projectId", "project", "userDefinedCloudaicompanionProject"} {
		switch value := data[key].(type) {
		case string:
			if trimmed := strings.TrimSpace(value); trimmed != "" {
				return trimmed
			}
		case map[string]any:
			if id, ok := value["id"].(string); ok && strings.TrimSpace(id) != "" {
				return strings.TrimSpace(id)
			}
		}
	}
	return ""
}

func extractTier(data map[string]any) string {
	for _, key := range []string{"paidTier", "currentTier"} {
		if tier, ok := data[key].(map[string]any); ok {
			if id, ok := tier["id"].(string); ok && strings.TrimSpace(id) != "" {
				return strings.TrimSpace(id)
			}
		}
	}
	return ""
}

func selectOnboardTier(data map[string]any) string {
	tiers, _ := data["allowedTiers"].([]any)
	for _, raw := range tiers {
		tier, _ := raw.(map[string]any)
		isDefault, _ := tier["isDefault"].(bool)
		id, _ := tier["id"].(string)
		if isDefault && strings.TrimSpace(id) != "" {
			return strings.TrimSpace(id)
		}
	}
	for _, raw := range tiers {
		tier, _ := raw.(map[string]any)
		id, _ := tier["id"].(string)
		if strings.EqualFold(strings.TrimSpace(id), "free-tier") {
			return strings.TrimSpace(id)
		}
	}
	return "LEGACY"
}
