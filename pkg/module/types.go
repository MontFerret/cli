package module

import "context"

type (
	// Registry provides the public registry operations used by the module service.
	Registry interface {
		Catalog(context.Context) (*RegistryCatalog, error)
		Module(context.Context, string) (*RegistryModule, error)
		Version(context.Context, string) (*RegistryVersion, error)
	}

	// SearchResult is one row in module discovery output.
	SearchResult struct {
		Name        string
		Version     string
		Description string
	}

	// ModuleInfo is the detailed registry view of a module.
	ModuleInfo struct {
		Name            string
		Description     string
		License         string
		Latest          string
		Newest          string
		SelectedVersion string
		Versions        []string
		Namespace       string
		Ferret          string
		Repository      string
		SourcePath      string
		Commit          string
		Documentation   string
	}

	// CreateOptions controls module project scaffolding.
	CreateOptions struct {
		Name      string
		GoModule  string
		Directory string
		Namespace string
	}

	// CreateResult describes a completed scaffold.
	CreateResult struct {
		Directory string
		Namespace string
	}

	// PublishOptions controls local publication preparation.
	PublishOptions struct {
		Directory string
		Tag       string
	}

	// Publication contains validated Barn source records and their target paths.
	Publication struct {
		Name               string
		Repository         string
		SourcePath         string
		Version            string
		Tag                string
		Commit             string
		ModuleManifestPath string
		ModuleManifestJSON []byte
		VersionRecordPath  string
		VersionRecordJSON  []byte
	}

	// ScaffoldEnvironment pins toolchain and Ferret versions in generated projects.
	ScaffoldEnvironment struct {
		GoVersion     string
		FerretVersion string
	}

	// EnvironmentProvider resolves scaffold dependency versions lazily.
	EnvironmentProvider func() (ScaffoldEnvironment, error)

	// GitRunner runs read-only Git commands in a repository.
	GitRunner interface {
		Run(context.Context, string, ...string) (string, error)
	}
)
