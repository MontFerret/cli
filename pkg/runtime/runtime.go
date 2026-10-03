package runtime

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/MontFerret/ferret/v2/pkg/source"
)

func Run(ctx context.Context, opts Options, query source.Source, params map[string]any) (out io.ReadCloser, err error) {
	rt, err := New(ctx, opts)

	if err != nil {
		return nil, err
	}

	defer func() {
		if closeErr := rt.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close runtime: %w", closeErr))
		}
	}()

	return RunSource(ctx, rt, query, params)
}

func RunArtifact(ctx context.Context, opts Options, data []byte, params map[string]any) (out io.ReadCloser, err error) {
	if !IsBuiltinType(opts.Type) {
		return nil, ErrArtifactRequiresBuiltinRuntime
	}

	rt, err := New(ctx, opts)

	if err != nil {
		return nil, err
	}

	defer func() {
		if closeErr := rt.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close runtime: %w", closeErr))
		}
	}()

	return rt.builtin.RunArtifact(ctx, data, params)
}

func IsBuiltinType(name string) bool {
	return normalizeRuntimeType(name) == DefaultRuntime
}

func normalizeRuntimeType(name string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(name)), " ", "")
}
