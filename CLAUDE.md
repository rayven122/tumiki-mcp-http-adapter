# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## プロジェクト概要

stdio MCP サーバーを HTTP エンドポイントとして公開する Go 実装プロキシサーバー。
Streamable HTTP パターンを採用し、HTTP ヘッダーから動的に環境変数・引数を設定可能。

## アーキテクチャ構成

### パッケージ構造

```
cmd/tumiki-mcp-http/     # エントリーポイント
  ├─ CLI フラグ解析
  ├─ 設定ビルド (buildConfigFromFlags)
  └─ サーバー起動 (startServer)

internal/proxy/          # HTTPサーバー層
  ├─ HTTP リクエスト処理 (handleMCP)
  ├─ ヘッダー解析 (parseHeaders)
  └─ 動的マッピング管理

internal/process/        # プロセス実行層
  ├─ stdio プロセス起動 (Execute)
  ├─ stdin/stdout/stderr パイプ管理
  └─ タイムアウト・Context 制御
```

### データフロー

1. **HTTP Request** → `proxy.handleMCP()`
2. **ヘッダー解析** → `parseHeaders()` で環境変数・引数を抽出
3. **設定マージ** → デフォルト環境変数 + ヘッダー由来の値
4. **プロセス実行** → `process.Execute()` で stdio MCP サーバー起動
5. **HTTP Response** → stdout から読み取った JSON-RPC レスポンスを返却

### 重要な設計パターン

#### 1. Streamable HTTP パターン

HTTP リクエストごとに異なる環境変数・引数を動的に設定できる設計。

**CLI 起動時（マッピング定義）**:
```bash
--header-env "X-Slack-Token=SLACK_TOKEN"  # ヘッダー → 環境変数
--header-arg "X-Team-Id=team-id"          # ヘッダー → 引数
```

**HTTP リクエスト時（実際の値）**:
```
X-Slack-Token: xoxp-12345  →  環境変数 SLACK_TOKEN=xoxp-12345
X-Team-Id: T123            →  引数 --team-id T123
```

実装: `internal/proxy/server.go:parseHeaders()`

#### 2. リクエストごとの独立プロセス

- 各 HTTP リクエストで独立した stdio プロセスを起動
- ステートレス設計（プロセス間で状態共有なし）
- Context 伝播でクライアント切断時の適切なクリーンアップ

実装: `internal/process/executor.go:Execute()`

#### 3. データレース対策

`stderr` の非同期読み取りで `sync.WaitGroup` を使用してデータレースを防止。

```go
var stderrWg sync.WaitGroup
stderrWg.Add(1)
go func() {
    defer stderrWg.Done()
    io.Copy(&stderrBuf, stderr)
}()
// ... プロセス処理 ...
stderrWg.Wait()  // stderr 読み取り完了を待つ
```

実装: `internal/process/executor.go:Execute()`

## 開発コマンド

### 必須ツール

- **Go 1.25+**
- **[Task](https://taskfile.dev/)** - タスクランナー

```bash
# 開発ツールの一括インストール
task install-tools
```

### よく使うコマンド

```bash
# ビルド
task build

# テスト実行（ローカル開発用、race detector 有効）
task test

# カバレッジ測定
task coverage

# コードフォーマット + リント + テスト（コミット前に実行）
task check

# クリーンアップ
task clean

# ビルドせずに実行（開発時）
go run ./cmd/tumiki-mcp-http --stdio "npx -y @modelcontextprotocol/server-filesystem /data"

# 特定パッケージのみテスト
go test -v ./internal/proxy
go test -v ./internal/process

# レースディテクター付きテスト
go test -race ./...
```

### CI 用コマンド

```bash
# CI では race detector を無効化（パフォーマンス理由）
task check-ci
```

## テストポリシー

### カバレッジ要件

**原則: 単体テストは対象ファイルのカバレッジ100%を目指す**

#### テスト対象
- ✅ ビジネスロジックを含む全ての関数
- ✅ パブリック関数・メソッド
- ✅ 内部ヘルパー関数（ロジックを含む場合）
- ✅ データパース・変換関数
- ✅ バリデーション関数

#### テスト除外対象
- ❌ `main()` 関数（エントリーポイント）
- ❌ サーバー起動関数（`startServer`など）
- ❌ ロガー初期化などのセットアップ関数
- ❌ `os.Exit()`や`log.Fatal()`を呼ぶコードパス

これらは統合テストやE2Eテストでカバーする。

### テスト名の命名規則

**原則: テスト名は「入力条件_期待される結果」の形式で記述する**

```go
// ❌ 悪い例: 結果が不明確
{name: "値にイコールを含む場合", ...}

// ✅ 良い例: 入力条件_期待される結果
{name: "値にイコールを含む環境変数_エラーを返す", ...}
{name: "空の入力_空のマップを返す", ...}
```

**命名パターン例:**

```go
// 正常系
"正常な環境変数1つ_マップに変換される"
"複数の環境変数_全てマップに変換される"

// 異常系
"値に=を含む環境変数_エラーを返す"
"無効なフォーマット_無視される"

// エッジケース
"特殊文字を含む値_正しくエスケープされる"
"Unicode文字列_正しくパースされる"
```

### テストケースの網羅性

各関数のテストは以下を含むこと:

```go
func TestXxx(t *testing.T) {
    tests := []struct {
        name     string
        input    InputType
        expected OutputType
        wantError bool
    }{
        // ✅ 正常系: 典型的なケース
        {name: "正常な入力_期待する出力を返す", ...},

        // ✅ 正常系: 境界値
        {name: "空の入力_空の結果を返す", ...},
        {name: "最大値_正しく処理される", ...},

        // ✅ 異常系: エラーケース
        {name: "無効な入力_エラーを返す", ..., wantError: true},

        // ✅ エッジケース: 特殊な条件
        {name: "特殊文字を含む入力_エスケープされて処理される", ...},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := FunctionUnderTest(tt.input)

            if tt.wantError {
                if err == nil {
                    t.Errorf("expected error but got none")
                }
                return
            }

            if err != nil {
                t.Errorf("unexpected error: %v", err)
            }

            if !reflect.DeepEqual(result, tt.expected) {
                t.Errorf("got %v, want %v", result, tt.expected)
            }
        })
    }
}
```

### エラーハンドリングのテスト

**必須**: エラーを返す関数は、エラーケースを必ずテストすること

```go
// ✅ 良い例: エラーケースもテスト
func TestParseEnvVars(t *testing.T) {
    tests := []struct {
        name      string
        input     ArrayFlags
        wantError bool
    }{
        {name: "正常_エラーなし", input: ArrayFlags{"KEY=value"}, wantError: false},
        {name: "値に=を含む_エラーを返す", input: ArrayFlags{"KEY=value=invalid"}, wantError: true},
    }
    // ...
}
```

### カバレッジ測定

```bash
# カバレッジ測定
go test -coverprofile=coverage.out ./...

# カバレッジ詳細表示
go tool cover -func=coverage.out

# HTMLレポート生成
go tool cover -html=coverage.out -o coverage.html
```

## コーディング規約

### エラーハンドリング

```go
// ✅ 良い例: エラーを返す（テスト可能）
func parseEnvVars(envVars ArrayFlags) (map[string]string, error) {
    if invalidCondition {
        return nil, fmt.Errorf("エラーメッセージ")
    }
    return result, nil
}

// ❌ 悪い例: log.Fatalで即終了（テスト不可能）
func parseEnvVars(envVars ArrayFlags) map[string]string {
    if invalidCondition {
        log.Fatal("エラーメッセージ")  // テストできない
    }
    return result
}
```

### バリデーション

- 入力値のバリデーションは関数の最初で行う
- エラーメッセージは明確で、問題の特定を容易にする
- バリデーションエラーは必ずテストする

```go
func parseMapping(mappings ArrayFlags) (map[string]string, error) {
    result := make(map[string]string)
    for _, mapping := range mappings {
        parts := strings.SplitN(mapping, "=", 2)
        if len(parts) == 2 {
            // バリデーション
            if strings.Contains(parts[1], "=") {
                return nil, fmt.Errorf(
                    "mapping value cannot contain '=' character: %s\nValue: %s",
                    mapping, parts[1],
                )
            }
            result[parts[0]] = parts[1]
        }
    }
    return result, nil
}
```

## 重要な実装ポイント

### 1. スライスの不変性

引数マージ時に元のスライスを変更しない（並行処理の安全性）。

```go
// ✅ 良い例: 新しいスライスを作成
args := make([]string, 0, len(s.cfg.Args)+len(headerArgs))
args = append(args, s.cfg.Args...)
args = append(args, headerArgs...)

// ❌ 悪い例: 元のスライスを変更
s.cfg.Args = append(s.cfg.Args, headerArgs...)
```

実装箇所: `internal/proxy/server.go:handleMCP()`

### 2. Context 伝播

- HTTP リクエストの Context をプロセス実行に伝播
- クライアント切断時にプロセスも終了（`exec.CommandContext`）
- タイムアウト制御（30秒のデフォルトタイムアウト）

```go
ctx, cancel := context.WithTimeout(r.Context(), ProcessTimeout)
defer cancel()

cmd := exec.CommandContext(ctx, command, args...)
```

実装箇所: `internal/proxy/server.go:handleMCP()`, `internal/process/executor.go:Execute()`

### 3. Graceful Shutdown

defer + exitCode パターンでシグナルハンドリング時も適切にクリーンアップ。

```go
ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
var exitCode int
defer func() {
    stop()
    if exitCode != 0 {
        os.Exit(exitCode)
    }
}()
```

実装箇所: `cmd/tumiki-mcp-http/main.go:startServer()`

## 関連ドキュメント

- **[docs/DESIGN.md](docs/DESIGN.md)** - 詳細なシステムアーキテクチャ設計書
- **[docs/DEVELOPMENT.md](docs/DEVELOPMENT.md)** - 開発環境セットアップ、リリース手順
- **[README.md](README.md)** - プロジェクト概要と使用方法
- **[Taskfile.yml](Taskfile.yml)** - タスク定義の詳細
- **[.golangci.yml](.golangci.yml)** - リンター設定の詳細
