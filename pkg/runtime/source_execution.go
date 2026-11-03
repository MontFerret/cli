package runtime

import (
	"context"

	"github.com/MontFerret/api"
	"github.com/MontFerret/ferret/v2/pkg/source"
)

// RunSource borrows resources for execution. The caller must settle the returned
// output before closing resources; terminal failures are observed by consumption.
func RunSource(ctx context.Context, resources *Resources, src source.Source, params map[string]any) (api.Output, error) {
	return resources.Runtime.Run(ctx, api.NewSource(src.Name(), src.Content()), api.WithParams(params))
}
