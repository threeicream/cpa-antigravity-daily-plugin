package main

import (
	"encoding/json"
	"testing"
)

func TestRouteVertexModel(t *testing.T) {
	for _, model := range []string{modelID, modelLow, modelMedium, modelHigh, modelTiered} {
		raw, err := route([]byte(`{"RequestedModel":"` + model + `"}`))
		if err != nil {
			t.Fatal(err)
		}
		var got envelope
		if err := json.Unmarshal(raw, &got); err != nil {
			t.Fatal(err)
		}
		if !got.OK || string(got.Result) == "" {
			t.Fatalf("unexpected route response for %s: %s", model, raw)
		}
	}
}

func TestRewriteThinkingBody(t *testing.T) {
	input := []byte(`{"generationConfig":{"thinkingConfig":{"thinkingBudget":1024}}}`)
	output, changed := rewriteThinkingBody(input, "LOW")
	if !changed {
		t.Fatal("expected payload to change")
	}
	var got map[string]any
	if err := json.Unmarshal(output, &got); err != nil {
		t.Fatal(err)
	}
	thinking := got["generationConfig"].(map[string]any)["thinkingConfig"].(map[string]any)
	if thinking["thinkingLevel"] != "LOW" {
		t.Fatalf("thinking level = %v", thinking["thinkingLevel"])
	}
	if _, ok := thinking["thinkingBudget"]; ok {
		t.Fatal("thinking budget was not removed")
	}
}

func TestRouteIgnoresOtherModels(t *testing.T) {
	raw, err := route([]byte(`{"RequestedModel":"gemini-3.6-flash"}`))
	if err != nil {
		t.Fatal(err)
	}
	var got envelope
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	var result map[string]any
	if err := json.Unmarshal(got.Result, &result); err != nil {
		t.Fatal(err)
	}
	if result["Handled"] != false {
		t.Fatalf("unexpected route response: %s", raw)
	}
}
