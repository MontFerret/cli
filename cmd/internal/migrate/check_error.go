package migrate

import (
	"fmt"

	"github.com/MontFerret/cli/v2/internal/migration"
)

// compatibilityCheckError preserves the exact terminal sentence that main prints verbatim.
type compatibilityCheckError string

func (err compatibilityCheckError) Error() string {
	return string(err)
}

func newCompatibilityCheckError(result *migration.CompatibilityResult) error {
	issueCount := 0
	issueFiles := make(map[string]struct{})
	failureFiles := make(map[string]struct{})

	for _, diagnostic := range result.Diagnostics {
		switch diagnostic.Kind {
		case migration.CompatibilityDiagnosticIssue:
			issueCount++
			issueFiles[diagnostic.Path] = struct{}{}
		case migration.CompatibilityDiagnosticFailure:
			failureFiles[diagnostic.Path] = struct{}{}
		}
	}

	if issueCount == 0 && len(failureFiles) == 0 {
		return nil
	}

	if issueCount == 0 {
		return compatibilityCheckError(fmt.Sprintf(
			"Could not check %d of %d FQL %s for v1 compatibility.",
			len(failureFiles),
			result.ScannedFiles,
			migrationNoun(result.ScannedFiles, "file", "files"),
		))
	}

	issueSummary := fmt.Sprintf(
		"Found %d v1 compatibility %s in %d of %d FQL %s",
		issueCount,
		migrationNoun(issueCount, "issue", "issues"),
		len(issueFiles),
		result.ScannedFiles,
		migrationNoun(result.ScannedFiles, "file", "files"),
	)
	if len(failureFiles) == 0 {
		return compatibilityCheckError(issueSummary + ".")
	}

	return compatibilityCheckError(fmt.Sprintf(
		"%s; could not check %d FQL %s.",
		issueSummary,
		len(failureFiles),
		migrationNoun(len(failureFiles), "file", "files"),
	))
}
