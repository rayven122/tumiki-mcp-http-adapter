package main

import (
	"reflect"
	"testing"

	"github.com/rayven122/tumiki-mcp-http-adapter/internal/proxy"
)

func TestParseKeyValuePairs(t *testing.T) {
	tests := []struct {
		name      string
		pairs     ArrayFlags
		valueType string
		expected  map[string]string
		wantError bool
	}{
		{
			name: "単一のマッピング_正しくパースされる",
			pairs: ArrayFlags{
				"X-Slack-Token=SLACK_TOKEN",
			},
			valueType: "mapping",
			expected: map[string]string{
				"X-Slack-Token": "SLACK_TOKEN",
			},
			wantError: false,
		},
		{
			name: "複数のマッピング_全てパースされる",
			pairs: ArrayFlags{
				"X-Slack-Token=SLACK_TOKEN",
				"X-Team-Id=team-id",
				"Authorization=API_KEY",
			},
			valueType: "mapping",
			expected: map[string]string{
				"X-Slack-Token": "SLACK_TOKEN",
				"X-Team-Id":     "team-id",
				"Authorization": "API_KEY",
			},
			wantError: false,
		},
		{
			name: "単一の環境変数_マップに変換される",
			pairs: ArrayFlags{
				"KEY=value",
			},
			valueType: "environment variable",
			expected: map[string]string{
				"KEY": "value",
			},
			wantError: false,
		},
		{
			name: "複数の環境変数_全てマップに変換される",
			pairs: ArrayFlags{
				"API_KEY=secret123",
				"DATABASE_URL=postgres://localhost/db",
				"LOG_LEVEL=debug",
			},
			valueType: "environment variable",
			expected: map[string]string{
				"API_KEY":      "secret123",
				"DATABASE_URL": "postgres://localhost/db",
				"LOG_LEVEL":    "debug",
			},
			wantError: false,
		},
		{
			name: "値に=を含む場合_エラーを返す",
			pairs: ArrayFlags{
				"Header=value=with=equals",
			},
			valueType: "mapping",
			expected:  nil,
			wantError: true,
		},
		{
			name: "イコールなしの無効フォーマット_無視される",
			pairs: ArrayFlags{
				"InvalidMapping",
			},
			valueType: "mapping",
			expected:  map[string]string{},
			wantError: false,
		},
		{
			name:      "空の入力_空のマップを返す",
			pairs:     ArrayFlags{},
			valueType: "mapping",
			expected:  map[string]string{},
			wantError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseKeyValuePairs(tt.pairs, tt.valueType)
			if tt.wantError {
				if err == nil {
					t.Errorf("parseKeyValuePairs() expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("parseKeyValuePairs() unexpected error: %v", err)
				}
				if !reflect.DeepEqual(result, tt.expected) {
					t.Errorf("parseKeyValuePairs() = %v, want %v", result, tt.expected)
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
			name:     "シンプルなコマンド_正しくパースされる",
			command:  "echo hello",
			expected: []string{"echo", "hello"},
		},
		{
			name:     "複数の引数を持つコマンド_全て分割される",
			command:  "npx -y @modelcontextprotocol/server-filesystem /data",
			expected: []string{"npx", "-y", "@modelcontextprotocol/server-filesystem", "/data"},
		},
		{
			name:     "ダブルクォートで囲まれた引数_1つの要素として扱われる",
			command:  `echo "hello world"`,
			expected: []string{"echo", "hello world"},
		},
		{
			name:     "シングルクォートで囲まれた引数_1つの要素として扱われる",
			command:  `echo 'hello world'`,
			expected: []string{"echo", "hello world"},
		},
		{
			name:     "複雑なコマンド_正しくパースされる",
			command:  `sh -c "echo hello && echo world"`,
			expected: []string{"sh", "-c", "echo hello && echo world"},
		},
		{
			name:     "空のコマンド_空の配列を返す",
			command:  "",
			expected: []string{},
		},
		{
			name:     "ダブルクォート内にシングルクォート_そのまま保持される",
			command:  `echo "it's working"`,
			expected: []string{"echo", "it's working"},
		},
		{
			name:     "シングルクォート内にダブルクォート_そのまま保持される",
			command:  `echo 'say "hello"'`,
			expected: []string{"echo", `say "hello"`},
		},
		{
			name:     "複数のスペース_正しく分割される",
			command:  "echo  hello   world",
			expected: []string{"echo", "hello", "world"},
		},
		{
			name:     "先頭と末尾にスペース_トリムされる",
			command:  "  echo hello  ",
			expected: []string{"echo", "hello"},
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

func TestBuildConfigFromFlags(t *testing.T) {
	tests := []struct {
		name              string
		stdioCmd          string
		envVars           ArrayFlags
		headerEnvMappings ArrayFlags
		headerArgMappings ArrayFlags
		port              int
		expectedConfig    *proxy.Config
		expectPanic       bool
	}{
		{
			name:     "基本的な設定_正しくConfigが生成される",
			stdioCmd: "npx -y server-filesystem /data",
			envVars: ArrayFlags{
				"API_KEY=secret123",
			},
			headerEnvMappings: ArrayFlags{
				"X-Token=MY_TOKEN",
			},
			headerArgMappings: ArrayFlags{
				"X-Team-Id=team-id",
			},
			port: 8080,
			expectedConfig: &proxy.Config{
				Port:    8080,
				Command: "npx",
				Args:    []string{"-y", "server-filesystem", "/data"},
				DefaultEnv: map[string]string{
					"API_KEY": "secret123",
				},
				HeaderEnvMapping: map[string]string{
					"X-Token": "MY_TOKEN",
				},
				HeaderArgMapping: map[string]string{
					"X-Team-Id": "team-id",
				},
			},
			expectPanic: false,
		},
		{
			name:              "環境変数なしの設定_空のマップでConfigが生成される",
			stdioCmd:          "echo hello",
			envVars:           ArrayFlags{},
			headerEnvMappings: ArrayFlags{},
			headerArgMappings: ArrayFlags{},
			port:              9999,
			expectedConfig: &proxy.Config{
				Port:             9999,
				Command:          "echo",
				Args:             []string{"hello"},
				DefaultEnv:       map[string]string{},
				HeaderEnvMapping: map[string]string{},
				HeaderArgMapping: map[string]string{},
			},
			expectPanic: false,
		},
		{
			name:     "複数の環境変数とマッピング_全て正しく設定される",
			stdioCmd: "node server.js",
			envVars: ArrayFlags{
				"KEY1=value1",
				"KEY2=value2",
			},
			headerEnvMappings: ArrayFlags{
				"X-Header-1=ENV_VAR_1",
				"X-Header-2=ENV_VAR_2",
			},
			headerArgMappings: ArrayFlags{
				"X-Arg-1=arg-1",
				"X-Arg-2=arg-2",
			},
			port: 3000,
			expectedConfig: &proxy.Config{
				Port:    3000,
				Command: "node",
				Args:    []string{"server.js"},
				DefaultEnv: map[string]string{
					"KEY1": "value1",
					"KEY2": "value2",
				},
				HeaderEnvMapping: map[string]string{
					"X-Header-1": "ENV_VAR_1",
					"X-Header-2": "ENV_VAR_2",
				},
				HeaderArgMapping: map[string]string{
					"X-Arg-1": "arg-1",
					"X-Arg-2": "arg-2",
				},
			},
			expectPanic: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectPanic {
				defer func() {
					if r := recover(); r == nil {
						t.Errorf("buildConfigFromFlags() expected panic but didn't panic")
					}
				}()
			}

			result := buildConfigFromFlags(
				tt.stdioCmd,
				tt.envVars,
				tt.headerEnvMappings,
				tt.headerArgMappings,
				tt.port,
			)

			if !tt.expectPanic {
				if !reflect.DeepEqual(result, tt.expectedConfig) {
					t.Errorf("buildConfigFromFlags() = %+v, want %+v", result, tt.expectedConfig)
				}
			}
		})
	}
}

func TestArrayFlags_String(t *testing.T) {
	tests := []struct {
		name     string
		flags    ArrayFlags
		expected string
	}{
		{
			name:     "空のフラグ_空文字列を返す",
			flags:    ArrayFlags{},
			expected: "",
		},
		{
			name:     "単一のフラグ_そのまま返す",
			flags:    ArrayFlags{"value1"},
			expected: "value1",
		},
		{
			name:     "複数のフラグ_カンマ区切りで返す",
			flags:    ArrayFlags{"value1", "value2", "value3"},
			expected: "value1,value2,value3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.flags.String()
			if result != tt.expected {
				t.Errorf("String() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestArrayFlags_Set(t *testing.T) {
	tests := []struct {
		name     string
		initial  ArrayFlags
		setValue string
		expected ArrayFlags
	}{
		{
			name:     "空のフラグに値を追加_配列に追加される",
			initial:  ArrayFlags{},
			setValue: "value1",
			expected: ArrayFlags{"value1"},
		},
		{
			name:     "既存のフラグに値を追加_配列の末尾に追加される",
			initial:  ArrayFlags{"existing"},
			setValue: "new",
			expected: ArrayFlags{"existing", "new"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags := tt.initial
			err := flags.Set(tt.setValue)
			if err != nil {
				t.Errorf("Set() unexpected error: %v", err)
			}
			if !reflect.DeepEqual(flags, tt.expected) {
				t.Errorf("Set() = %v, want %v", flags, tt.expected)
			}
		})
	}
}
