package cmd

import (
	"context"
	"fmt"

	"github.com/EkeMinusYou/gelf/internal/ai"
	"github.com/EkeMinusYou/gelf/internal/config"
	"github.com/EkeMinusYou/gelf/internal/git"
	"github.com/EkeMinusYou/gelf/internal/ui"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"
)

var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Review code changes using AI",
	Long:  `Analyzes git diff and provides code review feedback using Vertex AI (Gemini).`,
	RunE:  runReview,
}

var (
	reviewStaged bool
	reviewModel  string
	noStyle      bool
)

func init() {
	reviewCmd.Flags().BoolVar(&reviewStaged, "staged", false, "Review staged changes instead of unstaged changes")
	reviewCmd.Flags().StringVar(&reviewModel, "model", "", "Override default model for this review")
	reviewCmd.Flags().BoolVar(&noStyle, "no-style", false, "Disable markdown styling")
}

func runReview(cmd *cobra.Command, args []string) error {
	ctx := context.Background()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	if reviewModel != "" {
		cfg.FlashModel = reviewModel
	}

	var diff string
	if reviewStaged {
		diff, err = git.GetStagedDiff()
		if err != nil {
			return fmt.Errorf("failed to get staged changes: %w", err)
		}
	} else {
		diff, err = git.GetUnstagedDiff()
		if err != nil {
			return fmt.Errorf("failed to get unstaged changes: %w", err)
		}
	}

	if diff == "" {
		warningStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("3")).
			Bold(true).
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("3")).
			Padding(0, 1).
			Margin(1, 0)

		var message string
		if reviewStaged {
			message = warningStyle.Render("⚠ No staged changes found. Please stage some changes first with 'git add'.")
		} else {
			message = warningStyle.Render("⚠ No unstaged changes found.")
		}
		fmt.Print(message + "\n")
		return nil
	}

	aiClient, err := ai.NewVertexAIClient(ctx, cfg)
	if err != nil {
		return fmt.Errorf("failed to create AI client: %w", err)
	}

	// Use TUI for loading experience
	reviewTUI := ui.NewReviewTUI(aiClient, diff, noStyle)
	review, err := reviewTUI.Run()
	if err != nil {
		return fmt.Errorf("failed to generate code review: %w", err)
	}

	// Style and display the review
	if noStyle {
		fmt.Print(review + "\n")
	} else {
		// Create a glamour renderer with auto-detected style
		renderer, err := glamour.NewTermRenderer(
			glamour.WithAutoStyle(),
			glamour.WithWordWrap(80),
		)
		if err != nil {
			// Fallback to plain text if glamour fails
			fmt.Print(review + "\n")
			return nil
		}

		styled, err := renderer.Render(review)
		if err != nil {
			// Fallback to plain text if rendering fails
			fmt.Print(review + "\n")
			return nil
		}

		fmt.Print(styled)
	}
	return nil
}