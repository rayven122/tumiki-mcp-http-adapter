// Package main は tumiki-mcp-http プロキシサーバーのメインエントリーポイントを提供します。
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

	"github.com/rayven122/tumiki-mcp-http-adapter/internal/proxy"
)

// ArrayFlags は複数回指定可能なフラグ型です。
type ArrayFlags []string

func (a *ArrayFlags) String() string {
	return strings.Join(*a, ",")
}

// Set は ArrayFlags に値を追加します。
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
		port = flag.Int("port", 8080, "listen port (default: 8080)")

		// ログレベル
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
		fmt.Println("\n  # Custom host binding (use HOST environment variable)")
		fmt.Println("  HOST=127.0.0.1 tumiki-mcp-http --stdio \"npx -y server-filesystem /data\"")
		os.Exit(1)
	}

	// 設定を構築
	cfg := buildConfigFromFlags(
		*stdioCmd, envVars, headerEnvMappings, headerArgMappings, *port,
	)

	// サーバー起動
	startServer(cfg, *logLevel)
}

func buildConfigFromFlags(
	stdioCmd string,
	envVars, headerEnvMappings, headerArgMappings ArrayFlags,
	port int,
) *proxy.Config {
	// stdioコマンドのパース
	cmdParts := parseStdioCommand(stdioCmd)
	if len(cmdParts) == 0 {
		log.Fatal("Error: No command specified")
	}

	// 環境変数のパース（--envフラグ）
	envMap, err := parseKeyValuePairs(envVars, "environment variable")
	if err != nil {
		log.Fatal(err)
	}

	// ヘッダーマッピングのパース
	headerEnvMap, err := parseKeyValuePairs(headerEnvMappings, "header-env mapping")
	if err != nil {
		log.Fatal(err)
	}
	headerArgMap, err := parseKeyValuePairs(headerArgMappings, "header-arg mapping")
	if err != nil {
		log.Fatal(err)
	}

	cfg := &proxy.Config{
		Port:             port,
		Command:          cmdParts[0],
		Args:             cmdParts[1:],
		DefaultEnv:       envMap,
		HeaderEnvMapping: headerEnvMap,
		HeaderArgMapping: headerArgMap,
	}

	return cfg
}

func parseStdioCommand(stdioCmd string) []string {
	// シェルスタイルのコマンド文字列を解析
	parts := []string{}
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for i, r := range stdioCmd {
		switch {
		case r == '"' || r == '\'':
			switch {
			case !inQuote:
				inQuote = true
				quoteChar = r
			case r == quoteChar:
				inQuote = false
				quoteChar = 0
			default:
				current.WriteRune(r)
			}
		case r == ' ':
			switch {
			case inQuote:
				current.WriteRune(r)
			case current.Len() > 0:
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

// parseKeyValuePairs は "KEY=VALUE" 形式の配列をマップに変換します。
// valueType パラメータはエラーメッセージに使用されます（例: "environment variable", "mapping"）。
func parseKeyValuePairs(pairs ArrayFlags, valueType string) (map[string]string, error) {
	result := make(map[string]string)
	for _, pair := range pairs {
		parts := strings.SplitN(pair, "=", 2)
		if len(parts) == 2 {
			// 値に '=' が含まれているかチェック
			if strings.Contains(parts[1], "=") {
				return nil, fmt.Errorf("%s value cannot contain '=' character: %s\nValue: %s", valueType, pair, parts[1])
			}
			result[parts[0]] = parts[1]
		}
	}
	return result, nil
}

func startServer(cfg *proxy.Config, logLevel string) {
	logger := initLogger(logLevel)

	proxyServer, err := proxy.NewServer(cfg, logger)
	if err != nil {
		logger.Error("Server initialization failed", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	// deferが実行されるように、os.Exit前にstopを呼ぶ
	var exitCode int
	defer func() {
		stop()
		if exitCode != 0 {
			os.Exit(exitCode)
		}
	}()

	if err := proxyServer.Start(ctx); err != nil {
		logger.Error("Server error", "error", err)
		exitCode = 1
		return
	}

	logger.Info("Server stopped")
}

func initLogger(logLevel string) *slog.Logger {
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

	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewJSONHandler(os.Stdout, opts)
	return slog.New(handler)
}
