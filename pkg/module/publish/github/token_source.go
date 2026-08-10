package github

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

type (
	// TokenProvider supplies a GitHub access token without exposing it to output.
	TokenProvider interface {
		Token(context.Context) (string, error)
	}

	commandRunner interface {
		Run(context.Context, string, ...string) ([]byte, error)
	}

	// TokenSource resolves GitHub credentials from the environment or GitHub CLI.
	TokenSource struct {
		getenv func(string) string
		run    commandRunner
	}
)

// NewTokenSource constructs the default non-interactive credential resolver.
func NewTokenSource() *TokenSource {
	return &TokenSource{
		getenv: os.Getenv,
		run: commandRunnerFunc(func(ctx context.Context, name string, arguments ...string) ([]byte, error) {
			return exec.CommandContext(ctx, name, arguments...).Output()
		}),
	}
}

// Token resolves GH_TOKEN, GITHUB_TOKEN, or the current gh CLI credential.
func (s *TokenSource) Token(ctx context.Context) (string, error) {
	for _, name := range []string{"GH_TOKEN", "GITHUB_TOKEN"} {
		if token := strings.TrimSpace(s.getenv(name)); token != "" {
			return token, nil
		}
	}

	data, err := s.run.Run(ctx, "gh", "auth", "token", "--hostname", "github.com")
	if contextErr := ctx.Err(); contextErr != nil {
		return "", contextErr
	}

	if err == nil {
		if token := strings.TrimSpace(string(data)); token != "" {
			return token, nil
		}

		err = fmt.Errorf("GitHub CLI returned no credential")
	}

	return "", newAuthenticationError(fmt.Errorf("resolve GitHub credential: %w", err))
}
