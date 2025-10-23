# Tumiki MCP HTTP Adapter

stdio MCP サーバーを HTTP エンドポイントとして公開する Go 実装プロキシサーバー

[![Go Version](https://img.shields.io/badge/Go-1.25%2B-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

> **📝 注意**: GitHubにpushする前に、README内の `YOUR_USERNAME` を実際のGitHubユーザー名に置き換え、`go.mod` のモジュール名を `github.com/YOUR_USERNAME/tumiki-mcp-http-adapter` に更新してください。

---

## 目次

- [概要](#概要)
- [インストール](#インストール)
- [使用例](#使用例)
- [コマンドラインオプション](#コマンドラインオプション)
- [開発](#開発)
- [ライセンス](#ライセンス)
- [貢献](#貢献)
- [サポート](#サポート)

---

## 概要

既存の stdio ベースの MCP（Model Context Protocol）サーバーを、HTTP エンドポイントとして公開するプロキシサーバー。

### 主な特徴

- ✅ **軽量**: シンプルな stdio プロキシ
- ✅ **即座に起動**: `--stdio`フラグだけで起動可能
- ✅ **動的設定**: HTTP ヘッダーから環境変数・引数を設定可能（streamable http 対応）
- ✅ **カスタムヘッダーマッピング**: 完全に自由なヘッダー名で環境変数・引数を指定可能

> **📖 技術的詳細**: システムアーキテクチャ、コンポーネント設計、セキュリティ設計などの詳細は [docs/DESIGN.md](docs/DESIGN.md) を参照してください。

### セキュリティに関する注意

このプロキシは**認証・認可機能を持たない**シンプルな実装です。本番環境で使用する場合は、以下を推奨します：

- リバースプロキシ（nginx, Caddy, Traefik 等）による認証・認可の実装
- TLS/HTTPS の有効化
- ファイアウォールによるアクセス制限
- レート制限の実装

詳細は [docs/DESIGN.md - セキュリティ設計](docs/DESIGN.md#セキュリティ設計) を参照してください。

---

## インストール

### ソースからビルド

```bash
# リポジトリをクローン
git clone https://github.com/YOUR_USERNAME/tumiki-mcp-http-adapter.git
cd tumiki-mcp-http-adapter

# ビルドとインストール
task build

# バイナリを PATH に配置（オプション）
sudo cp bin/tumiki-mcp-http /usr/local/bin/

# 確認
tumiki-mcp-http --help
```

### Go install（GitHubにpush後）

```bash
# 最新版をインストール
go install github.com/YOUR_USERNAME/tumiki-mcp-http-adapter/cmd/tumiki-mcp-http@latest

# 確認
tumiki-mcp-http --help
```

**注意**: `$GOPATH/bin` (通常 `~/go/bin`) が PATH に含まれていることを確認してください。

---

## 使用例

### Filesystem サーバー

```bash
# 起動
tumiki-mcp-http \
  --stdio "npx -y @modelcontextprotocol/server-filesystem /Users/yourname/Documents"

# テスト
curl -X POST http://localhost:8080/mcp \
  -H "Content-Type: application/json" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

### GitHub サーバー（環境変数付き）

```bash
# 起動
tumiki-mcp-http \
  --stdio "npx -y @modelcontextprotocol/server-github" \
  --env "GITHUB_TOKEN=ghp_xxxxx"

# テスト
curl -X POST http://localhost:8080/mcp \
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
  --env "SLACK_MCP_TRANSPORT=stdio"
```

---

## コマンドラインオプション

### オプション一覧

| オプション | 説明 | 必須 | 複数指定 |
|-----------|------|------|---------|
| `--stdio <command>` | stdio モードで実行する MCP サーバーのコマンド | ✅ | ❌ |
| `--env <KEY=VALUE>` | デフォルト環境変数の設定 | ❌ | ✅ |
| `--header-env <HEADER=ENV>` | HTTP ヘッダーから環境変数へのマッピング | ❌ | ✅ |
| `--header-arg <HEADER=ARG>` | HTTP ヘッダーからコマンド引数へのマッピング | ❌ | ✅ |
| `--log-level <level>` | ログレベル（debug/info/warn/error、デフォルト: info） | ❌ | ❌ |

### 環境変数での設定

サーバーの起動設定は環境変数でも指定可能です。

| 環境変数 | 説明 | デフォルト |
|---------|------|-----------|
| `HOST` | サーバーのホスト | `0.0.0.0` |
| `PORT` | サーバーのポート | `8080` |

```bash
# 例: ローカルホストのみでポート3000で起動
HOST=127.0.0.1 PORT=3000 tumiki-mcp-http --stdio "npx -y server-filesystem /data"
```

### ヘッダーマッピングの使用例

**ステップ1: CLI起動時にマッピングを定義**

```bash
tumiki-mcp-http --stdio "npx -y server-slack" \
  --header-env "X-Slack-Token=SLACK_TOKEN" \
  --header-arg "X-Team-Id=team-id" \
  --header-arg "X-Channel=channel"
```

**ステップ2: HTTPリクエスト時に実際の値を渡す**

```bash
curl -X POST http://localhost:8080/mcp \
  -H "X-Slack-Token: xoxp-xxxxx" \
  -H "X-Team-Id: T123" \
  -H "X-Channel: general" \
  -d '{"jsonrpc":"2.0","id":1,"method":"tools/list"}'
```

**実行結果**:
```bash
# 実行されるコマンド
npx -y server-slack --team-id T123 --channel general

# 設定される環境変数
SLACK_TOKEN=xoxp-xxxxx
```

---

## 開発

開発環境のセットアップ、テスト、コーディング規約などの詳細については、以下を参照してください：

**[開発ガイド (docs/DEVELOPMENT.md)](docs/DEVELOPMENT.md)**

---

## ライセンス

MIT License - 詳細は [LICENSE](LICENSE) を参照

---

## 貢献

プルリクエストと Issue を歓迎します！

### 貢献の流れ

1. このリポジトリをフォーク
2. 機能ブランチを作成 (`git checkout -b feature/amazing-feature`)
3. 変更をコミット (`git commit -m 'Add amazing feature'`)
4. コミット前チェックを実行 (`task check`)
5. ブランチをプッシュ (`git push origin feature/amazing-feature`)
6. プルリクエストを作成

### 貢献ガイドライン

- コーディング規約に従ってください（[docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) を参照）
- テストを追加してください（カバレッジ100%を目指す）
- コミット前に `task check` を実行してください
- わかりやすいコミットメッセージを書いてください

---

## サポート

- **バグ報告**: [Issues](https://github.com/YOUR_USERNAME/tumiki-mcp-http-adapter/issues) で報告してください
- **機能リクエスト**: [Issues](https://github.com/YOUR_USERNAME/tumiki-mcp-http-adapter/issues) で提案してください
- **質問**: [Discussions](https://github.com/YOUR_USERNAME/tumiki-mcp-http-adapter/discussions) でお気軽に質問してください
