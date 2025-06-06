# CLAUDE.md

このファイルは、このリポジトリでコードを扱う際のClaude Code (claude.ai/code) へのガイダンスを提供します。

## プロジェクト概要

geminielfは、Vertex AI (Gemini) を使用してGitコミットメッセージを自動生成するGo製CLIツールです。ステージングされた変更を分析し、Bubble Teaで構築されたインタラクティブなTUIインターフェースを通じて適切なコミットメッセージを生成します。

## 開発環境セットアップ

devcontainerは`.devcontainer/setup.sh`を自動実行し、以下を設定します：
- 日本語ロケール設定 (ja_JP.UTF-8)
- aquaパッケージマネージャーをPATHに追加

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
  └── commit.go        # commitサブコマンド実装
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
- **モデル設定**: 環境変数 `GEMINIELF_DEFAULT_MODEL` で変更可能
- **入力**: 生のgit diff出力 (フィルタリングなし)
- **UIフレームワーク**: Bubble Tea (TUI用)
- **ユーザーインタラクション**: シンプルなYes/No確認 (初期版では編集機能なし)
- **エラーハンドリング**: 最小限 (初期版では優先度低)

## コマンド使用法

```bash
geminielf commit          # Vertex AIを使ってステージング済み変更をコミット
geminielf --help          # ヘルプ表示
```

## 開発コマンド

```bash
go mod init geminielf     # Goモジュール初期化
go build                  # プロジェクトビルド
go test ./...             # テスト実行
go mod tidy               # 依存関係整理
go run main.go commit     # アプリケーション実行 (commitサブコマンド)
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

## 環境変数

devcontainerの設定：
- `GOPROXY=direct`
- `GOPRIVATE=github.com/groove-x`

アプリケーションの環境変数：
- `GOOGLE_APPLICATION_CREDENTIALS` - サービスアカウントキーへのパス
- `VERTEXAI_PROJECT` - Google CloudプロジェクトID
- `VERTEXAI_LOCATION` - Vertex AIのロケーション (デフォルト: us-central1)
- `GEMINIELF_DEFAULT_MODEL` - 使用するGeminiモデル (デフォルト: gemini-2.5-flash-preview-05-20)