package git

import (
	"github.com/spf13/cobra"
)

var GitCmd = &cobra.Command{
	Use:   "git",
	Short: "Git-related AI tasks",
	Long:  "AI-powered Git operations including commit message generation and code review.",
}

func init() {
	GitCmd.AddCommand(commitCmd)
	GitCmd.AddCommand(messageCmd)
}