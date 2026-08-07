package module

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Git executes read-only local Git inspection commands.
type Git struct{}

// NewGit constructs a local Git command runner.
func NewGit() *Git {
	return &Git{}
}

// Run executes Git in directory and returns trimmed standard output.
func (g *Git) Run(ctx context.Context, directory string, args ...string) (string, error) {
	commandArgs := append([]string{"-C", directory}, args...)
	command := exec.CommandContext(ctx, "git", commandArgs...)
	command.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "LC_ALL=C")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message == "" {
			message = err.Error()
		}

		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), message)
	}

	return strings.TrimSpace(stdout.String()), nil
}
