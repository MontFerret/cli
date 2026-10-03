package wirehost

import (
	"context"

	"github.com/MontFerret/api"
)

// versionRuntime changes only metadata retrieval; execution and ownership stay native.
type versionRuntime struct {
	api.Runtime
	retrieve func(context.Context) (api.Version, error)
}

func (r *versionRuntime) Version(ctx context.Context) (api.Version, error) {
	return r.retrieve(ctx)
}
