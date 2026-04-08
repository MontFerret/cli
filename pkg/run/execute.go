package run

import (
	"context"
	"fmt"
	"io"

	cliruntime "github.com/MontFerret/cli/v2/pkg/runtime"
)

func Execute(ctx context.Context, opts cliruntime.Options, params map[string]any, input *Input) (io.ReadCloser, error) {
	if input == nil {
		return nil, fmt.Errorf("run input is nil")
	}

	if len(input.Artifact) > 0 {
		return cliruntime.RunArtifact(ctx, opts, input.Artifact, params)
	}

	if input.Source == nil {
		return nil, fmt.Errorf("run source is nil")
	}

	return cliruntime.Run(ctx, opts, input.Source, params)
}
