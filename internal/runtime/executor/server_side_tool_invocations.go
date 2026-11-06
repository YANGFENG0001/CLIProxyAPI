package executor

import (
	"github.com/MoeForge/moego/ojson"
	log "github.com/sirupsen/logrus"
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

// ensureGeminiServerSideToolInvocationsDoc enables the flag on the final
// business payload before payload rules are applied. The official Gemini API
// keeps tools and toolConfig at the root of the request body.
func ensureGeminiServerSideToolInvocationsDoc(doc *ojson.Doc) {
	ensureServerSideToolInvocationsDoc(doc, "tools", "toolConfig")
}

// ensureAntigravityServerSideToolInvocationsDoc is the Antigravity equivalent:
// both fields live under request.
func ensureAntigravityServerSideToolInvocationsDoc(doc *ojson.Doc) {
	ensureServerSideToolInvocationsDoc(doc, "request.tools", "request.toolConfig")
}

// ensureServerSideToolInvocationsDoc enables the Gemini 3 tool context loop when
// a request mixes built-in tools with function calling.
//
// Gemini 3 rejects that combination with HTTP 400 "Please enable
// tool_config.include_server_side_tool_invocations to use Built-in tools with
// Function calling." until the flag is set. Requests that carry only built-in
// tools or only function declarations are left untouched. The return value
// reports whether the document changed so byte-oriented callers can keep the
// original payload bytes when nothing needs to be injected.
func ensureServerSideToolInvocationsDoc(doc *ojson.Doc, toolsPath, toolConfigPath string) bool {
	if doc == nil {
		return false
	}
	tools := doc.Get(toolsPath)
	if !tools.IsArray() {
		return false
	}

	hasBuiltInTool := false
	hasFunctionDeclarations := false
	for _, tool := range tools.Array() {
		if !tool.IsObject() {
			continue
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
		if hasBuiltInTool && hasFunctionDeclarations {
			break
		}
	}
	if !hasBuiltInTool || !hasFunctionDeclarations {
		return false
	}

	flagPath := toolConfigPath + ".includeServerSideToolInvocations"
	if doc.Get(flagPath).Bool() {
		return false
	}
	if errSet := doc.Set(flagPath, true); errSet != nil {
		log.Debugf("server-side tool invocations: failed to set %s: %v", flagPath, errSet)
		return false
	}
	return true
}

// toolHasFunctionDeclarations reports whether a Gemini tool object carries at
// least one function declaration. Both the camelCase and the snake_case
// spellings reach the executors depending on the inbound protocol.
func toolHasFunctionDeclarations(tool ojson.Result) bool {
	for _, key := range []string{"functionDeclarations", "function_declarations"} {
		declarations := tool.Get(key)
		if declarations.IsArray() && len(declarations.Array()) > 0 {
			return true
		}
	}
	return false
}

// ensureGeminiServerSideToolInvocations keeps the byte-oriented entry point
// used by the regression tests.
func ensureGeminiServerSideToolInvocations(payload []byte) []byte {
	return ensureServerSideToolInvocations(payload, "tools", "toolConfig")
}

// ensureServerSideToolInvocations applies the same rule to a raw payload and
// returns the original bytes when no injection is needed.
func ensureServerSideToolInvocations(payload []byte, toolsPath, toolConfigPath string) []byte {
	doc := ojson.NewDoc(payload)
	if !ensureServerSideToolInvocationsDoc(doc, toolsPath, toolConfigPath) {
		return payload
	}
	return doc.Bytes()
}
