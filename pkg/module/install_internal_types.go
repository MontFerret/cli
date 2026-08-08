package module

import (
	"go/ast"
	"go/token"
	"io/fs"

	barnregistry "github.com/MontFerret/barn/pkg/registry"
)

const ferretCoreModulePath = "github.com/MontFerret/ferret/v2"

type (
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
