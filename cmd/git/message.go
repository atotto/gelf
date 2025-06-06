package git

import (
	"context"
	"fmt"

	"gelf/internal/ai"
	"gelf/internal/config"
	"gelf/internal/git"

	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var messageCmd = &cobra.Command{
	Use:   "message",
	Short: "Generate commit message and output to stdout",
	Long:  "Analyzes staged changes and generates commit message using Vertex AI (Gemini). Outputs only the message for external tool integration.",
	RunE:  runMessage,
}

var (
	dryRun bool
	model  string
)

func init() {
	messageCmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show diff along with generated message")
	messageCmd.Flags().StringVar(&model, "model", "", "Override default model for this generation")
}

func runMessage(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if model != "" {
		cfg.Model = model
	}

	diff, err := git.GetStagedDiff()
	if err != nil {
		return fmt.Errorf("failed to get staged changes: %w", err)
	}

	if diff == "" {
		warningStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("3")).
			Bold(true).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("3")).
			Padding(0, 1).
			Margin(1, 0)
		
		message := warningStyle.Render("⚠ No staged changes found. Please stage some changes first with 'git add'.")
		fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", message)
		return fmt.Errorf("no staged changes")
	}

	if dryRun {
		fmt.Fprintf(cmd.ErrOrStderr(), "=== Staged Changes ===\n%s\n\n", diff)
	}

	aiClient, err := ai.NewVertexAIClient(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create AI client: %w", err)
	}

	message, err := aiClient.GenerateCommitMessage(ctx, diff)
	if err != nil {
		return fmt.Errorf("failed to generate commit message: %w", err)
	}

	fmt.Print(message)
	return nil
}