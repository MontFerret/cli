package github

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestTokenSourceUsesEnvironmentPrecedence(t *testing.T) {
	source := &TokenSource{
		getenv: func(name string) string {
			return map[string]string{"GH_TOKEN": " gh-token ", "GITHUB_TOKEN": "github-token"}[name]
		},
		run: commandRunnerFunc(func(context.Context, string, ...string) ([]byte, error) {
			t.Fatal("GitHub CLI ran despite an environment token")

			return nil, nil
		}),
	}

	token, err := source.Token(context.Background())
	if err != nil || token != "gh-token" {
		t.Fatalf("unexpected token resolution: token=%q err=%v", token, err)
	}
}

func TestTokenSourceFallsBackToGitHubCLI(t *testing.T) {
	var command string
	var arguments []string
	source := &TokenSource{
		getenv: func(string) string { return "" },
		run: commandRunnerFunc(func(_ context.Context, name string, args ...string) ([]byte, error) {
			command, arguments = name, args

			return []byte("cli-token\n"), nil
		}),
	}

	token, err := source.Token(context.Background())
	if err != nil || token != "cli-token" || command != "gh" || strings.Join(arguments, " ") != "auth token --hostname github.com" {
		t.Fatalf("unexpected CLI token resolution: token=%q command=%q args=%q err=%v", token, command, arguments, err)
	}
}

func TestTokenSourceReturnsActionableAuthenticationError(t *testing.T) {
	source := &TokenSource{
		getenv: func(string) string { return "" },
		run: commandRunnerFunc(func(context.Context, string, ...string) ([]byte, error) {
			return nil, errors.New("gh is unavailable")
		}),
	}

	_, err := source.Token(context.Background())
	if err == nil || !strings.Contains(err.Error(), "GH_TOKEN") || !strings.Contains(err.Error(), "gh auth login") {
		t.Fatalf("unexpected authentication error: %v", err)
	}
}
