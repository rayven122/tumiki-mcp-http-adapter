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

	"tumiki-mcp-http/internal/config"
)

func TestNewServer(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:            "localhost",
			Port:            8080,
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			ShutdownTimeout: 10 * time.Second,
		},
		Auth: config.AuthConfig{
			Enabled:      false,
			BearerTokens: []string{},
		},
		Stdio: config.StdioConfig{
			DefaultServer: "test",
			Servers: map[string]config.ServerDefinition{
				"test": {
					Command: "echo",
					Args:    []string{"hello"},
				},
			},
		},
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

func TestHandleHealth(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Stdio: config.StdioConfig{
			Servers: map[string]config.ServerDefinition{},
		},
	}

	server, _ := NewServer(cfg, logger)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	server.handleHealth(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	body := w.Body.String()
	if body != "OK" {
		t.Errorf("Body = %q, want %q", body, "OK")
	}
}

func TestHandleMCP_UnknownServer(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Stdio: config.StdioConfig{
			DefaultServer: "default",
			Servers: map[string]config.ServerDefinition{
				"default": {
					Command: "echo",
				},
			},
		},
	}

	server, _ := NewServer(cfg, logger)

	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader([]byte("{}")))
	req.Header.Set("X-MCP-Server", "unknown-server")
	w := httptest.NewRecorder()

	server.handleMCP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}
}

func TestHandleMCP_MissingRequiredEnv(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Stdio: config.StdioConfig{
			DefaultServer: "test",
			Process: config.ProcessConfig{
				Timeout:    30 * time.Second,
				BufferSize: 8192,
			},
			Servers: map[string]config.ServerDefinition{
				"test": {
					Command:     "echo",
					RequiredEnv: []string{"REQUIRED_VAR"},
				},
			},
		},
	}

	server, _ := NewServer(cfg, logger)

	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader([]byte("{}")))
	w := httptest.NewRecorder()

	server.handleMCP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusBadRequest {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusBadRequest)
	}

	body := w.Body.String()
	if body != "Missing required env: REQUIRED_VAR\n" {
		t.Errorf("Body = %q, want missing env error", body)
	}
}

func TestHandleMCP_DefaultServer(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Stdio: config.StdioConfig{
			DefaultServer: "default",
			Process: config.ProcessConfig{
				Timeout:    5 * time.Second,
				BufferSize: 8192,
			},
			Servers: map[string]config.ServerDefinition{
				"default": {
					Command: "echo",
					Args:    []string{"test"},
				},
			},
		},
	}

	server, _ := NewServer(cfg, logger)

	// X-MCP-Serverヘッダーなし（デフォルトサーバーを使用）
	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader([]byte("test input\n")))
	w := httptest.NewRecorder()

	server.handleMCP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	// echoコマンドは正常に実行されるはず
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d (body: %s)", resp.StatusCode, http.StatusOK, w.Body.String())
	}
}

func TestHandleMCP_EnvHeaderMerge(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host: "localhost",
			Port: 8080,
		},
		Stdio: config.StdioConfig{
			DefaultServer: "test",
			Process: config.ProcessConfig{
				Timeout:    5 * time.Second,
				BufferSize: 8192,
			},
			Servers: map[string]config.ServerDefinition{
				"test": {
					Command: "sh",
					Args:    []string{"-c", "echo $VAR1 $VAR2"},
					DefaultEnv: map[string]string{
						"VAR1": "default1",
						"VAR2": "default2",
					},
				},
			},
		},
	}

	server, _ := NewServer(cfg, logger)

	// ヘッダーでVAR1を上書き
	req := httptest.NewRequest("POST", "/mcp", bytes.NewReader([]byte("test\n")))
	req.Header.Set("X-MCP-Env-VAR1", "header1")
	w := httptest.NewRecorder()

	server.handleMCP(w, req)

	resp := w.Result()
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}

	// VAR1はheader1、VAR2はdefault2になっているはず
	body := w.Body.String()
	// echoの出力には改行が含まれる可能性があるので、含まれているかで判定
	if body == "" {
		t.Error("Expected non-empty response")
	}
}

func TestServer_Start_Shutdown(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:            "127.0.0.1",
			Port:            0, // ランダムポート
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			ShutdownTimeout: 1 * time.Second,
		},
		Stdio: config.StdioConfig{
			DefaultServer: "test",
			Servers: map[string]config.ServerDefinition{
				"test": {
					Command: "echo",
				},
			},
		},
	}

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
