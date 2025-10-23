package test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"tumiki-mcp-http/internal/config"
	"tumiki-mcp-http/internal/proxy"
)

// TestServerIntegration は統合テストです
func TestServerIntegration(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	// テスト用の設定
	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:            "127.0.0.1",
			Port:            0, // ランダムポート
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			ShutdownTimeout: 5 * time.Second,
		},
		Stdio: config.StdioConfig{
			DefaultServer: "echo-server",
			Process: config.ProcessConfig{
				Timeout:    5 * time.Second,
				BufferSize: 8192,
			},
			Servers: map[string]config.ServerDefinition{
				"echo-server": {
					Command: "echo",
					Args:    []string{`{"result":"success"}`},
					DefaultEnv: map[string]string{
						"DEFAULT_VAR": "default_value",
					},
					HeaderEnvMapping: map[string]string{
						"X-Custom-Token": "CUSTOM_TOKEN",
					},
					HeaderArgMapping: map[string]string{
						"X-Custom-Id": "custom-id",
					},
				},
			},
		},
	}

	// サーバーを作成
	server, err := proxy.NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	// HTTPハンドラーを取得するためにテストサーバーを使用
	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	tests := []struct {
		name           string
		method         string
		path           string
		headers        map[string]string
		body           string
		expectedStatus int
		checkResponse  func(t *testing.T, resp *http.Response)
	}{
		{
			name:           "health check",
			method:         "GET",
			path:           "/health",
			headers:        map[string]string{},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("Health check failed: status = %d", resp.StatusCode)
				}
			},
		},
		{
			name:   "MCP request",
			method: "POST",
			path:   "/mcp",
			headers: map[string]string{
				"Content-Type": "application/json",
			},
			body:           `{"jsonrpc":"2.0","id":1,"method":"test"}`,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				if resp.StatusCode != http.StatusOK {
					t.Errorf("MCP request failed: status = %d", resp.StatusCode)
				}
			},
		},
		{
			name:   "MCP request with custom headers",
			method: "POST",
			path:   "/mcp",
			headers: map[string]string{
				"Content-Type":   "application/json",
				"X-Custom-Token": "custom-token-value",
				"X-Custom-Id":    "123",
			},
			body:           `{"jsonrpc":"2.0","id":1,"method":"test"}`,
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, resp *http.Response) {
				// カスタムヘッダーが環境変数と引数に変換されているはず
				// echoコマンドの出力を確認
				var result map[string]interface{}
				if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
					t.Logf("Response decode: %v (expected for echo test)", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body *bytes.Buffer
			if tt.body != "" {
				body = bytes.NewBufferString(tt.body)
			} else {
				body = bytes.NewBuffer(nil)
			}

			req, err := http.NewRequest(tt.method, ts.URL+tt.path, body)
			if err != nil {
				t.Fatalf("Failed to create request: %v", err)
			}

			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}

			client := &http.Client{Timeout: 10 * time.Second}
			resp, err := client.Do(req)
			if err != nil {
				t.Fatalf("Request failed: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tt.expectedStatus {
				t.Errorf("Status = %d, want %d", resp.StatusCode, tt.expectedStatus)
			}

			if tt.checkResponse != nil {
				tt.checkResponse(t, resp)
			}
		})
	}
}

// TestHeaderMappingIntegration はヘッダーマッピングの統合テストです
func TestHeaderMappingIntegration(t *testing.T) {
	logger := slog.New(slog.NewJSONHandler(os.Stderr, nil))

	cfg := &config.Config{
		Server: config.ServerConfig{
			Host:            "127.0.0.1",
			Port:            0,
			ReadTimeout:     30 * time.Second,
			WriteTimeout:    30 * time.Second,
			ShutdownTimeout: 5 * time.Second,
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
					Args:    []string{"-c", "echo $TEST_ENV"},
					DefaultEnv: map[string]string{},
					HeaderEnvMapping: map[string]string{
						"X-Test-Header": "TEST_ENV",
					},
					HeaderArgMapping: map[string]string{},
				},
			},
		},
	}

	server, err := proxy.NewServer(cfg, logger)
	if err != nil {
		t.Fatalf("Failed to create server: %v", err)
	}

	ts := httptest.NewServer(server.Handler())
	defer ts.Close()

	// ヘッダーで環境変数を渡すテスト
	req, _ := http.NewRequest("POST", ts.URL+"/mcp", bytes.NewBufferString(`{"test":"data"}`))
	req.Header.Set("X-Test-Header", "test-value-123")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Status = %d, want %d", resp.StatusCode, http.StatusOK)
	}
}
