package install

import (
	"fmt"

	"github.com/MontFerret/cli/v2/internal/buildinfo"
)

func currentFerretVersion() (string, error) {
	version, err := buildinfo.FerretVersion()
	if err != nil {
		return "", err
	}

	if _, err := parseProjectFerretVersion(version); err != nil {
		return "", fmt.Errorf("CLI build reports an invalid Ferret dependency: %w", err)
	}

	return version, nil
}
