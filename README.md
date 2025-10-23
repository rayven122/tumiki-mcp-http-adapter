# Tumiki MCP HTTP Adapter

stdio MCP サーバーを HTTP エンドポイントとして公開する Go 実装プロキシサーバー

[![Go Version](https://img.shields.io/badge/Go-1.25%2B-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

---

## 概要

既存の stdio ベースの MCP（Model Context Protocol）サーバーを、HTTP エンドポイントとして公開するプロキシサーバー。シンプルな CLI 体験と柔軟な設定管理を両立。

### 主な特徴

- ✅ **軽量**: 認証なし、シンプルな stdio プロキシ
- ✅ **即座に起動**: `--stdio`フラグだけで起動可能
- ✅ **動的設定**: HTTP ヘッダーから環境変数・引数を設定可能（streamable http 対応）
- ✅ **カスタムヘッダーマッピング**: 完全に自由なヘッダー名で環境変数・引数を指定可能

---

## クイックスタート

### シンプルな起動

```bash
# filesystemサーバーを起動
tumiki-mcp-http \
  --stdio "npx -y @modelcontextprotocol/server-filesystem /data"

# 環境変数付きでGitHubサーバーを起動
tumiki-mcp-http \
  --stdio "npx -y @modelcontextprotocol/server-github" \
  --env "GITHUB_TOKEN=ghp_xxxxx"

# ヘッダーマッピングを定義（値はHTTPリクエスト時に渡す）
tumiki-mcp-http \
  --stdio "npx -y server-slack" \
  --header-env "X-Slack-Token=SLACK_TOKEN" \
  --header-arg "X-Team-Id=team-id"
```

---

## 使用例

### Filesystem サーバー

```bash
# 起動
tumiki-mcp-http \
  --stdio "npx -y @modelcontextprotocol/server-filesystem /Users/yourname/Documents" \
  --port 8001

# テスト
curl -X POST http://localhost:8001/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

### GitHub サーバー（環境変数付き）

```bash
# 起動
tumiki-mcp-http \
  --stdio "npx -y @modelcontextprotocol/server-github" \
  --env "GITHUB_TOKEN=ghp_xxxxx" \
  --port 8001

# テスト
curl -X POST http://localhost:8001/mcp \
  -H "Content-Type: application/json" \
  -d '{
    "jsonrpc": "2.0",
    "id": 1,
    "method": "tools/call",
    "params": {
      "name": "search_repositories",
      "arguments": {"query": "language:go"}
    }
  }'
```

### Slack サーバー

```bash
tumiki-mcp-http \
  --stdio "npx -y slack-mcp-server" \
  --env "SLACK_MCP_XOXP_TOKEN=xoxp-xxxxx" \
  --env "SLACK_MCP_TRANSPORT=stdio" \
  --port 8002
```

---

## インストール

### ビルドから

```bash
# クローン
git clone https://github.com/your-org/tumiki-mcp-http-adapter.git
cd tumiki-mcp-http-adapter

# ビルド
go build -o tumiki-mcp-http ./cmd/tumiki-mcp-http

# インストール
sudo cp tumiki-mcp-http /usr/local/bin/
```

### Docker で実行

```bash
# ビルド
docker build -t tumiki-mcp-http .

# 実行
docker run -p 8080:8080 \
  tumiki-mcp-http \
  --stdio "npx -y @modelcontextprotocol/server-filesystem /data"
```

---

## コマンドラインオプション

### 基本オプション

```bash
--stdio <command>          # stdioコマンド全体を指定（必須）
--env <KEY=VALUE>          # 環境変数（複数指定可）
--port <number>            # ポート（デフォルト: 8080）
--host <address>           # ホスト（デフォルト: 0.0.0.0）
```

### ヘッダーマッピングオプション

`--header-env`と`--header-arg`フラグで、HTTP ヘッダー名と環境変数/引数の**マッピング**を定義します。
完全に自由なヘッダー名が使用可能です。

#### CLI 起動時（マッピング定義）

```bash
# 環境変数マッピング: ヘッダー名=環境変数名
--header-env "X-Slack-Token=SLACK_TOKEN"
--header-env "Authorization=API_KEY"

# 引数マッピング: ヘッダー名=引数名
--header-arg "X-Team-Id=team-id"
--header-arg "X-Channel=channel"

# 完全な例
tumiki-mcp-http --stdio "npx -y server-slack" \
  --header-env "X-Slack-Token=SLACK_TOKEN" \
  --header-arg "X-Team-Id=team-id" \
  --header-arg "X-Channel=channel"
```

#### HTTP リクエスト時（実際の値）

```bash
# CLI起動後、HTTPリクエストで値を渡す
curl -X POST http://localhost:8080/mcp \
  -H "X-Slack-Token: xoxp-xxxxx" \
  -H "X-Team-Id: T123" \
  -H "X-Channel: general" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'

# 実行結果:
# コマンド: npx -y server-slack --team-id T123 --channel general
# 環境変数: SLACK_TOKEN=xoxp-xxxxx
```

#### 仕組み

1. **CLI 起動時**: ヘッダー名と変数名の**マッピングだけ**定義
2. **HTTP リクエスト時**: 定義されたヘッダーで**実際の値**を渡す
3. **実行**: マッピングに従って環境変数と引数に変換して実行

### デバッグオプション

```bash
--verbose                  # 詳細ログ出力
--log-level <level>        # ログレベル (debug/info/warn/error)
```

---

## プロジェクト構造

```text
tumiki-mcp-http-adapter/
├── cmd/tumiki-mcp-http/     # メインアプリケーション
│   └── main.go
├── internal/                 # プライベートパッケージ
│   ├── config/              # 設定管理
│   ├── headers/             # ヘッダー解析
│   ├── proxy/               # HTTPサーバー
│   └── process/             # プロセス実行
├── pkg/                      # 外部から利用可能なパッケージ
│   └── headers/             # ヘッダーマッピングユーティリティ
├── test/                     # 統合テスト
└── docs/                     # ドキュメント
    └── IMPLEMENTATION_PLAN.md
```

---

## テスト

### 単体テストの実行

```bash
# すべてのテストを実行
go test ./...

# カバレッジ付き
go test -cover ./...

# 詳細出力
go test -v ./...

# 特定パッケージのみ
go test ./internal/config
go test ./internal/headers
go test ./internal/process
go test ./internal/proxy
go test ./pkg/headers
go test ./test
```

### テストの構成

**内部パッケージテスト**:

- `internal/config/config_test.go` - 設定ファイル読み込みと環境変数展開
- `internal/headers/parser_test.go` - カスタムヘッダーマッピング解析
- `internal/process/executor_test.go` - stdio プロセス実行
- `internal/proxy/server_test.go` - HTTP サーバーと MCP ハンドラー

**公開パッケージテスト**:

- `pkg/headers/mapper_test.go` - 公開ヘッダーマッピング API

**統合テスト**:

- `test/integration_test.go` - エンドツーエンド統合テスト

---

## ドキュメント

- **[実装計画書](docs/IMPLEMENTATION_PLAN.md)** - 全体設計とアーキテクチャ、詳細な実装仕様

---

## 参考リンク

- [Model Context Protocol](https://modelcontextprotocol.io/)
- [MCP Specification](https://spec.modelcontextprotocol.io/)

---

## ライセンス

MIT License - 詳細は [LICENSE](LICENSE) を参照

---

## 貢献

プルリクエストと Issue を歓迎します！

1. このリポジトリをフォーク
2. 機能ブランチを作成 (`git checkout -b feature/amazing`)
3. 変更をコミット (`git commit -m 'Add amazing feature'`)
4. ブランチをプッシュ (`git push origin feature/amazing`)
5. プルリクエストを作成
