package migrate

import (
	"context"

	"github.com/MontFerret/cli/v2/internal/migration"
)

type (
	// Service performs target migration independently of command rendering.
	Service interface {
		Migrate(context.Context, migration.Options) (*migration.Result, error)
	}

	compatibilityService interface {
		CheckCompatibility(
			context.Context,
			migration.CompatibilityOptions,
		) (*migration.CompatibilityResult, error)
	}
)
