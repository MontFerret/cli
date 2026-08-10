package github

import (
	"errors"
	"fmt"
)

var (
	// ErrImmutableConflict reports an existing Registry record with different bytes.
	ErrImmutableConflict = errors.New("immutable Registry record conflicts with the prepared release")
	// ErrBranchConflict reports a retry branch that is not the exact prepared change.
	ErrBranchConflict = errors.New("publication branch already exists with different content")
	// ErrPullConflict reports an open pull request for the same version with different changes.
	ErrPullConflict = errors.New("an open Registry pull request conflicts with the prepared release")
)

type (
	// Stage identifies the GitHub submission operation that failed.
	Stage string

	// Error adds a stable submission stage while preserving the underlying error.
	Error struct {
		Stage Stage
		Err   error
	}
)

const (
	// StageAuthentication resolves and validates the GitHub credential.
	StageAuthentication Stage = "authentication"
	// StageRepository resolves the canonical Barn base branch and commit.
	StageRepository Stage = "repository"
	// StageRecords reconciles prepared records against canonical Barn state.
	StageRecords Stage = "records"
	// StagePullRequest finds or creates the focused Barn pull request.
	StagePullRequest Stage = "pull request"
	// StageFork finds, creates, and waits for the personal Barn fork.
	StageFork Stage = "fork"
	// StageBranch validates or creates the deterministic publication branch.
	StageBranch Stage = "branch"
	// StageTree creates the exact Git tree containing the pending records.
	StageTree Stage = "tree"
	// StageCommit creates the publication commit from the canonical Barn base.
	StageCommit Stage = "commit"
)

func (e *Error) Error() string {
	return fmt.Sprintf("submit Registry publication at %s stage: %v", e.Stage, e.Err)
}

func (e *Error) Unwrap() error {
	return e.Err
}
