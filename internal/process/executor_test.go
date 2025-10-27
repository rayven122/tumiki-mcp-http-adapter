package process

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"
)

func TestExecutor_Execute(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	tests := []struct {
		name        string
		command     string
		args        []string
		env         map[string]string
		input       []byte
		expectError bool
		validate    func(t *testing.T, output []byte)
	}{
		{
			name:        "シンプルなcatコマンド",
			command:     "cat",
			args:        []string{},
			env:         map[string]string{},
			input:       []byte("test input"),
			expectError: false,
			validate: func(t *testing.T, output []byte) {
				if !strings.Contains(string(output), "test input") {
					t.Errorf("Output should contain input: got %s", output)
				}
			},
		},
		{
			name:        "catコマンド_標準入力を読み取る",
			command:     "cat",
			args:        []string{},
			env:         map[string]string{},
			input:       []byte("test message"),
			expectError: false,
			validate: func(t *testing.T, output []byte) {
				if !strings.Contains(string(output), "test message") {
					t.Errorf("Output should contain input: got %s", output)
				}
			},
		},
		{
			name:        "環境変数を使用するコマンド",
			command:     "sh",
			args:        []string{"-c", "read line && echo \"$TEST_VAR:$line\""},
			env:         map[string]string{"TEST_VAR": "test-value"},
			input:       []byte("input-data"),
			expectError: false,
			validate: func(t *testing.T, output []byte) {
				expected := "test-value:input-data"
				if !strings.Contains(string(output), expected) {
					t.Errorf("Output should contain '%s': got %s", expected, output)
				}
			},
		},
		{
			name:        "存在しないコマンド",
			command:     "nonexistent-command-12345",
			args:        []string{},
			env:         map[string]string{},
			input:       []byte(""),
			expectError: true,
			validate:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor := NewExecutor(tt.command, tt.args, tt.env, logger)

			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()

			output, err := executor.Execute(ctx, tt.input)

			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.validate != nil {
				tt.validate(t, output)
			}
		})
	}
}

func TestExecutor_ContextCancellation(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	// 長時間実行されるコマンド
	executor := NewExecutor("sleep", []string{"10"}, map[string]string{}, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	start := time.Now()
	_, err := executor.Execute(ctx, []byte(""))
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Expected context cancellation error")
	}

	// タイムアウトが正しく動作したことを検証
	if elapsed > 2*time.Second {
		t.Errorf("Command should have been cancelled quickly, took %v", elapsed)
	}
}

func TestExecutor_MultipleEnvVars(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	env := map[string]string{
		"VAR1": "value1",
		"VAR2": "value2",
		"VAR3": "value3",
	}

	executor := NewExecutor("sh", []string{"-c", "read line && echo \"$VAR1 $VAR2 $VAR3:$line\""}, env, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	output, err := executor.Execute(ctx, []byte("input"))
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	result := string(output)
	if !strings.Contains(result, "value1") ||
		!strings.Contains(result, "value2") ||
		!strings.Contains(result, "value3") ||
		!strings.Contains(result, "input") {
		t.Errorf("Output should contain all env vars and input: got %s", result)
	}
}

func TestExecutor_envSlice(t *testing.T) {
	env := map[string]string{
		"KEY1": "value1",
		"KEY2": "value2",
	}

	executor := &Executor{
		env: env,
	}

	slice := executor.envSlice()

	if len(slice) != 2 {
		t.Errorf("envSlice() length = %d, want 2", len(slice))
	}

	// 各エントリが正しいフォーマットであることを検証
	found := make(map[string]bool)
	for _, entry := range slice {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) != 2 {
			t.Errorf("Invalid env entry format: %s", entry)
			continue
		}
		found[parts[0]] = true
		if val, ok := env[parts[0]]; !ok || val != parts[1] {
			t.Errorf("Env entry %s has wrong value", entry)
		}
	}

	if !found["KEY1"] || !found["KEY2"] {
		t.Error("Not all env vars were converted to slice")
	}
}

func TestNewExecutor(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	command := "test-command"
	args := []string{"arg1", "arg2"}
	env := map[string]string{"KEY": "value"}

	executor := NewExecutor(command, args, env, logger)

	if executor.command != command {
		t.Errorf("command = %s, want %s", executor.command, command)
	}

	if len(executor.args) != len(args) {
		t.Errorf("args length = %d, want %d", len(executor.args), len(args))
	}

	if executor.env["KEY"] != "value" {
		t.Error("env not properly set")
	}

	if executor.logger != logger {
		t.Error("logger not properly set")
	}
}
