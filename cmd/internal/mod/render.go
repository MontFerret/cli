package mod

import (
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"text/tabwriter"

	barnpublish "github.com/MontFerret/barn/pkg/publish"

	"github.com/MontFerret/cli/v2/pkg/module/discovery"
	"github.com/MontFerret/cli/v2/pkg/module/install"
	modulepublish "github.com/MontFerret/cli/v2/pkg/module/publish"
	"github.com/MontFerret/cli/v2/pkg/module/scaffold"
)

type (
	publicationDocument struct {
		SchemaVersion int                  `json:"schemaVersion"`
		Status        modulepublish.Status `json:"status"`
		Module        string               `json:"module"`
		Version       string               `json:"version"`
		Tag           string               `json:"tag"`
		Kind          string               `json:"kind,omitempty"`
		Commit        string               `json:"commit,omitempty"`
		Records       []publicationRecord  `json:"records"`
	}

	publicationRecord struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
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

func renderModulePublication(output io.Writer, publication *modulepublish.Result, mode modulepublish.Mode) {
	if publication.Status == modulepublish.StatusAlreadyPublished {
		fmt.Fprintln(output, "✓ Validated ferret.yaml")
		fmt.Fprintf(output, "%s@%s is already published.\n", publication.Module, publication.Version)

		return
	}

	if publication.Prepared == nil {
		return
	}

	commit := publication.Prepared.Version.Commit
	shortCommit := commit
	if len(shortCommit) > 7 {
		shortCommit = shortCommit[:7]
	}

	fmt.Fprintln(output, "✓ Validated ferret.yaml")
	fmt.Fprintf(output, "✓ Resolved %s → %s\n", publication.Tag, shortCommit)
	fmt.Fprintln(output, "✓ Verified public source")
	fmt.Fprintln(output, "✓ Verified README.md and go.mod")
	fmt.Fprintf(output, "✓ Prepared %s@%s\n", publication.Module, publication.Version)

	switch publication.Status {
	case modulepublish.StatusReady:
		if mode == modulepublish.ModeDryRun {
			fmt.Fprintln(output, "Ready to publish.")
		}
	case modulepublish.StatusSubmitted:
		fmt.Fprintln(output, "✓ Submitted to Ferret Registry")
		fmt.Fprintln(output, publication.PullRequestURL)
	case modulepublish.StatusExistingSubmission:
		fmt.Fprintln(output, "✓ Found existing Registry submission")
		fmt.Fprintln(output, publication.PullRequestURL)
	}
}

func renderModulePublicationJSON(output io.Writer, publication *modulepublish.Result) error {
	document := publicationDocument{
		SchemaVersion: 1,
		Status:        publication.Status,
		Module:        publication.Module,
		Version:       publication.Version,
		Tag:           publication.Tag,
		Records:       []publicationRecord{},
	}

	if publication.Prepared != nil {
		document.Kind = string(publication.Prepared.Kind)
		document.Commit = publication.Prepared.Version.Commit
		files := append([]barnpublish.File{}, publication.Prepared.Files...)
		sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
		document.Records = make([]publicationRecord, len(files))

		for index, file := range files {
			document.Records[index] = publicationRecord{Path: file.Path, Content: string(file.Content)}
		}
	}

	encoder := json.NewEncoder(output)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")

	return encoder.Encode(document)
}
