package registryclient

import "net/http"

const DefaultBaseURL = "https://registry.ferretlang.org/"

type (
	// HTTPClient executes registry requests.
	HTTPClient interface {
		Do(*http.Request) (*http.Response, error)
	}

	// RootIndex identifies the public artifact indexes exposed by a registry.
	RootIndex struct {
		SchemaVersion int
		ModulesHref   string
	}

	// ModuleReference identifies one module document in the catalog.
	ModuleReference struct {
		ID   string
		Href string
	}

	// Catalog is the registry's module catalog.
	Catalog struct {
		Modules []ModuleReference
	}

	// ModuleVersionReference identifies one published version document.
	ModuleVersionReference struct {
		Version string
		Href    string
	}

	// Module describes registry metadata shared by all versions.
	Module struct {
		ID          string
		Owner       string
		Name        string
		Description string
		License     string
		Latest      string
		Versions    []ModuleVersionReference
	}

	// Source identifies the immutable source of one module version.
	Source struct {
		Repository string
		Path       string
		Commit     string
	}

	// Version describes one published module version.
	Version struct {
		ID            string
		Version       string
		Namespace     string
		Ferret        string
		Source        Source
		Documentation string
	}

	rootDocument struct {
		SchemaVersion int               `json:"schemaVersion"`
		Artifacts     map[string]string `json:"artifacts"`
	}

	catalogDocument struct {
		SchemaVersion int                       `json:"schemaVersion"`
		Modules       []moduleReferenceDocument `json:"modules"`
	}

	moduleReferenceDocument struct {
		ID   string `json:"id"`
		Href string `json:"href"`
	}

	moduleDocument struct {
		SchemaVersion int                        `json:"schemaVersion"`
		ID            string                     `json:"id"`
		Owner         string                     `json:"owner"`
		Name          string                     `json:"name"`
		Description   string                     `json:"description"`
		License       string                     `json:"license,omitempty"`
		Latest        string                     `json:"latest,omitempty"`
		Versions      []moduleVersionRefDocument `json:"versions"`
	}

	moduleVersionRefDocument struct {
		Version string `json:"version"`
		Href    string `json:"href"`
	}

	versionDocument struct {
		SchemaVersion int               `json:"schemaVersion"`
		ID            string            `json:"id"`
		Version       string            `json:"version"`
		Namespace     string            `json:"namespace"`
		Ferret        string            `json:"ferret,omitempty"`
		Source        sourceDocument    `json:"source"`
		Content       map[string]string `json:"content"`
	}

	sourceDocument struct {
		Repository string `json:"repository"`
		Path       string `json:"path,omitempty"`
		Commit     string `json:"commit"`
	}
)
