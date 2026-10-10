package executor

import (
	"encoding/json"
	"testing"

	geminiapi "github.com/router-for-me/CLIProxyAPI/v8/internal/translator/antigravity/gemini"
)

// inbound mirrors the real downstream request captured from a Codex -> new-api ->
// CLIProxyAPI chain: Gemini-native body carrying a built-in tool (googleSearch)
// next to function declarations, plus a toolConfig that only sets
// functionCallingConfig. This is the exact shape Gemini 3 rejects with
// "Please enable tool_config.include_server_side_tool_invocations".
const inboundMixedToolsGeminiBody = `{
	"contents": [{"role": "user", "parts": [{"text": "hi"}]}],
	"safetySettings": [{"category": "HARM_CATEGORY_HARASSMENT", "threshold": "OFF"}],
	"generationConfig": {"thinkingConfig": {"thinkingLevel": "high"}},
	"tools": [
		{"googleSearch": {}},
		{"functionDeclarations": [
			{"name": "exec_command", "description": "Runs a command", "parameters": {"type": "OBJECT", "properties": {"cmd": {"type": "STRING"}}, "required": ["cmd"]}}
		]}
	],
	"toolConfig": {"functionCallingConfig": {"mode": "AUTO"}}
}`

func TestAntigravityEndToEnd_MixedToolsFromGeminiNativeBody(t *testing.T) {
	translated, err := geminiapi.ConvertGeminiRequestToAntigravity("gemini-3-flash", []byte(inboundMixedToolsGeminiBody), true)
	if err != nil {
		t.Fatalf("translate: %v", err)
	}

	var translatedBody map[string]any
	if err := json.Unmarshal(translated, &translatedBody); err != nil {
		t.Fatalf("unmarshal translated: %v", err)
	}
	request, _ := translatedBody["request"].(map[string]any)
	if request == nil {
		t.Fatalf("translator did not wrap the body under request: %s", string(translated))
	}
	tools, _ := request["tools"].([]any)
	t.Logf("translated request.tools = %v", tools)
	if len(tools) == 0 {
		t.Fatalf("translator dropped request.tools: %s", string(translated))
	}

	body := buildRequestBodyFromRawPayload(t, "gemini-3-flash", translated)

	value, present := antigravityToolConfigFlag(t, body)
	if !present || !value {
		raw, _ := json.Marshal(body)
		t.Fatalf("includeServerSideToolInvocations missing or false: present=%v value=%v body=%s", present, value, raw)
	}

	toolConfig, _ := body["request"].(map[string]any)["toolConfig"].(map[string]any)
	functionCallingConfig, _ := toolConfig["functionCallingConfig"].(map[string]any)
	if mode, _ := functionCallingConfig["mode"].(string); mode != "AUTO" {
		t.Fatalf("functionCallingConfig.mode = %q, want AUTO", mode)
	}
}
