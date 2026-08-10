package buildinfo

import (
	"fmt"
	"runtime/debug"
)

// FerretModulePath is the canonical Go module path embedded by this CLI.
const FerretModulePath = "github.com/MontFerret/ferret/v2"

// FerretVersion returns the released Ferret version embedded in the CLI build.
func FerretVersion() (string, error) {
	build, ok := debug.ReadBuildInfo()
	if !ok {
		return "", fmt.Errorf("read CLI build information")
	}

	for _, dependency := range build.Deps {
		if dependency.Path != FerretModulePath {
			continue
		}

		version := dependency.Version
		if dependency.Replace != nil && dependency.Replace.Version != "" {
			version = dependency.Replace.Version
		}

		if version == "" || version == "(devel)" {
			return "", fmt.Errorf("CLI build does not report a released Ferret dependency version")
		}

		return version, nil
	}

	return "", fmt.Errorf("CLI build does not include %s", FerretModulePath)
}
