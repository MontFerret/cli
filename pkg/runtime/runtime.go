package runtime

import (
	"context"
	"errors"
	"strings"

	"github.com/MontFerret/api"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

// Run transfers the invocation runtime to its output. Consume or Close it before cancelling ctx.
func Run(ctx context.Context, opts Options, query source.Source, params map[string]any) (api.Output, error) {
	rt, err := New(ctx, opts)

	if err != nil {
		return nil, err
	}

	output, err := RunSource(ctx, rt, query, params)
	if err != nil {
		return nil, errors.Join(err, rt.Close())
	}

	return newOwnedOutput(ctx, output, rt), nil
}

// RunArtifact executes a builtin artifact and transfers invocation resources to its output.
func RunArtifact(ctx context.Context, opts Options, data []byte, params map[string]any) (api.Output, error) {
	if !IsBuiltinType(opts.Type) {
		return nil, ErrArtifactRequiresBuiltinRuntime
	}

	rt, err := New(ctx, opts)

	if err != nil {
		return nil, err
	}

	output, err := rt.builtin.RunArtifact(ctx, data, params)
	if err != nil {
		return nil, errors.Join(err, rt.Close())
	}

	return newOwnedOutput(ctx, output, rt), nil
}

func IsBuiltinType(name string) bool {
	return normalizeRuntimeType(name) == DefaultRuntime
}

func normalizeRuntimeType(name string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(name)), " ", "")
}
