# CLAUDE.md

このファイルは、このリポジトリでコードを扱う際のClaude Code (claude.ai/code) へのガイダンスを提供します。

## プロジェクト概要

gelfは、Vertex AI (Gemini) を使用してGitコミットメッセージを自動生成するGo製CLIツールです。ステージングされた変更を分析し、Bubble Teaで構築されたインタラクティブなTUIインターフェースを通じて適切なコミットメッセージを生成します。

## アプリケーションアーキテクチャ

### 核となる機能
ツールは以下のシンプルなワークフローに従います：
1. **ステージング変更をチェック** - `git diff --staged` でステージングされた変更を取得
2. **コミットメッセージ生成** - diffをVertex AI (Gemini) に送信してメッセージ生成
3. **ユーザー確認** - 生成されたメッセージをTUIでYes/Noプロンプトと共に表示
4. **コミット実行** - ユーザーが承認した場合に変更をコミット

### プロジェクト構造
```
cmd/
  ├── root.go          # ルートコマンド定義
  └── git/
      ├── git.go       # gitサブコマンドグループ
      ├── commit.go    # git commitサブコマンド実装
      └── message.go   # git messageサブコマンド実装
internal/
  ├── git/
  │   └── diff.go      # Git操作 (git diff --staged)
  ├── ai/
  │   └── vertex.go    # メッセージ生成のためのVertex AI統合
  ├── ui/
  │   └── tui.go       # Bubble Tea TUI実装
  └── config/
      └── config.go    # 設定管理 (APIキーなど)
main.go               # アプリケーションエントリーポイント
```

### TUIフロー
1. **起動** - ステージング変更の確認
2. **ローディング** - Vertex AI呼び出し中にスピナー表示
3. **確認** - 生成されたメッセージを[y/n]プロンプトと共に表示
4. **結果** - コミット成功/失敗を表示

### 技術仕様
- **対象**: ステージング済み変更のみ (`git diff --staged`)
- **AIプロバイダー**: Vertex AI (Geminiモデル)
- **デフォルトモデル**: gemini-2.5-flash-preview-05-20
- **モデル設定**: 環境変数 `GELF_DEFAULT_MODEL` で変更可能
- **入力**: 生のgit diff出力 (フィルタリングなし)
- **UIフレームワーク**: Bubble Tea (TUI用)
- **ユーザーインタラクション**: シンプルなYes/No確認 (初期版では編集機能なし)
- **エラーハンドリング**: 最小限 (初期版では優先度低)

## コマンド使用法

```bash
# コミット関連
gelf commit                    # Vertex AIを使ってステージング済み変更をコミット（TUI付き）
gelf commit --dry-run          # コミットメッセージ生成のみ（diffも表示）
gelf commit --dry-run --quiet  # メッセージ生成のみ（外部ツール連携用）
gelf commit --model MODEL      # 一時的にモデルを変更

# ヘルプ
gelf --help          # ヘルプ表示
gelf commit --help   # コミットコマンドのヘルプ
```

## 開発コマンド

```bash
go mod init gelf     # Goモジュール初期化
go build                  # プロジェクトビルド
go test ./...             # テスト実行
go mod tidy               # 依存関係整理
go run main.go commit        # アプリケーション実行 (commitコマンド)
go run main.go commit --dry-run  # メッセージ生成のみのテスト
```

## 依存関係

必要な主要Goモジュール：
- `github.com/charmbracelet/bubbletea` - TUIフレームワーク
- `github.com/charmbracelet/lipgloss` - スタイリングとレイアウト
- `google.golang.org/genai` - Vertex AIクライアント
- `github.com/spf13/cobra` - CLIフレームワーク (サブコマンド実装用)

## 設定

アプリケーションには以下のVertex AI設定が必要です：
- Google Cloud プロジェクトID
- Vertex AI API認証情報
- モデル選択 (デフォルト: gemini-2.5-flash-preview-05-20)

### 設定ファイル

以下の場所で`gelf.yml`設定を管理できます（優先順）：

1. `./gelf.yml` - カレントディレクトリ（プロジェクト固有設定）
2. `$XDG_CONFIG_HOME/gelf/gelf.yml` - XDG設定ディレクトリ
3. `~/.config/gelf/gelf.yml` - XDG_CONFIG_HOMEが未設定の場合
4. `~/.gelf.yml` - ホームディレクトリ（従来形式）

```yaml
vertex_ai:
  project_id: "your-gcp-project-id"
  location: "us-central1"

gelf:
  default_model: "gemini-2.5-flash-preview-05-20"
```

設定の優先順位（高い順）：
1. 環境変数
2. 設定ファイル
3. デフォルト値

## 環境変数

devcontainerの設定：
- `GOPROXY=direct`
- `GOPRIVATE=github.com/groove-x`

アプリケーションの環境変数：
- `GELF_CREDENTIALS` - サービスアカウントキーへのパス（GOOGLE_APPLICATION_CREDENTIALSより優先）
- `GOOGLE_APPLICATION_CREDENTIALS` - サービスアカウントキーへのパス
- `VERTEXAI_PROJECT` - Google CloudプロジェクトID
- `VERTEXAI_LOCATION` - Vertex AIのロケーション (デフォルト: us-central1)

モデル設定は設定ファイル（gelf.yml）でのみ変更可能です。

## 覚書

- 仕様に変更があった場合は、README.md を更新すること

## 開発ガイドライン

- 変更後に、コンパイルしたときにエラーにならないように、staticcheck コマンドを使ってください。