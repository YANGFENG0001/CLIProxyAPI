package executor

import (
	"github.com/router-for-me/CLIProxyAPI/v8/internal/util"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"github.com/tidwall/sjson"
)

// geminiBuiltInToolKeys lists the Gemini built-in (server-executed) tool keys
// that Gemini 3 accepts inside a tool list next to function declarations.
var geminiBuiltInToolKeys = []string{
	"googleSearch",
	"googleSearchRetrieval",
	"enterpriseWebSearch",
	"googleMaps",
	"codeExecution",
	"urlContext",
	"fileSearch",
	"computerUse",
	"retrieval",
}

// ensureGeminiServerSideToolInvocations is the API-key equivalent: the official
// Gemini API keeps tools and toolConfig at the root of the request body.
func ensureGeminiServerSideToolInvocations(payload []byte) []byte {
	return ensureServerSideToolInvocations(payload, "tools", "toolConfig")
}

// ensureServerSideToolInvocations enables the Gemini 3 tool context loop when a
// request mixes built-in tools with function calling.
//
// Gemini 3 rejects that combination with HTTP 400 "Please enable
// tool_config.include_server_side_tool_invocations to use Built-in tools with
// Function calling." until the flag is set. Requests that carry only built-in
// tools or only function declarations are returned unchanged, so every
// existing request keeps its current payload byte for byte.
func ensureServerSideToolInvocations(payload []byte, toolsPath, toolConfigPath string) []byte {
	tools := util.GetGJSONBytesNoCopy(payload, toolsPath)
	if !tools.IsArray() {
		return payload
	}

	hasBuiltInTool := false
	hasFunctionDeclarations := false
	tools.ForEach(func(_, tool gjson.Result) bool {
		if !tool.IsObject() {
			return true
		}
		if !hasFunctionDeclarations {
			hasFunctionDeclarations = toolHasFunctionDeclarations(tool)
		}
		if !hasBuiltInTool {
			for _, key := range geminiBuiltInToolKeys {
				if tool.Get(key).Exists() {
					hasBuiltInTool = true
					break
				}
			}
		}
		return !(hasBuiltInTool && hasFunctionDeclarations)
	})
	if !hasBuiltInTool || !hasFunctionDeclarations {
		return payload
	}

	flagPath := toolConfigPath + ".includeServerSideToolInvocations"
	if gjson.GetBytes(payload, flagPath).Bool() {
		return payload
	}

	updated, errSet := sjson.SetBytes(payload, flagPath, true)
	if errSet != nil {
		log.Debugf("server-side tool invocations: failed to set %s: %v", flagPath, errSet)
		return payload
	}
	return updated
}

// toolHasFunctionDeclarations reports whether a Gemini tool object carries at
// least one function declaration. Both the camelCase and the snake_case
// spellings reach the executors depending on the inbound protocol.
func toolHasFunctionDeclarations(tool gjson.Result) bool {
	for _, key := range []string{"functionDeclarations", "function_declarations"} {
		declarations := tool.Get(key)
		if declarations.IsArray() && declarations.Get("#").Int() > 0 {
			return true
		}
	}
	return false
}
