package run

import (
	"context"
	"fmt"

	"github.com/MontFerret/api"
	"github.com/MontFerret/ferret/v2/pkg/source"

	cliruntime "github.com/MontFerret/cli/v2/pkg/runtime"
)

// Execute returns caller-owned output; consumption observes the terminal result.
func Execute(ctx context.Context, opts cliruntime.Options, params map[string]any, input *Input) (api.Output, error) {
	if input == nil {
		return nil, fmt.Errorf("run input is nil")
	}

	if len(input.Artifact) > 0 {
		return cliruntime.RunArtifact(ctx, opts, input.Artifact, params)
	}

	if input.Source.ID() == (source.ID{}) {
		return nil, fmt.Errorf("run source is not set")
	}

	return cliruntime.Run(ctx, opts, input.Source, params)
}
