# Tumiki MCP HTTP Adapter - 実装計画書

## プロジェクト概要

### 目的
stdio ベースの MCP（Model Context Protocol）サーバーを HTTP エンドポイントとして公開する軽量プロキシサーバー。

### コアコンセプト
- **軽量性**: 認証なし、シンプルな stdio プロキシ
- **即座に起動**: `--stdio` フラグだけで起動可能
- **Streamable HTTP 対応**: ヘッダーで動的に環境変数・引数を設定可能
- **カスタムマッピング**: 完全に自由なヘッダー名で環境変数・引数を指定可能

---

## アーキテクチャ設計

### システム構成

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ HTTP Request
       │ (Custom Headers: X-Slack-Token, X-Team-Id, etc.)
       ▼
┌─────────────────────────────────────┐
│   Tumiki MCP HTTP Adapter           │
│                                     │
│  ┌──────────────────────────────┐  │
│  │  Proxy Handler               │  │
│  │  - Header Mapping            │  │
│  │  - Env/Args Building         │  │
│  └───────────┬──────────────────┘  │
│              ▼                      │
│  ┌──────────────────────────────┐  │
│  │  Process Executor            │  │
│  │  - Stdio Process Launch      │  │
│  │  - Input/Output Handling     │  │
│  └──────────────────────────────┘  │
└─────────────┬───────────────────────┘
              │
              ▼
      ┌──────────────┐
      │ MCP Server   │
      │ (stdio mode) │
      └──────────────┘
```

### データフロー

1. **リクエスト受信**: HTTP POST /mcp
2. **ヘッダー解析**: カスタムマッピングに基づいて環境変数・引数を抽出
3. **設定マージ**: デフォルト環境変数 + ヘッダー由来の値
4. **プロセス起動**: stdio モードで MCP サーバーを実行
5. **レスポンス返却**: MCP サーバーの出力を HTTP レスポンスとして返却

---

## 技術スタック

### 言語・バージョン
- **Go**: 1.25+ (推奨: 1.25.3)
- **標準ライブラリ**: net/http, context, exec, slog

### 外部依存
- **gopkg.in/yaml.v3**: YAML 設定ファイル解析

### 開発ツール
- **Testing**: Go 標準テストフレームワーク
- **Build**: Go modules

---

## ディレクトリ構造

```
tumiki-mcp-http-adapter/
├── cmd/
│   └── tumiki-mcp-http/
│       ├── main.go              # エントリーポイント、CLI フラグ解析
│       └── main_test.go         # CLI テスト
│
├── internal/                    # プライベートパッケージ
│   ├── headers/
│   │   ├── parser.go           # ヘッダー解析（カスタムマッピング）
│   │   └── parser_test.go      # ヘッダー解析テスト
│   │
│   ├── proxy/
│   │   ├── server.go           # HTTP サーバー、MCP ハンドラー、Config 定義
│   │   └── server_test.go      # プロキシテスト
│   │
│   └── process/
│       ├── executor.go         # stdio プロセス実行
│       └── executor_test.go    # プロセス実行テスト
│
├── test/
│   └── integration_test.go     # 統合テスト
│
├── docs/
│   └── IMPLEMENTATION_PLAN.md  # 本ドキュメント
│
├── README.md                    # ユーザー向けドキュメント
├── go.mod                       # Go モジュール定義
└── go.sum                       # 依存関係チェックサム
```

---

## コンポーネント詳細

### 1. cmd/tumiki-mcp-http/main.go

**責務**: アプリケーションのエントリーポイント、CLI フラグ解析

**主要機能**:
- コマンドラインフラグの定義と解析
- 設定ファイルまたは CLI フラグから設定をビルド
- サーバーの起動とシャットダウン処理

**実装されているフラグ**:
```go
// 基本オプション
--stdio <command>              // stdio コマンド全体（必須）
--env <KEY=VALUE>              // 環境変数（複数可）
--port <number>                // ポート番号（デフォルト: 8080）
--host <address>               // ホスト（デフォルト: 0.0.0.0）

// ヘッダーマッピング
--header-env <HEADER=ENV_VAR>  // ヘッダー→環境変数マッピング（複数可）
--header-arg <HEADER=arg-name> // ヘッダー→引数マッピング（複数可）

// デバッグ
--verbose                      // 詳細ログ
--log-level <level>            // ログレベル (debug/info/warn/error)
```

### 2. internal/proxy/

**責務**: HTTP サーバー、MCP エンドポイントハンドラー、設定定義

**主要型定義**:
```go
// Config - シンプル化された設定構造体（proxy パッケージ内に定義）
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
```

**主要メソッド**:
```go
func NewServer(cfg *Config, logger *slog.Logger) (*Server, error)
func (s *Server) Start(ctx context.Context) error
func (s *Server) Handler() http.Handler  // テスト用
func (s *Server) handleMCP(w http.ResponseWriter, r *http.Request)
```

**handleMCP の処理フロー**（シンプル化後）:
1. カスタムヘッダーマッピングで環境変数・引数を抽出
2. デフォルト環境変数とマージ
3. リクエストボディ読み込み
4. stdio プロセス実行
5. レスポンス返却

### 3. internal/headers/

**責務**: HTTP ヘッダーの解析とカスタムマッピング処理

**主要関数**:
```go
// カスタムマッピングに基づいてヘッダーを解析
func ParseCustomHeaders(
    headers http.Header,
    envMapping map[string]string,
    argMapping map[string]string,
) (envVars map[string]string, args []string)
```

**動作例**:
```go
// マッピング定義
envMapping := map[string]string{
    "X-Slack-Token": "SLACK_TOKEN",
}
argMapping := map[string]string{
    "X-Team-Id": "team-id",
    "X-Channel": "channel",
}

// HTTP ヘッダー
headers := http.Header{
    "X-Slack-Token": []string{"xoxp-xxxxx"},
    "X-Team-Id":     []string{"T123"},
    "X-Channel":     []string{"general"},
}

// 解析結果
envVars, args := ParseCustomHeaders(headers, envMapping, argMapping)
// envVars = {"SLACK_TOKEN": "xoxp-xxxxx"}
// args = ["--team-id", "T123", "--channel", "general"]
```

### 4. internal/process/

**責務**: stdio プロセスの起動と入出力処理

**主要型定義**:
```go
type Executor struct {
    command string
    args    []string
    env     map[string]string
    logger  *slog.Logger
}

func NewExecutor(
    command string,
    args []string,
    env map[string]string,
    logger *slog.Logger,
) *Executor

func (e *Executor) Execute(ctx context.Context, input []byte) ([]byte, error)
```

**Execute の処理**:
1. `exec.CommandContext` でプロセス作成
2. 環境変数設定
3. stdin/stdout/stderr パイプ接続
4. プロセス起動
5. 入力データを stdin に書き込み
6. stdout から出力を読み取り
7. プロセス終了を待機
8. 出力データを返却

---

## カスタムヘッダーマッピングの仕組み

### Streamable HTTP パターン

従来の MCP HTTP アダプターでは、環境変数や引数を CLI 起動時に固定で指定していましたが、本実装では **Streamable HTTP パターン** を採用し、HTTP リクエストごとに動的に値を変更できます。

### 動作原理

**ステップ 1: CLI 起動時（マッピング定義）**
```bash
tumiki-mcp-http \
  --stdio "npx -y server-slack" \
  --header-env "X-Slack-Token=SLACK_TOKEN" \
  --header-arg "X-Team-Id=team-id" \
  --header-arg "X-Channel=channel"
```

この時点では：
- ヘッダー名と環境変数/引数名の**対応関係のみ**を定義
- 実際の値は設定しない

**ステップ 2: HTTP リクエスト時（値を送信）**
```bash
curl -X POST http://localhost:8080/mcp \
  -H "X-Slack-Token: xoxp-12345" \
  -H "X-Team-Id: T123" \
  -H "X-Channel: general" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

**ステップ 3: サーバー側処理**
1. ヘッダーから値を抽出
2. マッピング定義に従って変換
3. 実行コマンド生成

**結果**:
```bash
# 実行されるコマンド
npx -y server-slack --team-id T123 --channel general

# 設定される環境変数
SLACK_TOKEN=xoxp-12345
```

### メリット

1. **動的設定**: リクエストごとに異なる値を使用可能
2. **マルチテナント**: チームやユーザーごとに異なる認証情報
3. **セキュリティ**: トークンをコマンドライン引数に含めない
4. **柔軟性**: 完全に自由なヘッダー名をサポート

---


---

## テスト戦略

### 単体テスト

**カバレッジ対象**:
- `internal/headers` - カスタムヘッダーマッピング解析
- `internal/process` - プロセス実行（モック使用）
- `internal/proxy` - HTTP ハンドラーロジック、Config 定義

**テスト実行**:
```bash
go test ./internal/...
```

### 統合テスト

**テスト対象**:
- エンドツーエンドの HTTP リクエスト処理
- カスタムヘッダーマッピングの動作確認
- プロセス実行とレスポンス返却の検証

**テストケース例**:
```go
// test/integration_test.go
func TestServerIntegration(t *testing.T) {
    // 基本的な MCP リクエスト処理
    // カスタムヘッダー付きリクエスト
    // 環境変数と引数のマッピング検証
}

func TestHeaderMappingIntegration(t *testing.T) {
    // ヘッダー→環境変数マッピングの検証
    // ヘッダー→引数マッピングの検証
}
```

**テスト実行**:
```bash
go test ./test
```

### テスト方針

1. **モック使用**: 外部プロセス呼び出しはモック化
2. **httptest 活用**: HTTP サーバーのテストは `httptest.NewServer` を使用
3. **テーブル駆動**: 複数のテストケースを構造化して実行
4. **エラーケース**: 正常系だけでなく異常系もカバー

---

## セキュリティ考慮事項

### 環境変数の扱い

- **機密情報の保護**: トークンや API キーは環境変数で渡す
- **ログ出力の制御**: 機密情報はログに出力しない
- **プロセス分離**: 各リクエストで独立したプロセスを起動

### HTTP セキュリティ

- **認証なし**: 軽量な stdio プロキシのため認証機能は含まれていません。必要に応じてリバースプロキシ（nginx、Caddy等）で認証を実装してください
- **タイムアウト設定**: Read/Write タイムアウトでリソース枯渇を防止
- **Graceful Shutdown**: SIGTERM/SIGINT で安全に終了
- **シンプルな設計**: 認証・複雑な設定管理を含まず、必要に応じて外部で実装

---

## パフォーマンス考慮事項

### プロセス管理

- **タイムアウト制御**: デフォルト 30 秒でプロセスをタイムアウト
- **Context 利用**: `context.Context` で適切なキャンセル処理
- **バッファサイズ**: デフォルト 8192 バイトの I/O バッファ

### HTTP サーバー

- **並行処理**: Go の goroutine で自動的に並行リクエスト処理
- **タイムアウト**: Read/Write タイムアウトでリソース保護
- **Keep-Alive**: HTTP 1.1 の Keep-Alive をサポート

### リソース管理

- **プロセスクリーンアップ**: リクエスト完了後は確実にプロセス終了
- **メモリ効率**: ストリーミング処理でメモリ使用量を抑制

---

## エラーハンドリング

### エラーレスポンス

**HTTP ステータスコード**:
- `200 OK`: 正常処理
- `400 Bad Request`: 不正なリクエスト（リクエストボディの読み込み失敗等）
- `500 Internal Server Error`: プロセス実行失敗

**エラーメッセージ**:
```go
http.Error(w, "Failed to read body", http.StatusBadRequest)
http.Error(w, "Process execution failed", http.StatusInternalServerError)
```

### ログ出力

**構造化ログ（slog）**:
```go
logger.Info("Server starting", "addr", s.server.Addr)
logger.Error("Process execution failed", "error", err)
```

**ログレベル**:
- `debug`: 詳細なデバッグ情報
- `info`: 通常の動作ログ
- `warn`: 警告（回復可能なエラー）
- `error`: エラー（処理失敗）

---

## 拡張可能性

### 外部統合

- **リバースプロキシ**: nginx、Caddy 等で認証・TLS・ロードバランシングを実装
- **監視**: ログ出力を外部監視システムに転送

### シンプルな設計のメリット

- **理解しやすい**: 小さなコードベースで保守が容易
- **拡張しやすい**: 必要な機能は外部で追加可能
- **デバッグしやすい**: シンプルなフローで問題の特定が簡単

---

## デプロイメント

### ビルドと実行

```bash
# ビルド
go build -o tumiki-mcp-http ./cmd/tumiki-mcp-http

# クロスコンパイル（Linux）
GOOS=linux GOARCH=amd64 go build -o tumiki-mcp-http ./cmd/tumiki-mcp-http

# 実行（シンプル）
./tumiki-mcp-http --stdio "npx -y server-filesystem /data"

# ヘッダーマッピング付き実行
./tumiki-mcp-http \
  --stdio "npx -y server-slack" \
  --header-env "X-Slack-Token=SLACK_TOKEN" \
  --header-arg "X-Team-Id=team-id"
```

---

## まとめ

本プロジェクトは、stdio ベースの MCP サーバーを HTTP エンドポイントとして公開する**軽量なプロキシサーバー**です。

**主要な実装特徴**:

1. ✅ **極めて軽量**: 認証なし、設定ファイルなし、シンプルな stdio プロキシ
2. ✅ **Streamable HTTP パターン**: リクエストごとに動的に環境変数・引数を設定
3. ✅ **カスタムヘッダーマッピング**: 完全に自由なヘッダー名をサポート
4. ✅ **即座に起動**: `--stdio` フラグだけで起動可能（他は全てオプション）
5. ✅ **シンプルな構成**: 単一サーバー専用、単一エンドポイント（/mcp のみ）

**技術的な強み**:

- Go 1.25+ 標準ライブラリ中心の実装（最小限の依存）
- 小さなコードベース（3パッケージのみ）
- 構造化ログ（slog）による運用性
- Context ベースの適切なリソース管理

**シンプル化のポイント**:

- **設定**: CLIフラグのみ（YAML設定ファイル不要）
- **エンドポイント**: /mcp のみ（/health 削除）
- **サーバー管理**: 単一サーバー専用（複数サーバー切り替え不要）
- **認証**: 認証機能なし（必要なら外部で実装）
- **公開API**: 内部利用のみ（pkg/ 削除）

**設計思想**:

- HTTPリクエストを受け取り、ヘッダーを環境変数・引数に変換してstdioプロセスを実行するだけの極めてシンプルな設計
- 認証・複雑な設定管理・複数サーバー対応は含まず、必要に応じて外部（リバースプロキシ等）で実装
- **コード量30-40%削減**: 理解しやすく、保守しやすい実装
