// Package proxy provides HTTP proxy server functionality for stdio-based MCP servers.
package proxy

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"tumiki-mcp-http/internal/process"
)

// タイムアウト設定は定数として定義
const (
	ReadTimeout     = 30 * time.Second
	WriteTimeout    = 30 * time.Second
	ShutdownTimeout = 5 * time.Second
	ProcessTimeout  = 30 * time.Second
)

// Config - 最小限の設定構造体
type Config struct {
	Port             int               // サーバーポート（必須）
	Command          string            // stdio コマンド（必須）
	Args             []string          // コマンド引数
	DefaultEnv       map[string]string // デフォルト環境変数
	HeaderEnvMapping map[string]string // ヘッダー→環境変数マッピング
	HeaderArgMapping map[string]string // ヘッダー→引数マッピング
}

// Server is an HTTP proxy server that forwards requests to stdio-based MCP servers.
type Server struct {
	cfg    *Config
	logger *slog.Logger
	server *http.Server
}

// NewServer creates a new Server with the specified configuration and logger.
func NewServer(cfg *Config, logger *slog.Logger) (*Server, error) {
	s := &Server{
		cfg:    cfg,
		logger: logger,
	}

	mux := http.NewServeMux()

	// MCP エンドポイント（/mcp のみ）
	mux.HandleFunc("/mcp", s.handleMCP)

	// ホスト設定は環境変数 HOST から取得（デフォルト: 0.0.0.0）
	host := os.Getenv("HOST")
	if host == "" {
		host = "0.0.0.0"
	}

	s.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", host, cfg.Port),
		Handler:      mux,
		ReadTimeout:  ReadTimeout,
		WriteTimeout: WriteTimeout,
	}

	return s, nil
}

func (s *Server) handleMCP(w http.ResponseWriter, r *http.Request) {
	// 1. ヘッダー解析（カスタムマッピング使用）
	envVars := make(map[string]string)

	// デフォルト環境変数
	for k, v := range s.cfg.DefaultEnv {
		envVars[k] = v
	}

	// カスタムヘッダーマッピングを使用してヘッダーを解析
	headerEnv, headerArgs := parseHeaders(
		r.Header,
		s.cfg.HeaderEnvMapping,
		s.cfg.HeaderArgMapping,
	)

	// ヘッダーから取得した環境変数（デフォルトを上書き）
	for k, v := range headerEnv {
		envVars[k] = v
	}

	// 2. 引数マージ（元のスライスを変更しない）
	args := make([]string, 0, len(s.cfg.Args)+len(headerArgs))
	args = append(args, s.cfg.Args...)
	args = append(args, headerArgs...)

	// 3. リクエストボディ読み込み
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer func() {
		if err := r.Body.Close(); err != nil && s.logger != nil {
			s.logger.Debug("Failed to close request body", "error", err)
		}
	}()

	// 4. stdio プロセス実行
	ctx, cancel := context.WithTimeout(r.Context(), ProcessTimeout)
	defer cancel()

	executor := process.NewExecutor(
		s.cfg.Command,
		args,
		envVars,
		s.logger,
	)

	response, err := executor.Execute(ctx, body)
	if err != nil {
		s.logger.Error("Process execution failed", "error", err)
		http.Error(w, "Process execution failed", http.StatusInternalServerError)
		return
	}

	// 5. レスポンス返却
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(response); err != nil && s.logger != nil {
		s.logger.Debug("Failed to write response", "error", err)
	}
}

// Handler returns the HTTP handler for testing purposes
func (s *Server) Handler() http.Handler {
	return s.server.Handler
}

// Start starts the HTTP server and blocks until the context is cancelled.
func (s *Server) Start(ctx context.Context) error {
	errChan := make(chan error, 1)

	go func() {
		s.logger.Info("Server starting", "addr", s.server.Addr)
		if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	select {
	case err := <-errChan:
		return err
	case <-ctx.Done():
		s.logger.Info("Shutting down server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
		defer cancel()
		return s.server.Shutdown(shutdownCtx)
	}
}

// parseHeaders はカスタムヘッダーマッピングに基づいて HTTP ヘッダーから環境変数と引数を抽出します
// envMapping: ヘッダー名 → 環境変数名 (例: "X-Slack-Token" → "SLACK_TOKEN")
// argMapping: ヘッダー名 → 引数名 (例: "X-Team-Id" → "team-id")
func parseHeaders(headers http.Header, envMapping, argMapping map[string]string) (map[string]string, []string) {
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
