package scaffold

import (
	"fmt"
	"runtime/debug"
	"strings"
)

const ferretModulePath = "github.com/MontFerret/ferret/v2"

// CurrentEnvironment reads the Ferret and Go versions embedded in the running CLI.
func CurrentEnvironment() (Environment, error) {
	build, ok := debug.ReadBuildInfo()
	if !ok {
		return Environment{}, fmt.Errorf("read CLI build information")
	}

	goVersion := strings.TrimPrefix(build.GoVersion, "go")
	if goVersion == "" {
		return Environment{}, fmt.Errorf("CLI build does not report a Go version")
	}

	for _, dependency := range build.Deps {
		if dependency.Path != ferretModulePath {
			continue
		}

		version := dependency.Version
		if dependency.Replace != nil && dependency.Replace.Version != "" {
			version = dependency.Replace.Version
		}

		if version == "" || version == "(devel)" {
			return Environment{}, fmt.Errorf("CLI build does not report a released Ferret dependency version")
		}

		return Environment{GoVersion: goVersion, FerretVersion: version}, nil
	}

	return Environment{}, fmt.Errorf("CLI build does not include %s", ferretModulePath)
}
