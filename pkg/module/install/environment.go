package install

import (
	"fmt"
	"runtime/debug"
)

func currentFerretVersion() (string, error) {
	build, ok := debug.ReadBuildInfo()
	if !ok {
		return "", fmt.Errorf("read CLI build information")
	}

	for _, dependency := range build.Deps {
		if dependency.Path != ferretCoreModulePath {
			continue
		}

		version := dependency.Version
		if dependency.Replace != nil && dependency.Replace.Version != "" {
			version = dependency.Replace.Version
		}
		if version == "" || version == "(devel)" {
			return "", fmt.Errorf("CLI build does not report a released Ferret dependency version")
		}
		if _, err := parseProjectFerretVersion(version); err != nil {
			return "", fmt.Errorf("CLI build reports an invalid Ferret dependency: %w", err)
		}

		return version, nil
	}

	return "", fmt.Errorf("CLI build does not include %s", ferretCoreModulePath)
}
