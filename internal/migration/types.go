package migration

import (
	"context"
	"io/fs"

	"golang.org/x/mod/modfile"

	"github.com/MontFerret/cli/v2/internal/goproject"
)

const (
	v1ModulePath     = "github.com/MontFerret/ferret"
	v2CompatPath     = "github.com/MontFerret/ferret/v2/compat"
	v2ModulePath     = "github.com/MontFerret/ferret/v2"
	defaultDirectory = "."
)

type (
	// Mode controls whether a migration plan is applied or only inspected.
	Mode uint8

	// Options selects the project directory and mutation mode.
	Options struct {
		Directory string
		Mode      Mode
	}

	// Change contains one deterministic planned file replacement.
	Change struct {
		Path         string
		Before       []byte
		After        []byte
		BeforeExists bool
	}

	// ManualAction identifies a v1 import that cannot be rewritten safely.
	ManualAction struct {
		Path       string
		ImportPath string
		Reason     string
		Line       int
	}

	// Result describes the complete migration plan and its application status.
	Result struct {
		Root                string
		GoModPath           string
		Changes             []Change
		ManualActions       []ManualAction
		ScannedFiles        int
		UpdatedImports      int
		FormattedFiles      int
		DependenciesChanged bool
		VendorDetected      bool
		Applied             bool
	}

	Runner = goproject.Runner

	planner interface {
		Plan(context.Context, Options) (*migrationPlan, error)
	}

	ferretVersionProvider func() (string, error)

	migrationProject struct {
		Root       string
		ModulePath string
		GoModPath  string
		GoSumPath  string
		GoModFile  *modfile.File
		GoFiles    []string
		GoMod      fileSnapshot
		GoSum      fileSnapshot
	}

	migrationPlan struct {
		result  Result
		changes []plannedChange
	}

	plannedChange struct {
		change Change
		before fileSnapshot
		mode   fs.FileMode
	}

	fileSnapshot struct {
		Path   string
		Data   []byte
		Mode   fs.FileMode
		Exists bool
	}

	sourcePlan struct {
		Changes        []plannedChange
		ManualActions  []ManualAction
		ScannedFiles   int
		UpdatedImports int
		FormattedFiles int
		HasCompat      bool
		RemainingV1    bool
	}

	goPackageInfo struct {
		ImportPath     string          `json:"ImportPath"`
		Dir            string          `json:"Dir"`
		GoFiles        []string        `json:"GoFiles"`
		CgoFiles       []string        `json:"CgoFiles"`
		IgnoredGoFiles []string        `json:"IgnoredGoFiles"`
		InvalidGoFiles []string        `json:"InvalidGoFiles"`
		TestGoFiles    []string        `json:"TestGoFiles"`
		XTestGoFiles   []string        `json:"XTestGoFiles"`
		Module         *goModuleInfo   `json:"Module"`
		Error          *goPackageError `json:"Error"`
	}

	goModuleInfo struct {
		Main bool `json:"Main"`
	}

	goPackageError struct {
		Err string `json:"Err"`
	}
)

const (
	// ModeApply atomically writes the planned project changes.
	ModeApply Mode = iota
	// ModeDryRun calculates the plan without modifying project files.
	ModeDryRun
	// ModePrint calculates the plan for unified-diff rendering without modifying project files.
	ModePrint
)
