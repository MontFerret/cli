package module

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Masterminds/semver/v3"

	barnregistry "github.com/MontFerret/barn/pkg/registry"
	registryartifact "github.com/MontFerret/specs/pkg/registry/artifact"
)

func TestInstallerResolvesPublishedArtifactThroughBarnClient(t *testing.T) {
	publishedAt := time.Date(2026, time.August, 7, 18, 24, 28, 0, time.UTC)
	commit := "0123456789abcdef0123456789abcdef01234567"
	moduleID := "acme/sqlite"
	packagePath := "example.com/sqlite"
	documents := map[string]any{
		"/index.json": &registryartifact.RootIndex{
			SchemaVersion: registryartifact.SchemaVersion,
			Artifacts: map[string]string{
				registryartifact.ArtifactKeyCategories: "/categories.json",
				registryartifact.ArtifactKeyModules:    "/modules/index.json",
				registryartifact.ArtifactKeyPlugins:    "/plugins/index.json",
			},
		},
		"/modules/index.json": &registryartifact.ModuleIndex{
			SchemaVersion: registryartifact.SchemaVersion,
			Modules: []registryartifact.ModuleIndexEntry{{
				ID:   moduleID,
				Href: "/modules/acme/sqlite/index.json",
			}},
		},
		"/modules/acme/sqlite/index.json": &registryartifact.ModuleDocument{
			SchemaVersion: registryartifact.SchemaVersion,
			ID:            moduleID,
			Owner:         "acme",
			Name:          "sqlite",
			Description:   "SQLite support.",
			Versions: []registryartifact.ModuleDocumentVersion{
				{Version: "1.0.0-rc.2", PublishedAt: publishedAt, Href: "/modules/acme/sqlite/versions/1.0.0-rc.2/index.json"},
				{Version: "1.0.0-rc.1", PublishedAt: publishedAt, Href: "/modules/acme/sqlite/versions/1.0.0-rc.1/index.json"},
			},
		},
		"/modules/acme/sqlite/versions/1.0.0-rc.2/index.json": installRegistryVersionDocument(
			moduleID,
			"1.0.0-rc.2",
			">=2.1.0 <3.0.0",
			packagePath,
			commit,
		),
		"/modules/acme/sqlite/versions/1.0.0-rc.1/index.json": installRegistryVersionDocument(
			moduleID,
			"1.0.0-rc.1",
			">=2.0.0-alpha.43 <3.0.0",
			packagePath,
			commit,
		),
	}

	encoded := make(map[string][]byte, len(documents))
	for path, document := range documents {
		data, err := json.Marshal(document)
		if err != nil {
			t.Fatalf("encode %s: %v", path, err)
		}

		encoded[path] = data
	}

	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		data, exists := encoded[request.URL.Path]
		if !exists {
			http.NotFound(response, request)

			return
		}

		response.Header().Set("Content-Type", "application/json")
		_, _ = response.Write(data)
	}))
	defer server.Close()

	client, err := barnregistry.NewClient(barnregistry.WithBaseURL(server.URL))
	if err != nil {
		t.Fatal(err)
	}

	release, err := NewInstaller(client, nil).resolveRelease(
		context.Background(),
		moduleID,
		"",
		semver.MustParse("2.0.0-alpha.44"),
	)
	if err != nil {
		t.Fatal(err)
	}
	if release.Version.Version != "1.0.0-rc.1" {
		t.Fatalf("selected %s, want 1.0.0-rc.1", release.Version.Version)
	}
	if release.Version.Package.Path != packagePath || release.Version.Source.Commit != commit {
		t.Fatalf("unexpected resolved release: %#v", release.Version)
	}
}

func installRegistryVersionDocument(id, version, constraint, packagePath, commit string) *registryartifact.VersionDocument {
	return &registryartifact.VersionDocument{
		SchemaVersion: registryartifact.SchemaVersion,
		ID:            id,
		Version:       version,
		Description:   "SQLite support.",
		Namespace:     "DB::SQLITE",
		Ferret:        constraint,
		License:       "Apache-2.0",
		Source: registryartifact.VersionSource{
			Repository: "https://example.com/sqlite.git",
			Commit:     commit,
		},
		Package: registryartifact.VersionPackage{Path: packagePath},
		Content: map[string]string{
			registryartifact.ContentKeyDocumentation:     "./docs.md",
			registryartifact.ContentKeyDocumentationHTML: "./docs.html",
		},
	}
}
