package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"tumiki-mcp-http/internal/proxy"
)

// ArrayFlags - 複数回指定可能なフラグ
type ArrayFlags []string

func (a *ArrayFlags) String() string {
	return strings.Join(*a, ",")
}

func (a *ArrayFlags) Set(value string) error {
	*a = append(*a, value)
	return nil
}

func main() {
	// フラグ定義
	var (
		// サーバー設定
		stdioCmd          = flag.String("stdio", "", "stdio command (e.g., 'npx -y server-filesystem /data')")
		envVars           ArrayFlags
		headerEnvMappings ArrayFlags
		headerArgMappings ArrayFlags

		// ネットワーク設定
		port = flag.Int("port", 8080, "listen port")
		host = flag.String("host", "0.0.0.0", "bind host")

		// デバッグ
		verbose  = flag.Bool("verbose", false, "verbose logging")
		logLevel = flag.String("log-level", "info", "log level (debug/info/warn/error)")
	)

	flag.Var(&envVars, "env", "environment variables KEY=VALUE (repeatable)")
	flag.Var(&headerEnvMappings, "header-env", "header to env mapping HEADER-NAME=ENV_VAR (repeatable)")
	flag.Var(&headerArgMappings, "header-arg", "header to arg mapping HEADER-NAME=arg-name (repeatable)")
	flag.Parse()

	// --stdio が必須
	if *stdioCmd == "" {
		fmt.Println("Error: --stdio flag is required")
		fmt.Println("\nUsage examples:")
		fmt.Println("  # Quick start")
		fmt.Println("  tumiki-mcp-http --stdio \"npx -y @modelcontextprotocol/server-filesystem /data\"")
		fmt.Println("\n  # With environment variables")
		fmt.Println("  tumiki-mcp-http --stdio \"npx -y server-github\" --env \"GITHUB_TOKEN=ghp_xxx\"")
		fmt.Println("\n  # With header mappings (define header → env/arg mapping)")
		fmt.Println("  tumiki-mcp-http --stdio \"npx -y server-slack\" \\")
		fmt.Println("    --header-env \"X-Slack-Token=SLACK_TOKEN\" \\")
		fmt.Println("    --header-arg \"X-Team-Id=team-id\"")
		os.Exit(1)
	}

	// 設定を構築
	cfg := buildConfigFromFlags(
		*stdioCmd, envVars, headerEnvMappings, headerArgMappings,
		*host, *port,
	)

	// サーバー起動
	startServer(cfg, *verbose, *logLevel)
}

func buildConfigFromFlags(
	stdioCmd string,
	envVars, headerEnvMappings, headerArgMappings ArrayFlags,
	host string, port int,
) *proxy.Config {
	// stdioコマンドのパース
	cmdParts := parseStdioCommand(stdioCmd)
	if len(cmdParts) == 0 {
		log.Fatal("Error: No command specified")
	}

	// 環境変数のパース（--envフラグ）
	envMap := parseEnvVars(envVars)

	// ヘッダーマッピングのパース
	headerEnvMap := parseMapping(headerEnvMappings)
	headerArgMap := parseMapping(headerArgMappings)

	cfg := &proxy.Config{
		// Server settings
		Host:            host,
		Port:            port,
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		ShutdownTimeout: 10 * time.Second,

		// Stdio command
		Command: cmdParts[0],
		Args:    cmdParts[1:],

		// Environment variables
		DefaultEnv:       envMap,
		HeaderEnvMapping: headerEnvMap,
		HeaderArgMapping: headerArgMap,

		// Process settings
		ProcessTimeout: 30 * time.Second,
	}

	return cfg
}

func parseStdioCommand(stdioCmd string) []string {
	// シェルスタイルのコマンド文字列を解析
	var parts []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for i, r := range stdioCmd {
		switch r {
		case '"', '\'':
			if !inQuote {
				inQuote = true
				quoteChar = r
			} else if r == quoteChar {
				inQuote = false
				quoteChar = 0
			} else {
				current.WriteRune(r)
			}
		case ' ':
			if inQuote {
				current.WriteRune(r)
			} else if current.Len() > 0 {
				parts = append(parts, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(r)
		}

		// 最後の文字
		if i == len(stdioCmd)-1 && current.Len() > 0 {
			parts = append(parts, current.String())
		}
	}

	return parts
}

func parseEnvVars(envVars ArrayFlags) map[string]string {
	envMap := make(map[string]string)
	for _, env := range envVars {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}
	return envMap
}

// parseMapping は "KEY=VALUE" 形式の配列をマップに変換します
func parseMapping(mappings ArrayFlags) map[string]string {
	result := make(map[string]string)
	for _, mapping := range mappings {
		parts := strings.SplitN(mapping, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

func startServer(cfg *proxy.Config, verbose bool, logLevel string) {
	logger := initLogger(verbose, logLevel)

	proxyServer, err := proxy.NewServer(cfg, logger)
	if err != nil {
		logger.Error("Server initialization failed", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := proxyServer.Start(ctx); err != nil {
		logger.Error("Server error", "error", err)
		os.Exit(1)
	}

	logger.Info("Server stopped")
}

func initLogger(verbose bool, logLevel string) *slog.Logger {
	var level slog.Level
	switch strings.ToLower(logLevel) {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	default:
		level = slog.LevelInfo
	}

	if verbose {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	return slog.New(handler)
}
