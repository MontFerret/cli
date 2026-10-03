package runtime

import (
	"bytes"
	"context"
	"io"

	"github.com/MontFerret/api"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

// RunSource executes through the selected canonical runtime or isolated Worker
// adapter. Output remains available alongside an execution or cleanup error.
func RunSource(ctx context.Context, resources *Resources, src source.Source, params map[string]any) (io.ReadCloser, error) {
	if resources.Legacy != nil {
		return resources.Legacy.Run(ctx, src, params)
	}

	output, err := resources.Runtime.Run(ctx, api.NewSource(src.Name(), src.Content()), api.WithParams(params))

	return outputReader(output), err
}

func outputReader(output *api.Output) io.ReadCloser {
	if output == nil {
		return nil
	}

	return io.NopCloser(bytes.NewReader(output.Content))
}
