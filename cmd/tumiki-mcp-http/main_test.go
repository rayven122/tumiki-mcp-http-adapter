package main

import (
	"reflect"
	"testing"
)

func TestParseMapping(t *testing.T) {
	tests := []struct {
		name     string
		mappings ArrayFlags
		expected map[string]string
	}{
		{
			name: "single mapping",
			mappings: ArrayFlags{
				"X-Slack-Token=SLACK_TOKEN",
			},
			expected: map[string]string{
				"X-Slack-Token": "SLACK_TOKEN",
			},
		},
		{
			name: "multiple mappings",
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
		},
		{
			name: "value with equals sign",
			mappings: ArrayFlags{
				"Header=value=with=equals",
			},
			expected: map[string]string{
				"Header": "value=with=equals",
			},
		},
		{
			name: "invalid format (no equals)",
			mappings: ArrayFlags{
				"InvalidMapping",
			},
			expected: map[string]string{},
		},
		{
			name:     "empty",
			mappings: ArrayFlags{},
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseMapping(tt.mappings)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseMapping() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestParseEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		envVars  ArrayFlags
		expected map[string]string
	}{
		{
			name: "single env var",
			envVars: ArrayFlags{
				"KEY=value",
			},
			expected: map[string]string{
				"KEY": "value",
			},
		},
		{
			name: "multiple env vars",
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
		},
		{
			name: "value with equals sign",
			envVars: ArrayFlags{
				"KEY=value=with=equals",
			},
			expected: map[string]string{
				"KEY": "value=with=equals",
			},
		},
		{
			name: "invalid format (no equals)",
			envVars: ArrayFlags{
				"INVALID",
			},
			expected: map[string]string{},
		},
		{
			name:     "empty",
			envVars:  ArrayFlags{},
			expected: map[string]string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseEnvVars(tt.envVars)
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("parseEnvVars() = %v, want %v", result, tt.expected)
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
			name:     "simple command",
			command:  "echo hello",
			expected: []string{"echo", "hello"},
		},
		{
			name:     "command with multiple args",
			command:  "npx -y @modelcontextprotocol/server-filesystem /data",
			expected: []string{"npx", "-y", "@modelcontextprotocol/server-filesystem", "/data"},
		},
		{
			name:     "command with quoted args",
			command:  `echo "hello world"`,
			expected: []string{"echo", "hello world"},
		},
		{
			name:     "command with single quotes",
			command:  `echo 'hello world'`,
			expected: []string{"echo", "hello world"},
		},
		{
			name:     "complex command",
			command:  `sh -c "echo hello && echo world"`,
			expected: []string{"sh", "-c", "echo hello && echo world"},
		},
		{
			name:     "empty command",
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
