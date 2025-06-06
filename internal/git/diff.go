package git

import (
	"os/exec"
	"strings"
)

func GetStagedDiff() (string, error) {
	cmd := exec.Command("git", "--no-pager", "diff", "--staged", "-U5")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(output)), nil
}

func CommitChanges(message string) error {
	cmd := exec.Command("git", "commit", "-m", message)
	return cmd.Run()
}