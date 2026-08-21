package migration

import (
	"context"
	"fmt"

	"github.com/MontFerret/cli/v2/internal/buildinfo"
)

// Migrator plans and applies the currently supported project migration path.
type Migrator struct {
	planner planner
}

// New constructs a v1-to-v2 compatibility migrator.
func New(runner Runner) *Migrator {
	if runner == nil {
		runner = execGoRunner{}
	}

	return &Migrator{
		planner: newV1ToV2Planner(runner, buildinfo.FerretVersion),
	}
}

// Migrate creates a complete plan and applies it only in ModeApply.
func (migrator *Migrator) Migrate(ctx context.Context, options Options) (*Result, error) {
	if migrator == nil || migrator.planner == nil {
		return nil, fmt.Errorf("migration planner is not configured")
	}

	if options.Mode > ModePrint {
		return nil, fmt.Errorf("unsupported migration mode %d", options.Mode)
	}

	plan, err := migrator.planner.Plan(ctx, options)
	if err != nil {
		return nil, err
	}

	if options.Mode == ModeApply && len(plan.changes) > 0 {
		if err := ctx.Err(); err != nil {
			return nil, err
		}

		if err := commitMigrationChanges(plan.changes); err != nil {
			return nil, fmt.Errorf("commit migration: %w", err)
		}

		plan.result.Applied = true
	}

	return &plan.result, nil
}

// CheckCompatibility reports supported v1 FQL behavior changes without modifying source files.
func (migrator *Migrator) CheckCompatibility(
	ctx context.Context,
	options CompatibilityOptions,
) (*CompatibilityResult, error) {
	if migrator == nil {
		return nil, fmt.Errorf("compatibility checker is not configured")
	}

	return checkFQLCompatibility(ctx, options)
}
