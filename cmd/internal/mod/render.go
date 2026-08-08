package mod

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	barnpublish "github.com/MontFerret/barn/pkg/publish"

	"github.com/MontFerret/cli/v2/pkg/module/discovery"
	"github.com/MontFerret/cli/v2/pkg/module/install"
	"github.com/MontFerret/cli/v2/pkg/module/scaffold"
)

func renderModuleSearch(output io.Writer, results []discovery.SearchResult) error {
	if len(results) == 0 {
		_, err := fmt.Fprintln(output, "No modules found.")
		return err
	}

	table := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(table, "NAME\tVERSION\tDESCRIPTION"); err != nil {
		return err
	}

	for _, result := range results {
		if _, err := fmt.Fprintf(table, "%s\t%s\t%s\n", result.Name, result.Version, result.Description); err != nil {
			return err
		}
	}

	return table.Flush()
}

func renderModuleInfo(output io.Writer, info *discovery.ModuleInfo) {
	fmt.Fprintf(output, "Name: %s\n", info.Name)
	fmt.Fprintf(output, "Description: %s\n", info.Description)

	if info.Latest == "" {
		fmt.Fprintln(output, "Latest stable: (none)")
	} else {
		fmt.Fprintf(output, "Latest stable: %s\n", info.Latest)
	}

	fmt.Fprintf(output, "Newest available: %s\n", info.Newest)
	fmt.Fprintf(output, "Selected version: %s\n", info.SelectedVersion)
	fmt.Fprintf(output, "Versions: %s\n", strings.Join(info.Versions, ", "))
	fmt.Fprintf(output, "Namespace: %s\n", info.Namespace)

	if info.Ferret != "" {
		fmt.Fprintf(output, "Ferret: %s\n", info.Ferret)
	}

	fmt.Fprintf(output, "Repository: %s\n", info.Repository)

	if info.SourcePath != "" {
		fmt.Fprintf(output, "Source path: %s\n", info.SourcePath)
	}

	fmt.Fprintf(output, "Commit: %s\n", info.Commit)

	if info.Documentation != "" {
		fmt.Fprintf(output, "Documentation: %s\n", info.Documentation)
	}
}

func renderModuleInstall(output io.Writer, result *install.Result) {
	fmt.Fprintf(output, "Resolved %s@%s\n", result.ID, result.Version)
	fmt.Fprintf(output, "Compatible with project Ferret %s (%s)\n", result.ProjectFerret, result.FerretConstraint)
	fmt.Fprintf(output, "Package: %s\n", result.PackagePath)

	if !result.Changed {
		fmt.Fprintf(output, "%s@%s is already installed\n", result.ID, result.Version)
		return
	}

	if result.DependenciesChanged {
		fmt.Fprintln(output, "Updated Go module dependencies")
	}

	if result.FerretDependencyAdded {
		fmt.Fprintf(output, "Added github.com/MontFerret/ferret/v2 %s\n", result.ProjectFerret)
	}

	if result.CompositionScaffolded {
		fmt.Fprintf(output, "Created Ferret composition helper in %s\n", result.EditedFile)
	} else if result.SourceChanged {
		fmt.Fprintf(output, "Registered module in %s\n", result.EditedFile)
	} else {
		fmt.Fprintf(output, "Module registration already present in %s\n", result.EditedFile)
	}

	fmt.Fprintln(output, "Validated owning package build")
}

func renderModuleInit(output io.Writer, result *scaffold.Result) {
	fmt.Fprintf(output, "Created Ferret module in %s\n", result.Directory)
	fmt.Fprintf(output, "Runtime namespace: %s\n", result.Namespace)
	fmt.Fprintln(output, "Next steps:")
	fmt.Fprintln(output, "  1. Replace the TODO metadata in ferret.yaml and README.md.")
	fmt.Fprintln(output, "  2. Implement and test the module.")
	fmt.Fprintln(output, "  3. Run go mod tidy when you are ready to resolve dependencies.")
}

func renderModulePublication(output io.Writer, publication *barnpublish.Result) {
	fmt.Fprintln(output, "Manifest: valid")
	fmt.Fprintf(output, "Repository: %s\n", publication.Module.Source.Repository)
	if publication.Module.Source.Path != "" {
		fmt.Fprintf(output, "Source path: %s\n", publication.Module.Source.Path)
	}
	fmt.Fprintf(output, "Version: %s\n", publication.Version.Version)
	fmt.Fprintf(output, "Tag: %s\n", publication.Version.Tag)
	fmt.Fprintf(output, "Commit: %s\n\n", publication.Version.Commit)

	for _, file := range publication.Files {
		fmt.Fprintf(output, "%s\n%s\n", file.Path, file.Content)
	}

	switch publication.Kind {
	case barnpublish.NewModule:
		fmt.Fprintln(output, "Add both records to a Barn pull request.")
	case barnpublish.NewVersion:
		fmt.Fprintln(output, "Add the new version record to a Barn pull request without modifying published records.")
	}
}
