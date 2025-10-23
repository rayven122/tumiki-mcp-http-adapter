package headers

import (
	"net/http"
	"reflect"
	"testing"
)

func TestParseEnvHeaders(t *testing.T) {
	tests := []struct {
		name     string
		headers  http.Header
		expected map[string]string
	}{
		{
			name: "single env var",
			headers: http.Header{
				"X-Mcp-Env-Api-Key": []string{"secret-key-123"},
			},
			expected: map[string]string{
				"API_KEY": "secret-key-123",
			},
		},
		{
			name: "multiple env vars",
			headers: http.Header{
				"X-Mcp-Env-Github-Token": []string{"ghp_token123"},
				"X-Mcp-Env-Database-Url": []string{"postgres://localhost/db"},
				"X-Mcp-Env-Log-Level":    []string{"debug"},
			},
			expected: map[string]string{
				"GITHUB_TOKEN": "ghp_token123",
				"DATABASE_URL": "postgres://localhost/db",
				"LOG_LEVEL":    "debug",
			},
		},
		{
			name: "mixed headers (only X-MCP-Env extracted)",
			headers: http.Header{
				"X-Mcp-Env-Api-Key": []string{"key123"},
				"Authorization":     []string{"Bearer token"},
				"Content-Type":      []string{"application/json"},
			},
			expected: map[string]string{
				"API_KEY": "key123",
			},
		},
		{
			name:     "no env headers",
			headers:  http.Header{},
			expected: map[string]string{},
		},
		{
			name: "case insensitive header matching",
			headers: http.Header{
				"x-mcp-env-api-key": []string{"lowercase"},
				"X-MCP-ENV-DB-URL":  []string{"uppercase"},
			},
			expected: map[string]string{
				"API_KEY": "lowercase",
				"DB_URL":  "uppercase",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseEnvHeaders(tt.headers)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParseEnvHeaders() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseArgsHeaders(t *testing.T) {
	tests := []struct {
		name         string
		headers      http.Header
		argsTemplate []string
		expected     []string
	}{
		{
			name: "JSON array args",
			headers: http.Header{
				"X-Mcp-Args": []string{`["--verbose", "--output", "json"]`},
			},
			argsTemplate: nil,
			expected:     []string{"--verbose", "--output", "json"},
		},
		{
			name: "template-based args",
			headers: http.Header{
				"X-Mcp-Arg-Team-Id": []string{"team-123"},
				"X-Mcp-Arg-Channel": []string{"general"},
			},
			argsTemplate: []string{"--team", "{{.TEAM_ID}}", "--channel", "{{.CHANNEL}}"},
			expected:     []string{"--team", "team-123", "--channel", "general"},
		},
		{
			name: "mixed case template variables",
			headers: http.Header{
				"X-Mcp-Arg-User-Id": []string{"user-456"},
			},
			argsTemplate: []string{"--user-id", "{{.USER_ID}}"},
			expected:     []string{"--user-id", "user-456"},
		},
		{
			name:         "no args headers",
			headers:      http.Header{},
			argsTemplate: nil,
			expected:     []string{},
		},
		{
			name: "invalid JSON array (ignored)",
			headers: http.Header{
				"X-Mcp-Args": []string{`invalid json`},
			},
			argsTemplate: nil,
			expected:     []string{},
		},
		{
			name: "template with missing variable (kept as-is)",
			headers: http.Header{
				"X-Mcp-Arg-Existing": []string{"value"},
			},
			argsTemplate: []string{"--existing", "{{.EXISTING}}", "--missing", "{{.MISSING}}"},
			expected:     []string{"--existing", "value", "--missing", "{{.MISSING}}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseArgsHeaders(tt.headers, tt.argsTemplate)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("ParseArgsHeaders() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestApplyArgsTemplate(t *testing.T) {
	tests := []struct {
		name      string
		templates []string
		data      map[string]string
		expected  []string
	}{
		{
			name:      "simple replacement",
			templates: []string{"--key", "{{.API_KEY}}"},
			data: map[string]string{
				"API_KEY": "secret123",
			},
			expected: []string{"--key", "secret123"},
		},
		{
			name:      "multiple replacements",
			templates: []string{"--host", "{{.HOST}}", "--port", "{{.PORT}}"},
			data: map[string]string{
				"HOST": "localhost",
				"PORT": "8080",
			},
			expected: []string{"--host", "localhost", "--port", "8080"},
		},
		{
			name:      "no templates",
			templates: []string{"--static", "value"},
			data:      map[string]string{},
			expected:  []string{"--static", "value"},
		},
		{
			name:      "empty data",
			templates: []string{"--key", "{{.MISSING}}"},
			data:      map[string]string{},
			expected:  []string{"--key", "{{.MISSING}}"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := applyArgsTemplate(tt.templates, tt.data)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("applyArgsTemplate() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestHeaderKeyConversion(t *testing.T) {
	tests := []struct {
		headerKey string
		expected  string
	}{
		{"X-Mcp-Env-Api-Key", "API_KEY"},
		{"X-Mcp-Env-Database-Url", "DATABASE_URL"},
		{"X-Mcp-Env-Single", "SINGLE"},
		{"X-Mcp-Env-Multi-Word-Key", "MULTI_WORD_KEY"},
		{"x-mcp-env-lowercase", "LOWERCASE"},
	}

	for _, tt := range tests {
		t.Run(tt.headerKey, func(t *testing.T) {
			headers := http.Header{
				tt.headerKey: []string{"value"},
			}
			result := ParseEnvHeaders(headers)
			if _, ok := result[tt.expected]; !ok {
				t.Errorf("Expected key %s not found in result %v", tt.expected, result)
			}
		})
	}
}

func TestParseCustomHeaders(t *testing.T) {
	tests := []struct {
		name         string
		headers      http.Header
		envMapping   map[string]string
		argMapping   map[string]string
		expectedEnv  map[string]string
		expectedArgs []string
	}{
		{
			name: "single env mapping",
			headers: http.Header{
				"X-Slack-Token": []string{"xoxp-token123"},
			},
			envMapping: map[string]string{
				"X-Slack-Token": "SLACK_TOKEN",
			},
			argMapping: map[string]string{},
			expectedEnv: map[string]string{
				"SLACK_TOKEN": "xoxp-token123",
			},
			expectedArgs: []string{},
		},
		{
			name: "single arg mapping",
			headers: http.Header{
				"X-Team-Id": []string{"T123"},
			},
			envMapping: map[string]string{},
			argMapping: map[string]string{
				"X-Team-Id": "team-id",
			},
			expectedEnv: map[string]string{},
			expectedArgs: []string{"--team-id", "T123"},
		},
		{
			name: "mixed env and arg mappings",
			headers: http.Header{
				"X-Slack-Token": []string{"xoxp-token"},
				"X-Team-Id":     []string{"T123"},
				"X-Channel":     []string{"general"},
			},
			envMapping: map[string]string{
				"X-Slack-Token": "SLACK_TOKEN",
			},
			argMapping: map[string]string{
				"X-Team-Id": "team-id",
				"X-Channel": "channel",
			},
			expectedEnv: map[string]string{
				"SLACK_TOKEN": "xoxp-token",
			},
			expectedArgs: []string{"--team-id", "T123", "--channel", "general"},
		},
		{
			name: "completely custom header names",
			headers: http.Header{
				"Authorization": []string{"Bearer secret123"},
				"My-Custom-Id":  []string{"custom-value"},
			},
			envMapping: map[string]string{
				"Authorization": "API_KEY",
			},
			argMapping: map[string]string{
				"My-Custom-Id": "custom-id",
			},
			expectedEnv: map[string]string{
				"API_KEY": "Bearer secret123",
			},
			expectedArgs: []string{"--custom-id", "custom-value"},
		},
		{
			name: "header not in mapping (ignored)",
			headers: http.Header{
				"X-Slack-Token": []string{"xoxp-token"},
				"X-Unknown":     []string{"value"},
			},
			envMapping: map[string]string{
				"X-Slack-Token": "SLACK_TOKEN",
			},
			argMapping: map[string]string{},
			expectedEnv: map[string]string{
				"SLACK_TOKEN": "xoxp-token",
			},
			expectedArgs: []string{},
		},
		{
			name: "empty headers",
			headers: http.Header{},
			envMapping: map[string]string{
				"X-Slack-Token": "SLACK_TOKEN",
			},
			argMapping: map[string]string{
				"X-Team-Id": "team-id",
			},
			expectedEnv:  map[string]string{},
			expectedArgs: []string{},
		},
		{
			name: "empty mappings",
			headers: http.Header{
				"X-Slack-Token": []string{"xoxp-token"},
			},
			envMapping:   map[string]string{},
			argMapping:   map[string]string{},
			expectedEnv:  map[string]string{},
			expectedArgs: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			env, args := ParseCustomHeaders(tt.headers, tt.envMapping, tt.argMapping)

			if !reflect.DeepEqual(env, tt.expectedEnv) {
				t.Errorf("ParseCustomHeaders() env = %v, want %v", env, tt.expectedEnv)
			}

			if !reflect.DeepEqual(args, tt.expectedArgs) {
				t.Errorf("ParseCustomHeaders() args = %v, want %v", args, tt.expectedArgs)
			}
		})
	}
}
