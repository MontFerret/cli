package migration

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type execGoRunner struct{}

func (execGoRunner) Run(ctx context.Context, directory string, args ...string) ([]byte, error) {
	command := exec.CommandContext(ctx, "go", args...)
	command.Dir = directory
	command.Env = append(os.Environ(), "GOWORK=off")

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr

	if err := command.Run(); err != nil {
		details := strings.TrimSpace(strings.Join([]string{stdout.String(), stderr.String()}, "\n"))
		if details == "" {
			return nil, fmt.Errorf("go %s: %w", strings.Join(args, " "), err)
		}

		return nil, fmt.Errorf("go %s: %w\n%s", strings.Join(args, " "), err, details)
	}

	return stdout.Bytes(), nil
}
