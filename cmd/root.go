package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "gelf",
	Short: "AI-powered Git commit message generator using Vertex AI (Gemini)",
	Long: `gelf is a CLI tool that generates Git commit messages using Vertex AI (Gemini).
It analyzes staged changes and creates appropriate commit messages through an interactive TUI.`,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(commitCmd)
	rootCmd.AddCommand(reviewCmd)
}