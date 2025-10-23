package process

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"log/slog"
	"os/exec"
)

type Executor struct {
	command string
	args    []string
	env     map[string]string
	logger  *slog.Logger
}

func NewExecutor(command string, args []string, env map[string]string, logger *slog.Logger) *Executor {
	return &Executor{
		command: command,
		args:    args,
		env:     env,
		logger:  logger,
	}
}

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
	go func() {
		io.Copy(&stderrBuf, stderr)
	}()

	// 6. stdin に JSON-RPC メッセージ送信
	if _, err := stdin.Write(input); err != nil {
		return nil, fmt.Errorf("write to stdin: %w", err)
	}
	if _, err := stdin.Write([]byte("\n")); err != nil {
		return nil, fmt.Errorf("write newline to stdin: %w", err)
	}
	stdin.Close()

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
	if err := cmd.Wait(); err != nil {
		if e.logger != nil {
			e.logger.Error("Process failed", "stderr", stderrBuf.String())
		}
		return nil, fmt.Errorf("process wait: %w", err)
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
