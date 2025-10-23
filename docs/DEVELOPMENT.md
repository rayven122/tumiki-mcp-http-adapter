# Tumiki MCP HTTP Adapter - 開発ガイド

## 開発環境のセットアップ

### 必要なツール

- **Go 1.25+** - プログラミング言語
- **[Task](https://taskfile.dev/)** - タスクランナー
- **[golangci-lint](https://golangci-lint.run/)** - Go 用リンター
- **[goimports](https://pkg.go.dev/golang.org/x/tools/cmd/goimports)** - import 文の整理

### 開発ツールのインストール

```bash
# 開発ツールの自動インストール
task install-tools
```

---

## 開発コマンド

利用可能なタスクを確認してください：

```bash
task --list
```

### 主要なコマンド

| コマンド | 説明 |
|---------|------|
| `task build` | バイナリをビルド |
| `task test` | テストの実行 |
| `task coverage` | カバレッジ付きテスト |
| `task fmt` | コードのフォーマット |
| `task lint` | リンターの実行 |
| `task check` | 全チェック（フォーマット・vet・リント・テスト） |
| `task clean` | ビルド成果物を削除 |

---

## ビルドせずに実行

開発中は、ビルド不要で直接実行できます：

```bash
# ビルドせずに直接実行
go run ./cmd/tumiki-mcp-http --stdio "npx -y @modelcontextprotocol/server-filesystem /data"

# 環境変数付きで実行
go run ./cmd/tumiki-mcp-http \
  --stdio "npx -y @modelcontextprotocol/server-github" \
  --env "GITHUB_TOKEN=ghp_xxxxx"

# ヘッダーマッピング付きで実行
go run ./cmd/tumiki-mcp-http \
  --stdio "npx -y server-slack" \
  --header-env "X-Slack-Token=SLACK_TOKEN" \
  --header-arg "X-Team-Id=team-id"
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

# レースディテクタ付き
go test -race ./...

# 特定パッケージのみ
go test ./internal/proxy
go test ./internal/process
```

### カバレッジレポート

```bash
# カバレッジ測定
task coverage

# HTMLレポート生成
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### テストポリシー

詳細なテストポリシーについては [CLAUDE.md](../CLAUDE.md) を参照してください。

- ✅ テスト可能な関数は 100% カバレッジを目指す
- ✅ 正常系・異常系・エッジケースを全てテスト
- ✅ テストケース名は「入力条件_期待される結果」の形式
- ✅ エラーハンドリングは必ずテスト

---

## コーディング規約

このプロジェクトでは以下の規約に従います：

### フォーマット

- **gofmt**: Go の標準フォーマッター
- **goimports**: import 文の自動整理

```bash
task fmt
```

### リンター

- **golangci-lint**: 総合的な静的解析ツール（設定: `.golangci.yml`）

```bash
task lint
```

### テスト

- **カバレッジ目標**: テスト可能な関数は 100%
- **テーブル駆動テスト**: 複数のテストケースを構造化
- **詳細**: [CLAUDE.md](../CLAUDE.md) を参照

### エラーハンドリング

- 明示的にエラーを返す
- `log.Fatal` は最小限に使用（main 関数のみ）
- エラーメッセージは具体的で分かりやすく

---

## コミット前のチェック

コミット前に必ず以下を実行してください：

```bash
task check
```

このコマンドで以下が実行されます：

1. コードのフォーマット（`gofmt`, `goimports`）
2. 静的解析（`go vet`）
3. リンター（`golangci-lint`）
4. 全テストの実行

---

## プロジェクト構成

```text
tumiki-mcp-http-adapter/
├── cmd/tumiki-mcp-http/     # メインアプリケーション
│   ├── main.go              # エントリーポイント
│   └── main_test.go         # CLI テスト
├── internal/                 # プライベートパッケージ
│   ├── proxy/               # HTTPサーバー、ヘッダー解析
│   │   ├── server.go
│   │   └── server_test.go
│   └── process/             # プロセス実行
│       ├── executor.go
│       └── executor_test.go
├── docs/                     # ドキュメント
│   ├── DESIGN.md            # 設計書
│   └── DEVELOPMENT.md       # 開発ガイド（本ドキュメント）
├── .golangci.yml            # リンター設定
├── Taskfile.yml             # タスク定義
├── go.mod                    # Go モジュール定義
├── CLAUDE.md                # 開発ガイドライン（テストポリシー）
└── README.md                # プロジェクト概要
```

---

## トラブルシューティング

### テストが失敗する

```bash
# 詳細なエラー情報を表示
go test -v ./...

# レースディテクタで並行処理の問題を検出
go test -race ./...
```

### リンターエラー

```bash
# 自動修正可能な問題を修正
task fmt

# リンターの詳細情報を表示
golangci-lint run --verbose
```

### ビルドが失敗する

```bash
# 依存関係を整理
go mod tidy

# ビルドキャッシュをクリア
go clean -cache

# 再ビルド
task build
```

---

## リリース

このプロジェクトは [GoReleaser](https://goreleaser.com/) と GitHub Actions を使って自動リリースされます。

### リリース手順

#### 1. リリース準備

```bash
# 最新のmainブランチを取得
git checkout main
git pull origin main

# すべてのテストが通ることを確認
task check
```

#### 2. バージョンタグの作成

```bash
# セマンティックバージョニングに従ってタグを作成
# MAJOR.MINOR.PATCH (例: v1.0.0, v1.2.3)

# タグを作成
git tag -a v1.0.0 -m "Release v1.0.0"

# タグをGitHubにpush
git push origin v1.0.0
```

#### 3. 自動ビルド＆リリース

タグをpushすると、GitHub Actionsが自動的に以下を実行します：

1. **クロスプラットフォームビルド**: macOS/Linux/Windows向けバイナリを生成
2. **アーカイブ作成**: tar.gz（Unix系）、zip（Windows）形式で圧縮
3. **チェックサム生成**: sha256チェックサムファイルを作成
4. **GitHub Releases作成**: リリースノートと共にバイナリを公開

#### 4. リリース確認

[Releases ページ](https://github.com/rayven122/tumiki-mcp-http-adapter/releases)でリリースが正常に公開されているか確認してください。

### バージョニング

[Semantic Versioning 2.0.0](https://semver.org/) に従います：

- **MAJOR**: 互換性のない API 変更
- **MINOR**: 後方互換性のある機能追加
- **PATCH**: 後方互換性のあるバグ修正

**例**:
- `v1.0.0`: 最初の安定版リリース
- `v1.1.0`: 新機能追加
- `v1.1.1`: バグ修正
- `v2.0.0`: 破壊的変更を含むメジャーアップデート

### プレリリース

ベータ版やリリース候補版は以下の形式でタグを作成します：

```bash
# ベータ版
git tag -a v1.0.0-beta.1 -m "Beta release v1.0.0-beta.1"

# リリース候補版
git tag -a v1.0.0-rc.1 -m "Release candidate v1.0.0-rc.1"

git push origin <tag>
```

GoReleaserは自動的にプレリリースとして扱います。

### リリース設定

リリース設定は以下のファイルで管理されています：

- **[.goreleaser.yaml](../.goreleaser.yaml)**: GoReleaserの設定
- **[.github/workflows/release.yml](../.github/workflows/release.yml)**: GitHub Actionsワークフロー

---

## 参考資料

- **[README.md](../README.md)** - プロジェクト概要と使用方法
- **[DESIGN.md](DESIGN.md)** - システムアーキテクチャと設計
- **[CLAUDE.md](../CLAUDE.md)** - テストポリシーとコーディング規約
- **[Taskfile.yml](../Taskfile.yml)** - タスク定義の詳細
- **[.golangci.yml](../.golangci.yml)** - リンター設定の詳細
- **[GoReleaser Documentation](https://goreleaser.com/)** - GoReleaserの公式ドキュメント
