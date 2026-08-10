package publish

import (
	"context"

	barnpublish "github.com/MontFerret/barn/pkg/publish"
)

type (
	// Mode selects whether a prepared publication is submitted or only inspected.
	Mode string

	// Status describes the outcome of a publication workflow.
	Status string

	// Options controls module publication.
	Options struct {
		Directory string
		Tag       string
		Mode      Mode
	}

	// Result contains the prepared records and any resulting Registry pull request.
	Result struct {
		Status         Status
		Module         string
		Version        string
		Tag            string
		Prepared       *barnpublish.Result
		PullRequestURL string
	}

	// Submission is the provider-independent result of submitting prepared records.
	Submission struct {
		URL              string
		Existing         bool
		AlreadyPublished bool
	}

	// Submitter sends prepared Barn records to the Registry governance workflow.
	Submitter interface {
		Submit(context.Context, *barnpublish.Result) (*Submission, error)
	}
)

const (
	// ModeSubmit prepares and submits the release to the Registry.
	ModeSubmit Mode = "submit"
	// ModeDryRun prepares the release without provider authentication or mutation.
	ModeDryRun Mode = "dry-run"
	// ModePrint prepares records for deterministic machine-readable output.
	ModePrint Mode = "print"
)

const (
	// StatusReady means validated records are ready but were not submitted.
	StatusReady Status = "ready"
	// StatusSubmitted means a new Registry pull request was created.
	StatusSubmitted Status = "submitted"
	// StatusExistingSubmission means an exact open Registry pull request was found.
	StatusExistingSubmission Status = "existing-submission"
	// StatusAlreadyPublished means the release records already exist in the Registry.
	StatusAlreadyPublished Status = "already-published"
)
