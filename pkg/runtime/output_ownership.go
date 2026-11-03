package runtime

import (
	"context"
	"io"

	"github.com/MontFerret/api"
)

// newOwnedOutput settles delivery before releasing parents in dependency order.
func newOwnedOutput(ctx context.Context, output api.Output, owners ...io.Closer) api.Output {
	return newManagedOutput(ctx, output.Metadata(), output.Consume, nil, func() error {
		err := output.Close()
		for _, owner := range owners {
			err = joinOutputErrors(err, owner.Close())
		}

		return err
	})
}
