# Tumiki MCP HTTP Adapter

[![Go Version](https://img.shields.io/badge/Go-1.25%2B-blue.svg)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![CI](https://github.com/rayven122/tumiki-mcp-http-adapter/workflows/CI/badge.svg)](https://github.com/rayven122/tumiki-mcp-http-adapter/actions/workflows/ci.yml)
[![Release](https://img.shields.io/github/v/release/rayven122/tumiki-mcp-http-adapter)](https://github.com/rayven122/tumiki-mcp-http-adapter/releases)
[![Go Report Card](https://goreportcard.com/badge/github.com/rayven122/tumiki-mcp-http-adapter)](https://goreportcard.com/report/github.com/rayven122/tumiki-mcp-http-adapter)

## 目次

- [概要](#概要)
- [インストール](#インストール)
- [使用例](#使用例)
- [コマンドラインオプション](#コマンドラインオプション)
- [開発](#開発)
- [ライセンス](#ライセンス)

---

## 概要

stdio MCP サーバーを HTTP エンドポイントとして公開する Go 実装プロキシサーバー

### 主な特徴

- ✅ **軽量**: シンプルな stdio プロキシ
- ✅ **即座に起動**: `--stdio`フラグだけで起動可能
- ✅ **動的設定**: HTTP ヘッダーから環境変数・引数を設定可能（streamable http 対応）
- ✅ **カスタムヘッダーマッピング**: 完全に自由なヘッダー名で環境変数・引数を指定可能

> **📖 技術的詳細**: システムアーキテクチャ、コンポーネント設計、セキュリティ設計などの詳細は [docs/DESIGN.md](docs/DESIGN.md) を参照してください。

---

## インストール

### ビルド済みバイナリ（推奨）

[GitHub Releases](https://github.com/rayven122/tumiki-mcp-http-adapter/releases) から最新版をダウンロードしてください。

#### macOS / Linux

```bash
# 最新版を自動ダウンロード＆インストール
curl -sL https://github.com/rayven122/tumiki-mcp-http-adapter/releases/latest/download/tumiki-mcp-http_$(uname -s)_$(uname -m).tar.gz | tar xz
sudo mv tumiki-mcp-http /usr/local/bin/

# 確認
tumiki-mcp-http --help
```

#### Windows

[Releases ページ](https://github.com/rayven122/tumiki-mcp-http-adapter/releases)から Windows 版の zip ファイルをダウンロードして展開してください。

### Go install

```bash
# Go がインストール済みの場合
go install github.com/rayven122/tumiki-mcp-http-adapter/cmd/tumiki-mcp-http@latest

# 確認
tumiki-mcp-http --help
```

**注意**: `$GOPATH/bin` (通常 `~/go/bin`) が PATH に含まれていることを確認してください。

### ソースからビルド

```bash
# リポジトリをクローン
git clone https://github.com/rayven122/tumiki-mcp-http-adapter.git
cd tumiki-mcp-http-adapter

# ビルド
task build

# バイナリを PATH に配置（オプション）
sudo cp bin/tumiki-mcp-http /usr/local/bin/

# 確認
tumiki-mcp-http --help
```

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

### ヘッダーマッピング（動的設定）

HTTP リクエストのヘッダーから環境変数やコマンド引数を動的に設定できます。

**ステップ 1: CLI 起動時にマッピングを定義**

```bash
tumiki-mcp-http --stdio "npx -y server-slack" \
  --header-env "X-Slack-Token=SLACK_TOKEN" \
  --header-arg "X-Team-Id=team-id" \
  --header-arg "X-Channel=channel"
```

**ステップ 2: HTTP リクエスト時に実際の値を渡す**

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

## コマンドラインオプション

### オプション一覧

| オプション                  | 説明                                                  | 必須 | 複数指定 |
| --------------------------- | ----------------------------------------------------- | ---- | -------- |
| `--stdio <command>`         | stdio モードで実行する MCP サーバーのコマンド         | ✅   | ❌       |
| `--env <KEY=VALUE>`         | デフォルト環境変数の設定                              | ❌   | ✅       |
| `--header-env <HEADER=ENV>` | HTTP ヘッダーから環境変数へのマッピング               | ❌   | ✅       |
| `--header-arg <HEADER=ARG>` | HTTP ヘッダーからコマンド引数へのマッピング           | ❌   | ✅       |
| `--log-level <level>`       | ログレベル（debug/info/warn/error、デフォルト: info） | ❌   | ❌       |

### 環境変数での設定

サーバーの起動設定は環境変数でも指定可能です。

| 環境変数 | 説明             | デフォルト |
| -------- | ---------------- | ---------- |
| `HOST`   | サーバーのホスト | `0.0.0.0`  |
| `PORT`   | サーバーのポート | `8080`     |

```bash
# 例: ローカルホストのみでポート3000で起動
HOST=127.0.0.1 PORT=3000 tumiki-mcp-http --stdio "npx -y server-filesystem /data"
```

---

## 開発

開発環境のセットアップ、テスト、コーディング規約などの詳細については、以下を参照してください：

**[開発ガイド (docs/DEVELOPMENT.md)](docs/DEVELOPMENT.md)**

---

## ライセンス

MIT License - 詳細は [LICENSE](LICENSE) を参照
