package cmd

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

	modulelifecycle "github.com/MontFerret/cli/v2/pkg/module"
)

func renderModuleSearch(output io.Writer, results []modulelifecycle.SearchResult) error {
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

func renderModuleInfo(output io.Writer, info *modulelifecycle.ModuleInfo) {
	fmt.Fprintf(output, "Name: %s\n", info.Name)
	fmt.Fprintf(output, "Description: %s\n", info.Description)
	if info.License != "" {
		fmt.Fprintf(output, "License: %s\n", info.License)
	}
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

func renderModuleCreate(output io.Writer, result *modulelifecycle.CreateResult) {
	fmt.Fprintf(output, "Created Ferret module in %s\n", result.Directory)
	fmt.Fprintf(output, "Runtime namespace: %s\n", result.Namespace)
	fmt.Fprintln(output, "Next steps:")
	fmt.Fprintln(output, "  1. Replace the TODO metadata in ferret.yaml and README.md.")
	fmt.Fprintln(output, "  2. Implement and test the module.")
	fmt.Fprintln(output, "  3. Run go mod tidy when you are ready to resolve dependencies.")
}

func renderModulePublication(output io.Writer, publication *modulelifecycle.Publication) {
	fmt.Fprintln(output, "Manifest: valid")
	fmt.Fprintf(output, "Repository: %s\n", publication.Repository)
	if publication.SourcePath != "" {
		fmt.Fprintf(output, "Source path: %s\n", publication.SourcePath)
	}
	fmt.Fprintf(output, "Version: %s\n", publication.Version)
	fmt.Fprintf(output, "Tag: %s\n", publication.Tag)
	fmt.Fprintf(output, "Commit: %s\n\n", publication.Commit)
	fmt.Fprintf(output, "%s\n%s\n", publication.ModuleManifestPath, publication.ModuleManifestJSON)
	fmt.Fprintf(output, "%s\n%s\n", publication.VersionRecordPath, publication.VersionRecordJSON)
	fmt.Fprintln(output, "For a first registration, add both records to a Barn pull request.")
	fmt.Fprintln(output, "For an existing module, add only the new version record and do not modify published records.")
}
