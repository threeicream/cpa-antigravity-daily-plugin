package main

import (
	"encoding/json"
	"testing"
)

func TestRewriteThinkingBody(t *testing.T) {
	input := []byte(`{"request":{"generationConfig":{"thinkingConfig":{"thinkingBudget":1024}}}}`)
	output, changed := rewriteThinkingBody(input, "low")
	if !changed {
		t.Fatal("expected payload to change")
	}
	var got map[string]any
	if err := json.Unmarshal(output, &got); err != nil {
		t.Fatal(err)
	}
	thinking := got["request"].(map[string]any)["generationConfig"].(map[string]any)["thinkingConfig"].(map[string]any)
	if thinking["thinkingLevel"] != "low" {
		t.Fatalf("thinking level = %v", thinking["thinkingLevel"])
	}
	if _, ok := thinking["thinkingBudget"]; ok {
		t.Fatal("thinking budget was not removed")
	}
}

func TestRewriteLeavesInvalidJSON(t *testing.T) {
	input := []byte("not-json")
	output, changed := rewriteThinkingBody(input, "high")
	if changed || string(output) != string(input) {
		t.Fatal("invalid JSON must pass through unchanged")
	}
}

func TestRouteAliases(t *testing.T) {
	for _, model := range []string{modelLow, modelMedium, modelHigh, modelTiered} {
		raw, err := route([]byte(`{"RequestedModel":"` + model + `"}`))
		if err != nil {
			t.Fatal(err)
		}
		if string(raw) == "" {
			t.Fatal("empty route response")
		}
	}
}
