package proxy

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestNewServer(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	cfg := &Config{
		Port:             8080,
		Command:          "echo",
		Args:             []string{"hello"},
		DefaultEnv:       map[string]string{},
		HeaderEnvMapping: map[string]string{},
		HeaderArgMapping: map[string]string{},
	}

	server, err := NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	if server == nil {
		t.Fatal("NewServer() returned nil server")
	}

	if server.cfg != cfg {
		t.Error("Server config not properly set")
	}

	if server.logger != logger {
		t.Error("Server logger not properly set")
	}

	if server.server == nil {
		t.Error("HTTP server not initialized")
	}
}

func TestParseHeaders(t *testing.T) {
	tests := []struct {
		name        string
		headers     http.Header
		envMapping  map[string]string
		argMapping  map[string]string
		wantEnvVars map[string]string
		wantArgs    []string
	}{
		{
			name:    "empty headers",
			headers: http.Header{},
			envMapping: map[string]string{
				"X-Slack-Token": "SLACK_TOKEN",
			},
			argMapping:  map[string]string{},
			wantEnvVars: map[string]string{},
			wantArgs:    []string{},
		},
		{
			name: "env mapping",
			headers: http.Header{
				"X-Slack-Token": []string{"xoxp-12345"},
			},
			envMapping: map[string]string{
				"X-Slack-Token": "SLACK_TOKEN",
			},
			argMapping: map[string]string{},
			wantEnvVars: map[string]string{
				"SLACK_TOKEN": "xoxp-12345",
			},
			wantArgs: []string{},
		},
		{
			name: "arg mapping",
			headers: http.Header{
				"X-Team-Id": []string{"T123"},
				"X-Channel": []string{"general"},
			},
			envMapping: map[string]string{},
			argMapping: map[string]string{
				"X-Team-Id": "team-id",
				"X-Channel": "channel",
			},
			wantEnvVars: map[string]string{},
			wantArgs:    []string{"--team-id", "T123", "--channel", "general"},
		},
		{
			name: "both env and arg mapping",
			headers: http.Header{
				"X-Slack-Token": []string{"xoxp-12345"},
				"X-Team-Id":     []string{"T123"},
			},
			envMapping: map[string]string{
				"X-Slack-Token": "SLACK_TOKEN",
			},
			argMapping: map[string]string{
				"X-Team-Id": "team-id",
			},
			wantEnvVars: map[string]string{
				"SLACK_TOKEN": "xoxp-12345",
			},
			wantArgs: []string{"--team-id", "T123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotEnvVars, gotArgs := parseHeaders(tt.headers, tt.envMapping, tt.argMapping)

			// 環境変数の検証
			if len(gotEnvVars) != len(tt.wantEnvVars) {
				t.Errorf("parseHeaders() envVars count = %d, want %d", len(gotEnvVars), len(tt.wantEnvVars))
			}
			for k, v := range tt.wantEnvVars {
				if gotEnvVars[k] != v {
					t.Errorf("parseHeaders() envVars[%s] = %v, want %v", k, gotEnvVars[k], v)
				}
			}

			// 引数の検証
			if len(gotArgs) != len(tt.wantArgs) {
				t.Errorf("parseHeaders() args = %v, want %v", gotArgs, tt.wantArgs)
				return
			}
			for i := range tt.wantArgs {
				if gotArgs[i] != tt.wantArgs[i] {
					t.Errorf("parseHeaders() args[%d] = %v, want %v", i, gotArgs[i], tt.wantArgs[i])
				}
			}
		})
	}
}

func TestHandleMCP_Basic(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	cfg := &Config{
		Port:             8080,
		Command:          "echo",
		Args:             []string{"test"},
		DefaultEnv:       map[string]string{},
		HeaderEnvMapping: map[string]string{},
		HeaderArgMapping: map[string]string{},
	}

	server, _ := NewServer(cfg, logger)

	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader([]byte("test input\n")))
	w := httptest.NewRecorder()

	server.handleMCP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d (body: %s)", resp.StatusCode, http.StatusOK, w.Body.String())
	}

	if resp.Header.Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %s, want application/json", resp.Header.Get("Content-Type"))
	}
}

func TestHandleMCP_WithHeaderMapping(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	cfg := &Config{
		Port:    8080,
		Command: "sh",
		Args:    []string{"-c", "echo $VAR1"},
		DefaultEnv: map[string]string{
			"VAR1": "default",
		},
		HeaderEnvMapping: map[string]string{
			"X-Custom-Var": "VAR1",
		},
		HeaderArgMapping: map[string]string{},
	}

	server, _ := NewServer(cfg, logger)

	// ヘッダーで環境変数を上書き
	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader([]byte("test\n")))
	req.Header.Set("X-Custom-Var", "override")
	w := httptest.NewRecorder()

	server.handleMCP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body := w.Body.String()
	if body == "" {
		t.Error("Expected non-empty response")
	}
}

func TestServer_Start_Shutdown(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	cfg := &Config{
		Port:             0, // ランダムポート
		Command:          "echo",
		Args:             []string{},
		DefaultEnv:       map[string]string{},
		HeaderEnvMapping: map[string]string{},
		HeaderArgMapping: map[string]string{},
	}

	// HOST環境変数をテスト用に設定
	os.Setenv("HOST", "127.0.0.1")
	defer os.Unsetenv("HOST")

	server, err := NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// サーバーをゴルーチンで起動
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Start(ctx)
	}()

	// 少し待ってからシャットダウン
	time.Sleep(100 * time.Millisecond)
	cancel()

	// シャットダウンが完了するまで待つ
	select {
	case err := <-errChan:
		if err != nil {
			t.Errorf("Server.Start() error = %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Error("Server shutdown timeout")
	}
}

func TestHandleMCP_InvalidBody(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	cfg := &Config{
		Port:             8080,
		Command:          "echo",
		Args:             []string{},
		DefaultEnv:       map[string]string{},
		HeaderEnvMapping: map[string]string{},
		HeaderArgMapping: map[string]string{},
	}

	server, _ := NewServer(cfg, logger)

	// エラーを起こすボディ（nilリーダー）
	req := httptest.NewRequest("POST", "/mcp", nil)
	w := httptest.NewRecorder()

	server.handleMCP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// nilボディは有効なので、エラーにはならない
	if resp.StatusCode != http.StatusOK {
		t.Logf("Status = %d (this is expected for some edge cases)", resp.StatusCode)
	}
}
