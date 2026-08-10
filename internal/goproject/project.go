package goproject

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrNoModule reports that Go cannot find a containing module.
var ErrNoModule = errors.New("current directory is not inside a Go module")

type (
	// Runner executes Go toolchain commands in a project directory.
	Runner interface {
		Run(context.Context, string, ...string) ([]byte, error)
	}

	// Project identifies the containing Go module and its dependency files.
	Project struct {
		Root       string
		ModulePath string
		GoModPath  string
		GoSumPath  string
	}

	moduleInfo struct {
		Path string `json:"Path"`
	}
)

// Discover locates and validates the Go module containing directory.
func Discover(ctx context.Context, runner Runner, directory string) (*Project, error) {
	if strings.TrimSpace(directory) == "" {
		directory = "."
	}

	absDirectory, err := filepath.Abs(directory)
	if err != nil {
		return nil, fmt.Errorf("resolve project directory: %w", err)
	}

	goModOutput, err := runner.Run(ctx, absDirectory, "env", "GOMOD")
	if err != nil {
		return nil, fmt.Errorf("locate project go.mod: %w", err)
	}

	goModPath := strings.TrimSpace(string(goModOutput))
	if goModPath == "" || goModPath == os.DevNull {
		return nil, ErrNoModule
	}

	goModPath, err = filepath.Abs(goModPath)
	if err != nil {
		return nil, fmt.Errorf("resolve go.mod path: %w", err)
	}

	root := filepath.Dir(goModPath)
	moduleOutput, err := runner.Run(ctx, root, "list", "-m", "-json")
	if err != nil {
		return nil, fmt.Errorf("inspect project module: %w", err)
	}

	var module moduleInfo
	if err := json.NewDecoder(bytes.NewReader(moduleOutput)).Decode(&module); err != nil {
		return nil, fmt.Errorf("decode project module metadata: %w", err)
	}

	if module.Path == "" {
		return nil, fmt.Errorf("project go.mod does not declare a module path")
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return &Project{
		Root:       root,
		ModulePath: module.Path,
		GoModPath:  goModPath,
		GoSumPath:  filepath.Join(root, "go.sum"),
	}, nil
}
