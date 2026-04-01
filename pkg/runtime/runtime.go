package runtime

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/MontFerret/ferret/v2/pkg/source"
)

type Runtime interface {
	Version(ctx context.Context) (string, error)

	Run(ctx context.Context, query *source.Source, params map[string]any) (io.ReadCloser, error)
	RunArtifact(ctx context.Context, data []byte, params map[string]any) (io.ReadCloser, error)
}

func New(opts Options) (Runtime, error) {
	name := normalizeRuntimeType(opts.Type)

	if IsBuiltinType(name) {
		return NewBuiltin(opts)
	}

	u, err := url.Parse(name)

	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}

	return NewRemote(*u, opts), nil
}

func Run(ctx context.Context, opts Options, query *source.Source, params map[string]any) (io.ReadCloser, error) {
	rt, err := New(opts)

	if err != nil {
		return nil, err
	}

	return rt.Run(ctx, query, params)
}

func RunArtifact(ctx context.Context, opts Options, data []byte, params map[string]any) (io.ReadCloser, error) {
	rt, err := New(opts)

	if err != nil {
		return nil, err
	}

	return rt.RunArtifact(ctx, data, params)
}

func IsBuiltinType(name string) bool {
	return normalizeRuntimeType(name) == DefaultRuntime
}

func normalizeRuntimeType(name string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(name)), " ", "")
}
