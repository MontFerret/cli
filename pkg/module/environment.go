package module

import (
	"fmt"
	"runtime/debug"
	"strings"
)

const ferretModulePath = "github.com/MontFerret/ferret/v2"

// CurrentScaffoldEnvironment reads the Ferret and Go versions embedded in the running CLI.
func CurrentScaffoldEnvironment() (ScaffoldEnvironment, error) {
	build, ok := debug.ReadBuildInfo()
	if !ok {
		return ScaffoldEnvironment{}, fmt.Errorf("read CLI build information")
	}

	goVersion := strings.TrimPrefix(build.GoVersion, "go")
	if goVersion == "" {
		return ScaffoldEnvironment{}, fmt.Errorf("CLI build does not report a Go version")
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
			return ScaffoldEnvironment{}, fmt.Errorf("CLI build does not report a released Ferret dependency version")
		}

		return ScaffoldEnvironment{GoVersion: goVersion, FerretVersion: version}, nil
	}

	return ScaffoldEnvironment{}, fmt.Errorf("CLI build does not include %s", ferretModulePath)
}
