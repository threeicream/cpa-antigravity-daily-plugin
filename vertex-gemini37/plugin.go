package main

/*
#include <stdint.h>
#include <stdlib.h>

typedef struct {
	void* ptr;
	size_t len;
} cliproxy_buffer;

typedef struct {
	uint32_t abi_version;
	void* host_ctx;
	void* call;
	void* free_buffer;
	void* shutdown;
} cliproxy_host_api;

typedef int (*cliproxy_plugin_call_fn)(char*, uint8_t*, size_t, cliproxy_buffer*);
typedef void (*cliproxy_plugin_free_fn)(void*, size_t);
typedef void (*cliproxy_plugin_shutdown_fn)(void);

typedef struct {
	uint32_t abi_version;
	cliproxy_plugin_call_fn call;
	cliproxy_plugin_free_fn free_buffer;
	cliproxy_plugin_shutdown_fn shutdown;
} cliproxy_plugin_api;

extern int cliproxyPluginCall(char*, uint8_t*, size_t, cliproxy_buffer*);
extern void cliproxyPluginFree(void*, size_t);
extern void cliproxyPluginShutdown(void);
*/
import "C"

import (
	"encoding/json"
	"strings"
	"unsafe"
)

const (
	abiVersion    = 1
	schemaVersion = 3
	modelID       = "gemini-3.7-flash"
	modelLow      = "gemini-3.7-flash-low"
	modelMedium   = "gemini-3.7-flash-medium"
	modelHigh     = "gemini-3.7-flash-high"
	modelTiered   = "gemini-3.7-flash-tiered"
	provider      = "vertex"
)

type envelope struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type routeRequest struct {
	RequestedModel string `json:"RequestedModel"`
}

type interceptRequest struct {
	ToFormat       string              `json:"ToFormat"`
	RequestedModel string              `json:"RequestedModel"`
	Headers        map[string][]string `json:"Headers"`
	Body           []byte              `json:"Body"`
}

func main() {}

//export cliproxy_plugin_init
func cliproxy_plugin_init(_ *C.cliproxy_host_api, plugin *C.cliproxy_plugin_api) C.int {
	if plugin == nil {
		return 1
	}
	plugin.abi_version = C.uint32_t(abiVersion)
	plugin.call = C.cliproxy_plugin_call_fn(C.cliproxyPluginCall)
	plugin.free_buffer = C.cliproxy_plugin_free_fn(C.cliproxyPluginFree)
	plugin.shutdown = C.cliproxy_plugin_shutdown_fn(C.cliproxyPluginShutdown)
	return 0
}

//export cliproxyPluginCall
func cliproxyPluginCall(method *C.char, request *C.uint8_t, requestLen C.size_t, response *C.cliproxy_buffer) C.int {
	if response != nil {
		response.ptr = nil
		response.len = 0
	}
	if method == nil {
		writeResponse(response, errorEnvelope("invalid_method", "method is required"))
		return 1
	}
	var raw []byte
	if request != nil && requestLen > 0 {
		raw = C.GoBytes(unsafe.Pointer(request), C.int(requestLen))
	}
	result, err := handleMethod(C.GoString(method), raw)
	if err != nil {
		writeResponse(response, errorEnvelope("plugin_error", err.Error()))
		return 1
	}
	writeResponse(response, result)
	return 0
}

//export cliproxyPluginFree
func cliproxyPluginFree(ptr unsafe.Pointer, _ C.size_t) {
	if ptr != nil {
		C.free(ptr)
	}
}

//export cliproxyPluginShutdown
func cliproxyPluginShutdown() {}

func handleMethod(method string, raw []byte) ([]byte, error) {
	switch method {
	case "plugin.register", "plugin.reconfigure":
		return okEnvelope(registration())
	case "model.register":
		return okEnvelope(modelRegistration())
	case "model.route":
		return route(raw)
	case "request.intercept_before":
		return passThrough(raw)
	case "request.intercept_after":
		return interceptAfter(raw)
	default:
		return errorEnvelope("unknown_method", "unknown method: "+method), nil
	}
}

func registration() map[string]any {
	return map[string]any{
		"schema_version": schemaVersion,
		"metadata": map[string]any{
			"Name":             "vertex-gemini37",
			"Version":          "0.1.0",
			"Author":           "local",
			"GitHubRepository": "https://github.com/router-for-me/CLIProxyAPI",
			"Logo":             "",
			"ConfigFields":     []any{},
		},
		"capabilities": map[string]any{
			"model_registrar":     true,
			"model_router":        true,
			"request_interceptor": true,
		},
	}
}

func modelRegistration() map[string]any {
	return map[string]any{
		"Provider": provider,
		"Models": []any{
			modelInfo(modelID, "Gemini 3.7 Flash"),
			modelInfo(modelLow, "Gemini 3.7 Flash Low"),
			modelInfo(modelMedium, "Gemini 3.7 Flash Medium"),
			modelInfo(modelHigh, "Gemini 3.7 Flash High"),
			modelInfo(modelTiered, "Gemini 3.7 Flash Tiered"),
		},
	}
}

func modelInfo(id, displayName string) map[string]any {
	return map[string]any{
		"ID":                         id,
		"Object":                     "model",
		"OwnedBy":                    provider,
		"DisplayName":                displayName,
		"SupportedGenerationMethods": []string{"chat", "generateContent"},
		"SupportedParameters":        []string{"thinking_level", "reasoning_effort"},
		"Thinking": map[string]any{
			"Min":            1,
			"Max":            65535,
			"DynamicAllowed": true,
			"Levels":         []string{"low", "medium", "high"},
		},
		"UserDefined": true,
	}
}

func route(raw []byte) ([]byte, error) {
	var req routeRequest
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &req); err != nil {
			return nil, err
		}
	}
	if !isAlias(req.RequestedModel) {
		return okEnvelope(map[string]any{"Handled": false})
	}
	return okEnvelope(map[string]any{
		"Handled":     true,
		"TargetKind":  "provider",
		"Target":      provider,
		"TargetModel": modelID,
		"Reason":      "route Vertex Gemini 3.7 Flash aliases to the built-in Vertex provider",
	})
}

func isAlias(model string) bool {
	switch strings.ToLower(strings.TrimSpace(model)) {
	case modelID, modelLow, modelMedium, modelHigh, modelTiered:
		return true
	default:
		return false
	}
}

func thinkingLevel(model string) string {
	switch strings.ToLower(strings.TrimSpace(model)) {
	case modelLow:
		return "LOW"
	case modelMedium:
		return "MEDIUM"
	case modelHigh:
		return "HIGH"
	default:
		return ""
	}
}

func passThrough(raw []byte) ([]byte, error) {
	var req interceptRequest
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &req); err != nil {
			return nil, err
		}
	}
	return okEnvelope(map[string]any{"Headers": req.Headers, "Body": req.Body})
}

func interceptAfter(raw []byte) ([]byte, error) {
	var req interceptRequest
	if len(raw) > 0 {
		if err := json.Unmarshal(raw, &req); err != nil {
			return nil, err
		}
	}
	level := thinkingLevel(req.RequestedModel)
	format := strings.ToLower(strings.TrimSpace(req.ToFormat))
	if level == "" || (format != "" && format != "gemini" && !strings.Contains(format, "vertex")) {
		return okEnvelope(map[string]any{"Headers": req.Headers, "Body": req.Body})
	}
	body, changed := rewriteThinkingBody(req.Body, level)
	if !changed {
		body = req.Body
	}
	return okEnvelope(map[string]any{"Headers": req.Headers, "Body": body})
}

func rewriteThinkingBody(body []byte, level string) ([]byte, bool) {
	var root map[string]any
	if len(body) == 0 || json.Unmarshal(body, &root) != nil {
		return body, false
	}
	target := root
	if nested, ok := root["request"].(map[string]any); ok {
		target = nested
	}
	config, ok := target["generationConfig"].(map[string]any)
	if !ok {
		if legacy, legacyOK := target["generation_config"].(map[string]any); legacyOK {
			config = legacy
		} else {
			config = map[string]any{}
			target["generationConfig"] = config
		}
	}
	thinking, ok := config["thinkingConfig"].(map[string]any)
	if !ok {
		if legacy, legacyOK := config["thinking_config"].(map[string]any); legacyOK {
			thinking = legacy
		} else {
			thinking = map[string]any{}
			config["thinkingConfig"] = thinking
		}
	}
	thinking["thinkingLevel"] = level
	delete(thinking, "thinking_level")
	delete(thinking, "thinkingBudget")
	delete(thinking, "thinking_budget")
	updated, err := json.Marshal(root)
	if err != nil {
		return body, false
	}
	return updated, true
}

func okEnvelope(result any) ([]byte, error) {
	raw, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}
	return json.Marshal(envelope{OK: true, Result: raw})
}

func errorEnvelope(code, message string) []byte {
	raw, _ := json.Marshal(envelope{OK: false, Error: &rpcError{Code: code, Message: message}})
	return raw
}

func writeResponse(response *C.cliproxy_buffer, raw []byte) {
	if response == nil || len(raw) == 0 {
		return
	}
	ptr := C.CBytes(raw)
	if ptr == nil {
		return
	}
	response.ptr = ptr
	response.len = C.size_t(len(raw))
}
