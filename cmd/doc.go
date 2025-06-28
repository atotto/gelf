package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/EkeMinusYou/gelf/internal/ai"
	"github.com/EkeMinusYou/gelf/internal/config"
	"github.com/EkeMinusYou/gelf/internal/doc"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var docCmd = &cobra.Command{
	Use:   "doc",
	Short: "Generate documentation using AI",
	Long:  `Analyzes source code and generates documentation using Vertex AI (Gemini).`,
	RunE:  runDoc,
}

var (
	docSrc      string
	docDst      string
	docTemplate string
	docFormat   string
	docModel    string
	docLanguage string
)


var docSuccessStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("2")). // グリーン
	Bold(true)

func init() {
	docCmd.Flags().StringVarP(&docSrc, "src", "s", "", "Source directory or file to analyze (required)")
	docCmd.Flags().StringVarP(&docDst, "dst", "d", "", "Output file path (required)")
	docCmd.Flags().StringVarP(&docTemplate, "template", "t", "", "Documentation template: readme, api, changelog, architecture, godoc (required)")
	docCmd.Flags().StringVarP(&docFormat, "format", "f", "markdown", "Output format: markdown, html, json")
	docCmd.Flags().StringVarP(&docModel, "model", "m", "", "Override default model for this generation")
	docCmd.Flags().StringVarP(&docLanguage, "language", "l", "", "Language for documentation generation (e.g., english, japanese)")

	// 必須フラグの設定
	docCmd.MarkFlagRequired("src")
	docCmd.MarkFlagRequired("dst")
	docCmd.MarkFlagRequired("template")
}

func runDoc(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	// 設定の読み込み
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// モデル設定
	if docModel != "" {
		cfg.ProModel = docModel
	} else {
		// ドキュメント生成には高品質なProモデルを使用
		cfg.FlashModel = cfg.ProModel
	}

	// 言語設定
	if docLanguage != "" {
		cfg.ReviewLanguage = docLanguage
	}

	// 入力パスの検証
	if err := validateSourcePath(docSrc); err != nil {
		return err
	}

	// テンプレートの検証
	if err := validateTemplate(docTemplate); err != nil {
		return err
	}

	// フォーマットの検証
	if err := validateFormat(docFormat); err != nil {
		return err
	}

	// 出力ディレクトリの作成
	if err := ensureOutputDirectory(docDst); err != nil {
		return err
	}

	// AIクライアントの作成
	aiClient, err := ai.NewVertexAIClient(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create AI client: %w", err)
	}

	// ドキュメント分析器の作成
	analyzer, err := doc.NewAnalyzer(docSrc, docTemplate)
	if err != nil {
		return fmt.Errorf("failed to create analyzer: %w", err)
	}

	fmt.Printf("Analyzing source code at: %s\n", docSrc)
	fmt.Printf("Generating %s documentation...\n", docTemplate)

	// ソースコードの分析
	sourceInfo, err := analyzer.Analyze()
	if err != nil {
		return fmt.Errorf("failed to analyze source code: %w", err)
	}

	// ドキュメント生成
	documentation, err := aiClient.GenerateDocumentation(ctx, sourceInfo, docTemplate, cfg.ReviewLanguage)
	if err != nil {
		return fmt.Errorf("failed to generate documentation: %w", err)
	}

	// 出力フォーマットの変換
	output, err := doc.FormatOutput(documentation, docFormat)
	if err != nil {
		return fmt.Errorf("failed to format output: %w", err)
	}

	// ファイルへの書き込み
	if err := os.WriteFile(docDst, []byte(output), 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	message := docSuccessStyle.Render(fmt.Sprintf("✓ Documentation generated successfully: %s", docDst))
	fmt.Println(message)

	return nil
}

func validateSourcePath(src string) error {
	if src == "" {
		return fmt.Errorf("source path is required")
	}

	if _, err := os.Stat(src); os.IsNotExist(err) {
		return fmt.Errorf("source path does not exist: %s", src)
	}

	return nil
}

func validateTemplate(template string) error {
	validTemplates := []string{"readme", "api", "changelog", "architecture", "godoc"}
	
	for _, valid := range validTemplates {
		if template == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid template: %s (valid options: %s)", template, strings.Join(validTemplates, ", "))
}

func validateFormat(format string) error {
	validFormats := []string{"markdown", "html", "json"}
	
	for _, valid := range validFormats {
		if format == valid {
			return nil
		}
	}

	return fmt.Errorf("invalid format: %s (valid options: %s)", format, strings.Join(validFormats, ", "))
}

func ensureOutputDirectory(dst string) error {
	dir := filepath.Dir(dst)
	if dir == "." {
		return nil
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	return nil
}