package install

import (
	"context"
	"go/ast"
	"go/token"
	"io/fs"

	barnregistry "github.com/MontFerret/barn/pkg/registry"
)

const ferretCoreModulePath = "github.com/MontFerret/ferret/v2"

type (
	// Registry provides the registry operations used to resolve module releases.
	Registry interface {
		Module(context.Context, string) (*barnregistry.Module, error)
		Version(context.Context, string, string) (*barnregistry.Version, error)
	}

	// Runner executes Go toolchain commands in a project directory.
	Runner interface {
		Run(context.Context, string, ...string) ([]byte, error)
	}

	// Options controls installation into an existing Go application.
	Options struct {
		Reference string
		Directory string
	}

	// Result describes a resolved and validated project installation.
	Result struct {
		ID                  string
		Version             string
		PackagePath         string
		FerretConstraint    string
		ProjectFerret       string
		EditedFile          string
		Changed             bool
		SourceChanged       bool
		DependenciesChanged bool
	}

	projectInfo struct {
		Root          string
		ModulePath    string
		GoModPath     string
		GoSumPath     string
		FerretVersion string
	}

	goModuleInfo struct {
		Path    string        `json:"Path"`
		Version string        `json:"Version"`
		Main    bool          `json:"Main"`
		Dir     string        `json:"Dir"`
		GoMod   string        `json:"GoMod"`
		Replace *goModuleInfo `json:"Replace"`
		Origin  *goOriginInfo `json:"Origin"`
	}

	goOriginInfo struct {
		VCS    string `json:"VCS"`
		URL    string `json:"URL"`
		Subdir string `json:"Subdir"`
		Hash   string `json:"Hash"`
		Ref    string `json:"Ref"`
	}

	goPackageInfo struct {
		Dir        string        `json:"Dir"`
		ImportPath string        `json:"ImportPath"`
		GoFiles    []string      `json:"GoFiles"`
		CgoFiles   []string      `json:"CgoFiles"`
		Module     *goModuleInfo `json:"Module"`
	}

	goDownloadInfo struct {
		Path    string        `json:"Path"`
		Version string        `json:"Version"`
		Error   string        `json:"Error"`
		Origin  *goOriginInfo `json:"Origin"`
	}

	installRelease struct {
		Version            *barnregistry.Version
		HistoricalPackages map[string]struct{}
	}

	composition struct {
		Filename  string
		Directory string
		Package   string
		Source    []byte
		Mode      fs.FileMode
		File      *ast.File
		FileSet   *token.FileSet
		Call      *ast.CallExpr
		CoreAlias string
	}

	compositionRewrite struct {
		Source     []byte
		Changed    bool
		Registered bool
	}

	fileSnapshot struct {
		Path   string
		Data   []byte
		Mode   fs.FileMode
		Exists bool
	}

	fileChange struct {
		Before fileSnapshot
		After  []byte
		Mode   fs.FileMode
	}

	overlayDocument struct {
		Replace map[string]string `json:"Replace"`
	}
)
