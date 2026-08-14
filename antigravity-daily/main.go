package main

/*
#include <stdint.h>
#include <stdlib.h>

typedef struct {
	void* ptr;
	size_t len;
} cliproxy_buffer;

typedef int (*cliproxy_host_call_fn)(void*, const char*, const uint8_t*, size_t, cliproxy_buffer*);
typedef void (*cliproxy_host_free_fn)(void*, size_t);

typedef struct {
	uint32_t abi_version;
	void* host_ctx;
	cliproxy_host_call_fn call;
	cliproxy_host_free_fn free_buffer;
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

static const cliproxy_host_api* stored_host;

static void store_host_api(const cliproxy_host_api* host) {
	stored_host = host;
}

static int call_host_api(const char* method, const uint8_t* request, size_t request_len, cliproxy_buffer* response) {
	if (stored_host == NULL || stored_host->call == NULL) {
		return 1;
	}
	return stored_host->call(stored_host->host_ctx, method, request, request_len, response);
}

static void free_host_buffer(void* ptr, size_t len) {
	if (stored_host != NULL && stored_host->free_buffer != NULL && ptr != NULL) {
		stored_host->free_buffer(ptr, len);
	}
}
*/
import "C"

import (
	"encoding/json"
	"fmt"
	"unsafe"
)

const abiVersion uint32 = 1

func main() {}

//export cliproxy_plugin_init
func cliproxy_plugin_init(host *C.cliproxy_host_api, plugin *C.cliproxy_plugin_api) C.int {
	if host == nil || plugin == nil {
		return 1
	}
	C.store_host_api(host)
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
	var requestBytes []byte
	if request != nil && requestLen > 0 {
		requestBytes = C.GoBytes(unsafe.Pointer(request), C.int(requestLen))
	}
	raw, err := handleMethod(C.GoString(method), requestBytes)
	if err != nil {
		writeResponse(response, errorEnvelope("plugin_error", err.Error()))
		return 1
	}
	writeResponse(response, raw)
	return 0
}

//export cliproxyPluginFree
func cliproxyPluginFree(ptr unsafe.Pointer, length C.size_t) {
	if ptr != nil {
		C.free(ptr)
	}
	_ = length
}

//export cliproxyPluginShutdown
func cliproxyPluginShutdown() {}

func handleMethod(method string, request []byte) ([]byte, error) {
	switch method {
	case "plugin.register", "plugin.reconfigure":
		if err := configure(request); err != nil {
			return nil, err
		}
		return okEnvelope(pluginRegistration())
	case "auth.identifier":
		return okEnvelope(map[string]string{"identifier": providerID})
	case "auth.parse":
		return okEnvelope(authParseResponse{Handled: false})
	case "auth.login.start":
		var req authLoginStartRequest
		if err := json.Unmarshal(request, &req); err != nil {
			return nil, fmt.Errorf("decode login start request: %w", err)
		}
		resp, err := startLogin(req)
		if err != nil {
			return nil, err
		}
		return okEnvelope(resp)
	case "auth.login.poll":
		var req authLoginPollRequest
		if err := json.Unmarshal(request, &req); err != nil {
			return nil, fmt.Errorf("decode login poll request: %w", err)
		}
		resp, err := pollLogin(req, hostHTTPDo)
		if err != nil {
			return nil, err
		}
		return okEnvelope(resp)
	case "auth.refresh":
		return nil, fmt.Errorf("refresh is handled by CPA's built-in antigravity executor")
	case "management.register":
		return okEnvelope(managementRegistration{
			Resources: []managementResource{{
				Path:        resourcePath,
				Menu:        "Antigravity Daily 登录",
				Description: "使用 daily endpoint 发现并保存带 project_id 的 Antigravity 凭证。",
			}},
		})
	case "management.handle":
		return okEnvelope(managementResponse{
			StatusCode: 200,
			Headers: map[string][]string{
				"Content-Type":            {"text/html; charset=utf-8"},
				"Cache-Control":           {"no-store"},
				"Content-Security-Policy": {"default-src 'none'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; connect-src 'self'; form-action 'none'; base-uri 'none'; frame-ancestors 'self'"},
			},
			Body: []byte(loginPageHTML),
		})
	default:
		return errorEnvelope("unknown_method", "unknown method: "+method), nil
	}
}

func pluginRegistration() registration {
	return registration{
		SchemaVersion: 2,
		Metadata: metadata{
			Name:             pluginID,
			Version:          pluginVersion,
			Author:           "threeicream",
			GitHubRepository: "https://github.com/threeicream/cpa-antigravity-daily-plugin",
			ConfigFields: []any{
				configField{Name: "client_id", Type: "string", Description: "Google OAuth installed-application client ID."},
				configField{Name: "client_secret", Type: "string", Description: "Google OAuth installed-application client secret. Keep this value in CPA config only."},
			},
		},
		Capabilities: registrationCapabilities{
			AuthProvider:  true,
			ManagementAPI: true,
		},
	}
}

func okEnvelope(value any) ([]byte, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	return json.Marshal(envelope{OK: true, Result: raw})
}

func errorEnvelope(code, message string) []byte {
	raw, _ := json.Marshal(envelope{OK: false, Error: &envelopeError{Code: code, Message: message}})
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

func hostHTTPDo(callbackID string, req upstreamRequest) (upstreamResponse, error) {
	payload := hostHTTPRequest{
		HostCallbackID: callbackID,
		Method:         req.Method,
		URL:            req.URL,
		Headers:        req.Headers,
		Body:           req.Body,
	}
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		return upstreamResponse{}, fmt.Errorf("encode host HTTP request: %w", err)
	}
	cMethod := C.CString("host.http.do")
	defer C.free(unsafe.Pointer(cMethod))
	cPayload := C.CBytes(rawPayload)
	if cPayload == nil {
		return upstreamResponse{}, fmt.Errorf("allocate host HTTP request")
	}
	defer C.free(cPayload)

	var response C.cliproxy_buffer
	code := C.call_host_api(cMethod, (*C.uint8_t)(cPayload), C.size_t(len(rawPayload)), &response)
	var rawResponse []byte
	if response.ptr != nil && response.len > 0 {
		rawResponse = C.GoBytes(response.ptr, C.int(response.len))
	}
	if response.ptr != nil {
		C.free_host_buffer(response.ptr, response.len)
	}
	if len(rawResponse) == 0 {
		return upstreamResponse{}, fmt.Errorf("host HTTP callback returned no response (code %d)", int(code))
	}
	var env envelope
	if err := json.Unmarshal(rawResponse, &env); err != nil {
		return upstreamResponse{}, fmt.Errorf("decode host HTTP envelope: %w", err)
	}
	if !env.OK {
		if env.Error != nil {
			return upstreamResponse{}, fmt.Errorf("host HTTP callback: %s", env.Error.Message)
		}
		return upstreamResponse{}, fmt.Errorf("host HTTP callback failed")
	}
	if code != 0 {
		return upstreamResponse{}, fmt.Errorf("host HTTP callback returned code %d", int(code))
	}
	var result upstreamResponse
	if err := json.Unmarshal(env.Result, &result); err != nil {
		return upstreamResponse{}, fmt.Errorf("decode host HTTP response: %w", err)
	}
	return result, nil
}
