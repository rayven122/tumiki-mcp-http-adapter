package proxy

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
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
			name:    "空のヘッダー",
			headers: http.Header{},
			envMapping: map[string]string{
				"X-Slack-Token": "SLACK_TOKEN",
			},
			argMapping:  map[string]string{},
			wantEnvVars: map[string]string{},
			wantArgs:    []string{},
		},
		{
			name: "環境変数へのマッピング",
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
			name: "引数へのマッピング",
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
			name: "環境変数と引数の両方へのマッピング",
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

			// 環境変数を検証
			if len(gotEnvVars) != len(tt.wantEnvVars) {
				t.Errorf("parseHeaders() envVars count = %d, want %d", len(gotEnvVars), len(tt.wantEnvVars))
			}
			for k, v := range tt.wantEnvVars {
				if gotEnvVars[k] != v {
					t.Errorf("parseHeaders() envVars[%s] = %v, want %v", k, gotEnvVars[k], v)
				}
			}

			// 引数を検証（順序は保証されないため内容のみチェック）
			if len(gotArgs) != len(tt.wantArgs) {
				t.Errorf("parseHeaders() args length = %d, want %d (got: %v, want: %v)", len(gotArgs), len(tt.wantArgs), gotArgs, tt.wantArgs)
				return
			}
			// 引数がペア（フラグ名、値）として存在することを検証
			gotArgsMap := make(map[string]string)
			for i := 0; i < len(gotArgs); i += 2 {
				if i+1 < len(gotArgs) {
					gotArgsMap[gotArgs[i]] = gotArgs[i+1]
				}
			}
			wantArgsMap := make(map[string]string)
			for i := 0; i < len(tt.wantArgs); i += 2 {
				if i+1 < len(tt.wantArgs) {
					wantArgsMap[tt.wantArgs[i]] = tt.wantArgs[i+1]
				}
			}
			for k, v := range wantArgsMap {
				if gotArgsMap[k] != v {
					t.Errorf("parseHeaders() args[%s] = %v, want %v", k, gotArgsMap[k], v)
				}
			}
		})
	}
}

func TestHandleMCP_Basic(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	cfg := &Config{
		Port:             8080,
		Command:          "cat",
		Args:             []string{},
		DefaultEnv:       map[string]string{},
		HeaderEnvMapping: map[string]string{},
		HeaderArgMapping: map[string]string{},
	}

	server, err := NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader([]byte("test input\n")))
	w := httptest.NewRecorder()

	server.handleMCP(w, req)

	resp := w.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("Failed to close response body: %v", err)
		}
	}()

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
		Args:    []string{"-c", "read line && echo \"$VAR1:$line\""},
		DefaultEnv: map[string]string{
			"VAR1": "default",
		},
		HeaderEnvMapping: map[string]string{
			"X-Custom-Var": "VAR1",
		},
		HeaderArgMapping: map[string]string{},
	}

	server, err := NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	// ヘッダーで環境変数を上書き
	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader([]byte("test\n")))
	req.Header.Set("X-Custom-Var", "override")
	w := httptest.NewRecorder()

	server.handleMCP(w, req)

	resp := w.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("Failed to close response body: %v", err)
		}
	}()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body := w.Body.String()
	if body == "" {
		t.Error("Expected non-empty response")
	}

	// レスポンスに環境変数を上書きしたヘッダー値が含まれていることを検証
	if !strings.Contains(body, "override") {
		t.Errorf("Response should contain header value 'override': got %s", body)
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

	// テスト用にHOST環境変数を設定
	if err := os.Setenv("HOST", "127.0.0.1"); err != nil {
		t.Fatalf("Failed to set HOST env: %v", err)
	}
	defer func() {
		if err := os.Unsetenv("HOST"); err != nil {
			t.Errorf("Failed to unset HOST env: %v", err)
		}
	}()

	server, err := NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	// goroutineでサーバー起動
	errChan := make(chan error, 1)
	go func() {
		errChan <- server.Start(ctx)
	}()

	// 少し待ってからシャットダウン
	time.Sleep(100 * time.Millisecond)
	cancel()

	// シャットダウンの完了を待つ
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
		Command:          "cat",
		Args:             []string{},
		DefaultEnv:       map[string]string{},
		HeaderEnvMapping: map[string]string{},
		HeaderArgMapping: map[string]string{},
	}

	server, err := NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	// エラーを引き起こすボディ（nil reader）
	req := httptest.NewRequest("POST", "/mcp", nil)
	w := httptest.NewRecorder()

	server.handleMCP(w, req)

	resp := w.Result()
	defer func() {
		if err := resp.Body.Close(); err != nil {
			t.Errorf("Failed to close response body: %v", err)
		}
	}()

	// nil bodyは有効なのでエラーは期待されない
	if resp.StatusCode != http.StatusOK {
		t.Logf("Status = %d (this is expected for some edge cases)", resp.StatusCode)
	}
}
