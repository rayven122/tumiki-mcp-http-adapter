package proxy

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"tumiki-mcp-http/internal/headers"
	"tumiki-mcp-http/internal/process"
)

// Config - シンプル化された設定構造体
type Config struct {
	// Server settings
	Host            string
	Port            int
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	ShutdownTimeout time.Duration

	// Stdio command
	Command string
	Args    []string

	// Environment variables
	DefaultEnv       map[string]string
	HeaderEnvMapping map[string]string
	HeaderArgMapping map[string]string

	// Process settings
	ProcessTimeout time.Duration
}

type Server struct {
	cfg    *Config
	logger *slog.Logger
	server *http.Server
}

func NewServer(cfg *Config, logger *slog.Logger) (*Server, error) {
	s := &Server{
		cfg:    cfg,
		logger: logger,
	}

	mux := http.NewServeMux()

	// MCP エンドポイント（/mcp のみ、/health は削除）
	mux.HandleFunc("/mcp", s.handleMCP)

	s.server = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Handler:      mux,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
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
	headerEnv, headerArgs := headers.ParseCustomHeaders(
		r.Header,
		s.cfg.HeaderEnvMapping,
		s.cfg.HeaderArgMapping,
	)

	// ヘッダーから取得した環境変数（デフォルトを上書き）
	for k, v := range headerEnv {
		envVars[k] = v
	}

	// 2. 引数マージ
	args := append(s.cfg.Args, headerArgs...)

	// 3. リクエストボディ読み込み
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// 4. stdio プロセス実行
	ctx, cancel := context.WithTimeout(r.Context(), s.cfg.ProcessTimeout)
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
	w.Write(response)
}

// Handler returns the HTTP handler for testing purposes
func (s *Server) Handler() http.Handler {
	return s.server.Handler
}

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
		shutdownCtx, cancel := context.WithTimeout(context.Background(), s.cfg.ShutdownTimeout)
		defer cancel()
		return s.server.Shutdown(shutdownCtx)
	}
}
