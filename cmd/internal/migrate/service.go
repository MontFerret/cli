package migrate

import (
	"context"

	"github.com/MontFerret/cli/v2/internal/migration"
)

// Service performs the project migration independently of command rendering.
type Service interface {
	Migrate(context.Context, migration.Options) (*migration.Result, error)
}
