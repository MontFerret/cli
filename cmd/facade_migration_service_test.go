package cmd

import (
	"context"

	"github.com/MontFerret/cli/v2/internal/migration"
)

type facadeMigrationService struct{}

func (facadeMigrationService) Migrate(context.Context, migration.Options) (*migration.Result, error) {
	return nil, nil
}
