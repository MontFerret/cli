package scaffold

type (
	// Options controls module project scaffolding.
	Options struct {
		Name      string
		GoModule  string
		Directory string
		Namespace string
	}

	// Result describes a completed scaffold.
	Result struct {
		Directory string
		Namespace string
	}

	// Environment pins toolchain and Ferret versions in generated projects.
	Environment struct {
		GoVersion     string
		FerretVersion string
	}

	// EnvironmentProvider resolves scaffold dependency versions lazily.
	EnvironmentProvider func() (Environment, error)
)
