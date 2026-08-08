package module

import (
	"fmt"
	"strings"

	"github.com/Masterminds/semver/v3"
)

func parseInstallReference(reference string) (string, string, error) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return "", "", fmt.Errorf("module reference is empty")
	}

	if strings.Count(reference, "@") > 1 {
		return "", "", fmt.Errorf("invalid module reference %q: expected <owner>/<name>[@version]", reference)
	}

	id, version, hasVersion := strings.Cut(reference, "@")
	if id == "" || (hasVersion && version == "") {
		return "", "", fmt.Errorf("invalid module reference %q: expected <owner>/<name>[@version]", reference)
	}

	if !hasVersion {
		return id, "", nil
	}

	if _, err := semver.StrictNewVersion(version); err != nil {
		return "", "", fmt.Errorf("invalid module version %q: %w", version, err)
	}

	return id, version, nil
}

func parseProjectFerretVersion(version string) (*semver.Version, error) {
	normalized := strings.TrimPrefix(strings.TrimSpace(version), "v")
	parsed, err := semver.StrictNewVersion(normalized)
	if err != nil {
		return nil, fmt.Errorf("project Ferret version %q is not valid SemVer: %w", version, err)
	}

	return parsed, nil
}

func releaseSupportsFerret(constraint string, version *semver.Version) (bool, error) {
	constraint = strings.TrimSpace(constraint)
	if constraint == "" {
		return false, fmt.Errorf("release does not declare a Ferret version constraint")
	}

	parsed, err := semver.NewConstraint(constraint)
	if err != nil {
		return false, fmt.Errorf("invalid Ferret version constraint %q: %w", constraint, err)
	}

	return parsed.Check(version), nil
}
