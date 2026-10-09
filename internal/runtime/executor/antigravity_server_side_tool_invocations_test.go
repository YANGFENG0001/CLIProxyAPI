package executor

import (
	"bytes"
	"testing"
)

func antigravityToolConfigFlag(t *testing.T, body map[string]any) (bool, bool) {
	t.Helper()

	request, ok := body["request"].(map[string]any)
	if !ok {
		t.Fatalf("request missing or invalid type")
	}
	toolConfig, ok := request["toolConfig"].(map[string]any)
	if !ok {
		return false, false
	}
	value, ok := toolConfig["includeServerSideToolInvocations"].(bool)
	if !ok {
		return false, false
	}
	return value, true
}

func TestAntigravityBuildRequest_EnablesServerSideToolInvocationsForMixedTools(t *testing.T) {
	body := buildRequestBodyFromRawPayload(t, "gemini-3-pro-preview", []byte(`{
		"request": {
			"tools": [
				{"googleSearch": {}},
				{
					"functionDeclarations": [
						{"name": "tool_1", "parameters": {"type": "object", "properties": {"arg": {"type": "string"}}}}
					]
				}
			],
			"contents": [{"role": "user", "parts": [{"text": "hello"}]}]
		}
	}`))

	value, present := antigravityToolConfigFlag(t, body)
	if !present {
		t.Fatalf("request.toolConfig.includeServerSideToolInvocations missing, body=%v", body)
	}
	if !value {
		t.Fatalf("request.toolConfig.includeServerSideToolInvocations = false, want true")
	}
}

func TestAntigravityBuildRequest_EnablesServerSideToolInvocationsForSnakeCaseDeclarations(t *testing.T) {
	body := buildRequestBodyFromRawPayload(t, "gemini-3-pro-preview", []byte(`{
		"request": {
			"tools": [
				{"codeExecution": {}},
				{
					"function_declarations": [
						{"name": "tool_1", "parameters": {"type": "object", "properties": {"arg": {"type": "string"}}}}
					]
				}
			],
			"contents": [{"role": "user", "parts": [{"text": "hello"}]}]
		}
	}`))

	value, present := antigravityToolConfigFlag(t, body)
	if !present || !value {
		t.Fatalf("expected includeServerSideToolInvocations=true, present=%v value=%v", present, value)
	}
}

func TestAntigravityBuildRequest_KeepsOnlyFunctionToolsUntouched(t *testing.T) {
	body := buildRequestBodyFromPayload(t, "gemini-3-pro-preview")

	if _, present := antigravityToolConfigFlag(t, body); present {
		t.Fatalf("request.toolConfig.includeServerSideToolInvocations should stay absent for function-only tools")
	}
}

func TestAntigravityBuildRequest_KeepsOnlyBuiltInToolsUntouched(t *testing.T) {
	body := buildRequestBodyFromRawPayload(t, "gemini-3-pro-preview", []byte(`{
		"request": {
			"tools": [{"googleSearch": {}}],
			"contents": [{"role": "user", "parts": [{"text": "hello"}]}]
		}
	}`))

	if _, present := antigravityToolConfigFlag(t, body); present {
		t.Fatalf("request.toolConfig.includeServerSideToolInvocations should stay absent for built-in-only tools")
	}
}

func TestAntigravityBuildRequest_PreservesExistingServerSideToolInvocations(t *testing.T) {
	body := buildRequestBodyFromRawPayload(t, "gemini-3-pro-preview", []byte(`{
		"request": {
			"tools": [
				{"urlContext": {}},
				{
					"functionDeclarations": [
						{"name": "tool_1", "parameters": {"type": "object", "properties": {"arg": {"type": "string"}}}}
					]
				}
			],
			"toolConfig": {"includeServerSideToolInvocations": true, "functionCallingConfig": {"mode": "AUTO"}},
			"contents": [{"role": "user", "parts": [{"text": "hello"}]}]
		}
	}`))

	value, present := antigravityToolConfigFlag(t, body)
	if !present || !value {
		t.Fatalf("expected includeServerSideToolInvocations=true, present=%v value=%v", present, value)
	}

	request, _ := body["request"].(map[string]any)
	toolConfig, _ := request["toolConfig"].(map[string]any)
	functionCallingConfig, _ := toolConfig["functionCallingConfig"].(map[string]any)
	if mode, _ := functionCallingConfig["mode"].(string); mode != "AUTO" {
		t.Fatalf("functionCallingConfig.mode = %q, want %q", mode, "AUTO")
	}
}

func TestEnsureAntigravityServerSideToolInvocations_NoToolsIsUnchanged(t *testing.T) {
	payload := []byte(`{"request":{"contents":[{"role":"user","parts":[{"text":"hi"}]}]}}`)
	got := ensureAntigravityServerSideToolInvocations(payload)
	if !bytes.Equal(got, payload) {
		t.Fatalf("payload changed: %s", string(got))
	}
}

func TestEnsureAntigravityServerSideToolInvocations_AlreadyEnabledIsUnchanged(t *testing.T) {
	payload := []byte(`{"request":{"tools":[{"googleSearch":{}},{"functionDeclarations":[{"name":"a"}]}],"toolConfig":{"includeServerSideToolInvocations":true}}}`)
	got := ensureAntigravityServerSideToolInvocations(payload)
	if !bytes.Equal(got, payload) {
		t.Fatalf("payload changed: %s", string(got))
	}
}

func TestEnsureAntigravityServerSideToolInvocations_EmptyDeclarationsAreIgnored(t *testing.T) {
	payload := []byte(`{"request":{"tools":[{"googleSearch":{}},{"functionDeclarations":[]}]}}`)
	got := ensureAntigravityServerSideToolInvocations(payload)
	if !bytes.Equal(got, payload) {
		t.Fatalf("payload changed: %s", string(got))
	}
}
