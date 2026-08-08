package install

import "fmt"

// MissingCompositionError describes a safe composition scaffold that requires
// user approval before the installer may add it to the project.
type MissingCompositionError struct {
	File    string
	Package string
}

func (err *MissingCompositionError) Error() string {
	return fmt.Sprintf(
		"project has no active ferret.New(...) composition; create %s with NewFerret in package %s",
		err.File,
		err.Package,
	)
}
