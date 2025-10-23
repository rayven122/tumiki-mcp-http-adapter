package main

import (
	"reflect"
	"testing"
)

func TestParseMapping(t *testing.T) {
	tests := []struct {
		name      string
		mappings  ArrayFlags
		expected  map[string]string
		wantError bool
	}{
		{
			name: "単一のマッピング",
			mappings: ArrayFlags{
				"X-Slack-Token=SLACK_TOKEN",
			},
			expected: map[string]string{
				"X-Slack-Token": "SLACK_TOKEN",
			},
			wantError: false,
		},
		{
			name: "複数のマッピング",
			mappings: ArrayFlags{
				"X-Slack-Token=SLACK_TOKEN",
				"X-Team-Id=team-id",
				"Authorization=API_KEY",
			},
			expected: map[string]string{
				"X-Slack-Token": "SLACK_TOKEN",
				"X-Team-Id":     "team-id",
				"Authorization": "API_KEY",
			},
			wantError: false,
		},
		{
			name: "値にイコールを含む場合（エラー）",
			mappings: ArrayFlags{
				"Header=value=with=equals",
			},
			expected:  nil,
			wantError: true,
		},
		{
			name: "無効なフォーマット（イコールなし）",
			mappings: ArrayFlags{
				"InvalidMapping",
			},
			expected:  map[string]string{},
			wantError: false,
		},
		{
			name:      "空",
			mappings:  ArrayFlags{},
			expected:  map[string]string{},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseMapping(tt.mappings)
			if tt.wantError {
				if err == nil {
					t.Errorf("parseMapping() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("parseMapping() unexpected error: %v", err)
				}
				if !reflect.DeepEqual(result, tt.expected) {
					t.Errorf("parseMapping() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

func TestParseEnvVars(t *testing.T) {
	tests := []struct {
		name      string
		envVars   ArrayFlags
		expected  map[string]string
		wantError bool
	}{
		{
			name: "単一の環境変数",
			envVars: ArrayFlags{
				"KEY=value",
			},
			expected: map[string]string{
				"KEY": "value",
			},
			wantError: false,
		},
		{
			name: "複数の環境変数",
			envVars: ArrayFlags{
				"API_KEY=secret123",
				"DATABASE_URL=postgres://localhost/db",
				"LOG_LEVEL=debug",
			},
			expected: map[string]string{
				"API_KEY":      "secret123",
				"DATABASE_URL": "postgres://localhost/db",
				"LOG_LEVEL":    "debug",
			},
			wantError: false,
		},
		{
			name: "値にイコールを含む場合（エラー）",
			envVars: ArrayFlags{
				"KEY=value=with=equals",
			},
			expected:  nil,
			wantError: true,
		},
		{
			name: "無効なフォーマット（イコールなし）",
			envVars: ArrayFlags{
				"INVALID",
			},
			expected:  map[string]string{},
			wantError: false,
		},
		{
			name:      "空",
			envVars:   ArrayFlags{},
			expected:  map[string]string{},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseEnvVars(tt.envVars)
			if tt.wantError {
				if err == nil {
					t.Errorf("parseEnvVars() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("parseEnvVars() unexpected error: %v", err)
				}
				if !reflect.DeepEqual(result, tt.expected) {
					t.Errorf("parseEnvVars() = %v, want %v", result, tt.expected)
				}
			}
		})
	}
}

func TestParseStdioCommand(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		expected []string
	}{
		{
			name:     "シンプルなコマンド",
			command:  "echo hello",
			expected: []string{"echo", "hello"},
		},
		{
			name:     "複数の引数を持つコマンド",
			command:  "npx -y @modelcontextprotocol/server-filesystem /data",
			expected: []string{"npx", "-y", "@modelcontextprotocol/server-filesystem", "/data"},
		},
		{
			name:     "ダブルクォートで囲まれた引数",
			command:  `echo "hello world"`,
			expected: []string{"echo", "hello world"},
		},
		{
			name:     "シングルクォートで囲まれた引数",
			command:  `echo 'hello world'`,
			expected: []string{"echo", "hello world"},
		},
		{
			name:     "複雑なコマンド",
			command:  `sh -c "echo hello && echo world"`,
			expected: []string{"sh", "-c", "echo hello && echo world"},
		},
		{
			name:     "空のコマンド",
			command:  "",
			expected: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseStdioCommand(tt.command)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseStdioCommand() = %v, want %v", result, tt.expected)
			}
		})
	}
}
