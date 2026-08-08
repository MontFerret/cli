package module

import (
	"context"
	"strings"
	"testing"

	"github.com/Masterminds/semver/v3"

	barnregistry "github.com/MontFerret/barn/pkg/registry"
)

func TestInstallerSelectsNewestCompatibleRelease(t *testing.T) {
	registry := &fakeRegistry{
		modules: map[string]*barnregistry.Module{
			"acme/archive": {
				ID: "acme/archive",
				Versions: []barnregistry.VersionSummary{
					{Version: "2.0.0-rc.1"},
					{Version: "1.1.0-rc.2"},
					{Version: "1.0.0"},
				},
			},
		},
		versions: map[string]*barnregistry.Version{
			"acme/archive@2.0.0-rc.1": installTestVersion("acme/archive", "2.0.0-rc.1", ">=2.2.0 <3.0.0", "example.com/archive/v2"),
			"acme/archive@1.1.0-rc.2": installTestVersion("acme/archive", "1.1.0-rc.2", ">=2.0.0-alpha.43 <3.0.0", "example.com/archive"),
			"acme/archive@1.0.0":      installTestVersion("acme/archive", "1.0.0", ">=2.0.0-alpha.1 <3.0.0", "example.com/archive"),
		},
	}
	installer := NewInstaller(registry, nil)
	projectVersion := semver.MustParse("2.0.0-alpha.44")

	release, err := installer.resolveRelease(context.Background(), "acme/archive", "", projectVersion)
	if err != nil {
		t.Fatal(err)
	}
	if release.Version.Version != "1.1.0-rc.2" {
		t.Fatalf("selected %s, want 1.1.0-rc.2", release.Version.Version)
	}
	if len(release.HistoricalPackages) != 2 {
		t.Fatalf("unexpected historical packages: %#v", release.HistoricalPackages)
	}
}

func TestInstallerRejectsExplicitIncompatibleRelease(t *testing.T) {
	registry := &fakeRegistry{
		modules: map[string]*barnregistry.Module{
			"acme/archive": {ID: "acme/archive", Versions: []barnregistry.VersionSummary{{Version: "1.0.0"}}},
		},
		versions: map[string]*barnregistry.Version{
			"acme/archive@1.0.0": installTestVersion("acme/archive", "1.0.0", ">=2.1.0 <3.0.0", "example.com/archive"),
		},
	}

	_, err := NewInstaller(registry, nil).resolveRelease(context.Background(), "acme/archive", "1.0.0", semver.MustParse("2.0.0"))
	if err == nil || !strings.Contains(err.Error(), "requires Ferret >=2.1.0 <3.0.0") || !strings.Contains(err.Error(), "project selects Ferret 2.0.0") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInstallerRejectsMissingOrMalformedCompatibility(t *testing.T) {
	for _, constraint := range []string{"", "definitely not semver"} {
		registry := &fakeRegistry{
			modules: map[string]*barnregistry.Module{
				"acme/archive": {ID: "acme/archive", Versions: []barnregistry.VersionSummary{{Version: "1.0.0"}}},
			},
			versions: map[string]*barnregistry.Version{
				"acme/archive@1.0.0": installTestVersion("acme/archive", "1.0.0", constraint, "example.com/archive"),
			},
		}

		_, err := NewInstaller(registry, nil).resolveRelease(context.Background(), "acme/archive", "1.0.0", semver.MustParse("2.0.0"))
		if err == nil || !strings.Contains(err.Error(), "unusable compatibility metadata") {
			t.Fatalf("constraint %q produced unexpected error: %v", constraint, err)
		}
	}
}

func installTestVersion(id, version, constraint, packagePath string) *barnregistry.Version {
	return &barnregistry.Version{
		ID: id, Version: version, Ferret: constraint,
		Package: barnregistry.Package{Path: packagePath},
		Source:  barnregistry.Source{Commit: "0123456789012345678901234567890123456789"},
	}
}
