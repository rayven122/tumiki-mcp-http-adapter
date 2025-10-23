# Tumiki MCP HTTP Adapter - 開発ガイドライン

このプロジェクトのコーディング規約とテストポリシーを定義します。

## テストポリシー

### カバレッジ要件

**原則: 単体テストは対象ファイルのカバレッジ100%を目指す**

#### テスト対象
以下の関数は必ず単体テストを作成すること:
- ✅ ビジネスロジックを含む全ての関数
- ✅ パブリック関数・メソッド
- ✅ 内部ヘルパー関数（ロジックを含む場合）
- ✅ データパース・変換関数
- ✅ バリデーション関数

#### テスト除外対象
以下は単体テストのカバレッジ要件から除外可能:
- ❌ `main()` 関数（エントリーポイント）
- ❌ サーバー起動関数（`startServer`など）
- ❌ ロガー初期化などのセットアップ関数
- ❌ `os.Exit()`や`log.Fatal()`を呼ぶコードパス

これらは統合テストやE2Eテストでカバーする。

### テストの品質基準

#### 0. テスト名の命名規則

**原則: テスト名は「入力条件_期待される結果」の形式で記述する**

テスト名は以下の要素を含むこと:
- ✅ **Given（前提条件）**: どのような入力・状態か
- ✅ **Then（期待結果）**: どうなるべきか

```go
// ❌ 悪い例: 入力の状態のみ
{name: "単一のマッピング", ...}
{name: "値にイコールを含む場合", ...}
{name: "空", ...}

// ✅ 良い例: 入力条件_期待される結果
{name: "単一のマッピング_正しくパースされる", ...}
{name: "値にイコールを含む場合_エラーを返す", ...}
{name: "空の入力_空のマップを返す", ...}
```

**命名パターン例:**

```go
// パターン1: 正常系
"正常な環境変数1つ_マップに変換される"
"複数の環境変数_全てマップに変換される"
"空の入力_空のマップを返す"

// パターン2: 異常系
"値に=を含む環境変数_エラーを返す"
"無効なフォーマット_無視される"
"nilの入力_エラーを返す"

// パターン3: エッジケース
"特殊文字を含む値_正しくエスケープされる"
"最大長の文字列_切り詰められずに処理される"
"Unicode文字列_正しくパースされる"
```

**テスト名で避けるべきこと:**
- ❌ 結果が不明確: "値にイコールを含む場合"（何が起こる？）
- ❌ 曖昧な表現: "正常な入力"（何が正常？）
- ❌ 実装詳細: "parseMapping関数が呼ばれる"
- ❌ 日本語と英語の混在: "正常なinput_成功する"

**推奨する形式:**
- ✅ `"入力条件_期待される動作"`
- ✅ `"前提条件_結果"`
- ✅ 日本語で統一（プロジェクトの標準）

#### 1. テストケースの網羅性
各関数のテストは以下を含むこと:

```go
func TestXxx(t *testing.T) {
    tests := []struct {
        name     string
        input    InputType
        expected OutputType
        wantError bool  // エラーケースの場合
    }{
        // ✅ 正常系: 典型的なケース
        {name: "正常な入力_期待する出力を返す", input: validInput, expected: validOutput},

        // ✅ 正常系: 境界値
        {name: "空の入力_空の結果を返す", input: emptyInput, expected: emptyOutput},
        {name: "最大値_正しく処理される", input: maxInput, expected: maxOutput},

        // ✅ 異常系: エラーケース
        {name: "無効な入力_エラーを返す", input: invalidInput, wantError: true},

        // ✅ エッジケース: 特殊な条件
        {name: "特殊文字を含む入力_エスケープされて処理される", input: specialInput, expected: specialOutput},
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

#### 2. エラーハンドリングのテスト

**必須**: エラーを返す関数は、エラーケースを必ずテストすること

```go
// ❌ 悪い例: エラーケースのテストなし
func TestParseEnvVars(t *testing.T) {
    result := parseEnvVars(ArrayFlags{"KEY=value"})
    // 正常系のみ
}

// ✅ 良い例: エラーケースもテスト
func TestParseEnvVars(t *testing.T) {
    tests := []struct {
        name      string
        input     ArrayFlags
        wantError bool
    }{
        {name: "正常", input: ArrayFlags{"KEY=value"}, wantError: false},
        {name: "エラー", input: ArrayFlags{"KEY=value=invalid"}, wantError: true},
    }
    // ...
}
```

#### 3. テストデータの品質

- ✅ テストケース名は日本語で明確に記述
- ✅ 各テストケースは独立して実行可能
- ✅ テストデータは実際の使用例に基づく
- ✅ エッジケースを必ず含める

### カバレッジ確認方法

#### テスト実行とカバレッジ測定
```bash
# テスト実行
go test -v ./cmd/tumiki-mcp-http/

# カバレッジ測定
go test -coverprofile=coverage.out ./cmd/tumiki-mcp-http/

# カバレッジ詳細表示
go tool cover -func=coverage.out

# HTMLレポート生成
go tool cover -html=coverage.out -o coverage.html
```

#### カバレッジ基準

```
テスト可能な関数: 100% 目標
├─ ビジネスロジック関数: 100% 必須
├─ パブリック関数: 100% 必須
├─ ヘルパー関数: 100% 必須
└─ 統合的な関数: 除外可能（main、startServerなど）

全体: 50%以上（統合テスト関数を除く）
```

### 実装例

#### parseEnvVars関数のテスト例
```go
func TestParseEnvVars(t *testing.T) {
    tests := []struct {
        name      string
        envVars   ArrayFlags
        expected  map[string]string
        wantError bool
    }{
        {
            name:      "単一の環境変数_マップに変換される",
            envVars:   ArrayFlags{"KEY=value"},
            expected:  map[string]string{"KEY": "value"},
            wantError: false,
        },
        {
            name:      "複数の環境変数_全てマップに変換される",
            envVars:   ArrayFlags{"KEY1=value1", "KEY2=value2"},
            expected:  map[string]string{"KEY1": "value1", "KEY2": "value2"},
            wantError: false,
        },
        {
            name:      "値に=を含む環境変数_エラーを返す",
            envVars:   ArrayFlags{"KEY=value=invalid"},
            expected:  nil,
            wantError: true,
        },
        {
            name:      "イコールなしの無効フォーマット_無視される",
            envVars:   ArrayFlags{"INVALID"},
            expected:  map[string]string{},
            wantError: false,
        },
        {
            name:      "空の入力_空のマップを返す",
            envVars:   ArrayFlags{},
            expected:  map[string]string{},
            wantError: false,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := parseEnvVars(tt.envVars)
            if tt.wantError {
                if err == nil {
                    t.Errorf("expected error but got none")
                }
            } else {
                if err != nil {
                    t.Errorf("unexpected error: %v", err)
                }
                if !reflect.DeepEqual(result, tt.expected) {
                    t.Errorf("got %v, want %v", result, tt.expected)
                }
            }
        })
    }
}
```

## コーディング規約

### Go標準スタイル

- `gofmt`でフォーマット
- `golint`に従う
- エラーハンドリングは明示的に行う

### エラーハンドリング

```go
// ✅ 良い例: エラーを返す
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

## 開発ワークフロー

### 新機能追加時

1. **仕様確認**: 要件を明確にする
2. **テスト設計**: テストケースを先に考える（TDD推奨）
3. **実装**: 機能を実装
4. **テスト作成**: カバレッジ100%を目指してテストを作成
5. **カバレッジ確認**: `go test -cover`で確認
6. **コミット**: テストが全て通ることを確認してからコミット

### バグ修正時

1. **再現テスト作成**: バグを再現するテストを先に作成
2. **修正**: バグを修正
3. **テスト確認**: 再現テストが通ることを確認
4. **リグレッションテスト**: 既存のテストが全て通ることを確認

## CI/CD

### 必須チェック

- ✅ 全てのテストが成功すること
- ✅ カバレッジが基準を満たすこと
- ✅ `gofmt`が適用されていること
- ✅ `go vet`でエラーがないこと

```bash
# CIで実行するコマンド例
go fmt ./...
go vet ./...
go test -v -race -coverprofile=coverage.out ./...
go tool cover -func=coverage.out
```

## まとめ

このプロジェクトでは**テストファーストの開発**を推奨します。

- 🎯 テスト可能な関数は100%カバレッジを目指す
- 🧪 正常系・異常系・エッジケースを全てテスト
- 📝 テストケース名は日本語で明確に
- 🔍 エラーハンドリングは必ずテスト
- ✅ コミット前に必ずテストを実行

良いテストは良いコードの証です。
