package build

type (
	Target struct {
		OutputPath string
		SourcePath string
	}

	Plan struct {
		OutputDir string
		Targets   []Target
	}
)
