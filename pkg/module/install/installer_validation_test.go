package install

import (
	"context"
	"strings"
	"testing"
)

func TestInstallerRejectsGoOriginCommitMismatch(t *testing.T) {
	const (
		modFile     = "/tmp/project.mod"
		packagePath = "example.com/ferret/archive"
		version     = "1.0.0"
	)
	runner := &scriptedGoRunner{responses: map[string]scriptedGoResponse{
		"list -mod=mod -modfile=" + modFile + " -m -json " + packagePath: {
			output: `{"Path":"example.com/ferret/archive","Version":"v1.0.0"}`,
		},
		"list -mod=mod -modfile=" + modFile + " -m -json " + ferretCoreModulePath: {
			output: `{"Path":"github.com/MontFerret/ferret/v2","Version":"v2.0.0-alpha.44"}`,
		},
		"mod download -json -modfile=" + modFile + " " + packagePath + "@v" + version: {
			output: `{"Path":"example.com/ferret/archive","Version":"v1.0.0","Origin":{"Hash":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}}`,
		},
	}}
	release := installTestVersion("acme/archive", version, ">=2.0.0-alpha.43 <3.0.0", packagePath)
	release.Source.Commit = "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

	err := New(nil, runner).validateResolvedModule(context.Background(), &projectInfo{
		Root: "/tmp", FerretVersion: "v2.0.0-alpha.44",
	}, modFile, release)
	if err == nil || !strings.Contains(err.Error(), "does not match Go module origin") {
		t.Fatalf("unexpected error: %v", err)
	}
}
