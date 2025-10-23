package headers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"
	"text/template"
)

const (
	EnvHeaderPrefix = "X-Mcp-Env-"
	ArgHeaderPrefix = "X-Mcp-Arg-"
	ArgsHeader      = "X-Mcp-Args"
)

// ParseEnvHeaders extracts environment variables from X-MCP-Env-* headers
func ParseEnvHeaders(headers http.Header) map[string]string {
	env := make(map[string]string)

	for key, values := range headers {
		if strings.HasPrefix(key, EnvHeaderPrefix) && len(values) > 0 {
			// X-MCP-Env-API-KEY -> API_KEY
			envKey := strings.TrimPrefix(key, EnvHeaderPrefix)
			envKey = strings.ToUpper(strings.ReplaceAll(envKey, "-", "_"))
			env[envKey] = values[0]
		}
	}

	return env
}

// ParseArgsHeaders extracts arguments from X-MCP-Args or X-MCP-Arg-* headers
func ParseArgsHeaders(headers http.Header, argsTemplate []string) []string {
	// Pattern 1: JSON array in X-MCP-Args header
	if argsJSON := headers.Get(ArgsHeader); argsJSON != "" {
		var args []string
		if err := json.Unmarshal([]byte(argsJSON), &args); err == nil {
			return args
		}
	}

	// Pattern 2: Named arguments in X-MCP-Arg-* headers
	namedArgs := make(map[string]string)
	for key, values := range headers {
		if strings.HasPrefix(key, ArgHeaderPrefix) && len(values) > 0 {
			// X-MCP-Arg-path -> path
			argKey := strings.TrimPrefix(key, ArgHeaderPrefix)
			argKey = strings.ToLower(argKey)
			namedArgs[argKey] = values[0]
		}
	}

	// Apply template if provided
	if len(argsTemplate) > 0 && len(namedArgs) > 0 {
		return applyArgsTemplate(argsTemplate, namedArgs)
	}

	return []string{}
}

func applyArgsTemplate(templates []string, data map[string]string) []string {
	result := make([]string, 0, len(templates))

	for _, tmpl := range templates {
		t, err := template.New("arg").Parse(tmpl)
		if err != nil {
			result = append(result, tmpl)
			continue
		}

		var buf bytes.Buffer
		if err := t.Execute(&buf, data); err != nil {
			result = append(result, tmpl)
			continue
		}

		result = append(result, buf.String())
	}

	return result
}

// ParseCustomHeaders はカスタムヘッダーマッピングに基づいてHTTPヘッダーから環境変数と引数を抽出します
// envMapping: ヘッダー名 → 環境変数名 (例: "X-Slack-Token" → "SLACK_TOKEN")
// argMapping: ヘッダー名 → 引数名 (例: "X-Team-Id" → "team-id")
func ParseCustomHeaders(headers http.Header, envMapping, argMapping map[string]string) (map[string]string, []string) {
	envVars := make(map[string]string)
	var args []string

	// 環境変数マッピング
	for headerName, envName := range envMapping {
		if value := headers.Get(headerName); value != "" {
			envVars[envName] = value
		}
	}

	// 引数マッピング
	for headerName, argName := range argMapping {
		if value := headers.Get(headerName); value != "" {
			// "team-id" → "--team-id value" 形式で追加
			args = append(args, "--"+argName, value)
		}
	}

	return envVars, args
}
