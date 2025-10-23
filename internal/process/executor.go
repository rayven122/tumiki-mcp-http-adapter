// Package process は MCP サーバー向けの stdio プロセス実行機能を提供します。
package process

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
	"sync"
)

// Executor は stdio ベースの MCP サーバープロセスを実行します。
type Executor struct {
	command string
	args    []string
	env     map[string]string
	logger  *slog.Logger
}

// NewExecutor は指定されたコマンド、引数、環境変数、ロガーで新しい Executor を作成します。
func NewExecutor(command string, args []string, env map[string]string, logger *slog.Logger) *Executor {
	return &Executor{
		command: command,
		args:    args,
		env:     env,
		logger:  logger,
	}
}

// Execute は指定された入力で stdio プロセスを実行し、レスポンスを返します。
func (e *Executor) Execute(ctx context.Context, input []byte) ([]byte, error) {
	// 1. コマンド準備
	cmd := exec.CommandContext(ctx, e.command, e.args...)

	// 2. 環境変数設定
	cmd.Env = append(cmd.Environ(), e.envSlice()...)

	// 3. stdin/stdout パイプ
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("stderr pipe: %w", err)
	}

	// 4. プロセス起動
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("process start: %w", err)
	}

	// 5. stderr を非同期で読み取り
	var stderrBuf bytes.Buffer
	var stderrWg sync.WaitGroup
	stderrWg.Add(1)
	go func() {
		defer stderrWg.Done()
		if _, err := io.Copy(&stderrBuf, stderr); err != nil && e.logger != nil {
			e.logger.Debug("Failed to copy stderr", "error", err)
		}
	}()

	// 6. stdin に JSON-RPC メッセージ送信
	if _, err := stdin.Write(input); err != nil {
		return nil, fmt.Errorf("write to stdin: %w", err)
	}
	if _, err := stdin.Write([]byte("\n")); err != nil {
		return nil, fmt.Errorf("write newline to stdin: %w", err)
	}
	if err := stdin.Close(); err != nil && e.logger != nil {
		e.logger.Debug("Failed to close stdin", "error", err)
	}

	// 7. stdout から JSON-RPC レスポンス読み取り
	var response []byte
	scanner := bufio.NewScanner(stdout)
	if scanner.Scan() {
		response = scanner.Bytes()
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read from stdout: %w", err)
	}

	// 8. プロセス終了待機
	waitErr := cmd.Wait()

	// 9. stderrの読み取り完了を待つ
	stderrWg.Wait()

	if waitErr != nil {
		if e.logger != nil {
			e.logger.Error("Process failed", "stderr", stderrBuf.String())
		}
		return nil, fmt.Errorf("process wait: %w", waitErr)
	}

	return response, nil
}

func (e *Executor) envSlice() []string {
	env := make([]string, 0, len(e.env))
	for k, v := range e.env {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}
	return env
}
