package executor

import (
	"bytes"
	"encoding/json"
	"testing"
)

func geminiRootToolConfigFlag(t *testing.T, payload []byte) (bool, bool) {
	t.Helper()

	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	toolConfig, ok := body["toolConfig"].(map[string]any)
	if !ok {
		return false, false
	}
	value, ok := toolConfig["includeServerSideToolInvocations"].(bool)
	if !ok {
		return false, false
	}
	return value, true
}

func TestEnsureGeminiServerSideToolInvocations_MixedTools(t *testing.T) {
	payload := []byte(`{"tools":[{"googleSearch":{}},{"functionDeclarations":[{"name":"tool_1"}]}],"contents":[]}`)

	got := ensureGeminiServerSideToolInvocations(payload)
	value, present := geminiRootToolConfigFlag(t, got)
	if !present || !value {
		t.Fatalf("toolConfig.includeServerSideToolInvocations missing or false: %s", string(got))
	}
}

func TestEnsureGeminiServerSideToolInvocations_FunctionOnlyIsUnchanged(t *testing.T) {
	payload := []byte(`{"tools":[{"functionDeclarations":[{"name":"tool_1"}]}],"contents":[]}`)

	if got := ensureGeminiServerSideToolInvocations(payload); !bytes.Equal(got, payload) {
		t.Fatalf("payload changed: %s", string(got))
	}
}

func TestEnsureGeminiServerSideToolInvocations_BuiltInOnlyIsUnchanged(t *testing.T) {
	payload := []byte(`{"tools":[{"googleSearch":{}}],"contents":[]}`)

	if got := ensureGeminiServerSideToolInvocations(payload); !bytes.Equal(got, payload) {
		t.Fatalf("payload changed: %s", string(got))
	}
}

func TestEnsureGeminiServerSideToolInvocations_NoToolsIsUnchanged(t *testing.T) {
	payload := []byte(`{"contents":[{"role":"user","parts":[{"text":"hi"}]}]}`)

	if got := ensureGeminiServerSideToolInvocations(payload); !bytes.Equal(got, payload) {
		t.Fatalf("payload changed: %s", string(got))
	}
}

func TestEnsureGeminiServerSideToolInvocations_AlreadyEnabledIsUnchanged(t *testing.T) {
	payload := []byte(`{"tools":[{"googleSearch":{}},{"functionDeclarations":[{"name":"a"}]}],"toolConfig":{"includeServerSideToolInvocations":true}}`)

	if got := ensureGeminiServerSideToolInvocations(payload); !bytes.Equal(got, payload) {
		t.Fatalf("payload changed: %s", string(got))
	}
}

func TestEnsureGeminiServerSideToolInvocations_PreservesFunctionCallingConfig(t *testing.T) {
	payload := []byte(`{"tools":[{"urlContext":{}},{"functionDeclarations":[{"name":"a"}]}],"toolConfig":{"functionCallingConfig":{"mode":"ANY"}}}`)

	got := ensureGeminiServerSideToolInvocations(payload)

	var body map[string]any
	if err := json.Unmarshal(got, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	toolConfig, _ := body["toolConfig"].(map[string]any)
	if flag, _ := toolConfig["includeServerSideToolInvocations"].(bool); !flag {
		t.Fatalf("flag not set: %s", string(got))
	}
	functionCallingConfig, _ := toolConfig["functionCallingConfig"].(map[string]any)
	if mode, _ := functionCallingConfig["mode"].(string); mode != "ANY" {
		t.Fatalf("functionCallingConfig.mode = %q, want ANY", mode)
	}
}
