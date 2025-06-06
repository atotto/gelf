package cmd

import (
	"context"
	"fmt"

	"geminielf/internal/ai"
	"geminielf/internal/config"
	"geminielf/internal/git"
	"geminielf/internal/ui"

	"github.com/spf13/cobra"
)

var commitCmd = &cobra.Command{
	Use:   "commit",
	Short: "Generate and commit with an AI-powered commit message",
	Long:  `Analyzes staged changes and generates a commit message using Vertex AI (Gemini).`,
	RunE:  runCommit,
}

func runCommit(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	diff, err := git.GetStagedDiff()
	if err != nil {
		return fmt.Errorf("failed to get staged changes: %w", err)
	}

	if diff == "" {
		fmt.Println("No staged changes found. Please stage some changes first with 'git add'.")
		return nil
	}

	aiClient, err := ai.NewVertexAIClient(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create AI client: %w", err)
	}

	tui := ui.NewTUI(aiClient, diff)
	if err := tui.Run(); err != nil {
		return fmt.Errorf("TUI error: %w", err)
	}

	return nil
}