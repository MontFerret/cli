package github

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	barnpublish "github.com/MontFerret/barn/pkg/publish"

	modulepublish "github.com/MontFerret/cli/v2/pkg/module/publish"
)

const (
	upstreamOwner = "MontFerret"
	upstreamName  = "barn"
	forkTimeout   = 30 * time.Second
)

// Submitter creates focused Ferret Registry pull requests through GitHub.
type Submitter struct {
	apiURL       string
	httpClient   *http.Client
	pollInterval time.Duration
	token        TokenProvider
}

// New constructs a GitHub Registry submitter.
func New(setters ...Option) (*Submitter, error) {
	options := defaultSubmitterOptions()

	for _, setter := range setters {
		if setter == nil {
			continue
		}

		if err := setter(&options); err != nil {
			return nil, err
		}
	}

	return &Submitter{
		apiURL: options.apiURL, httpClient: options.httpClient,
		pollInterval: options.pollInterval, token: options.token,
	}, nil
}

// Submit creates or recovers the pull request for a prepared Barn publication.
func (s *Submitter) Submit(ctx context.Context, publication *barnpublish.Result) (*modulepublish.Submission, error) {
	records, versionPath, err := publicationRecords(publication)
	if err != nil {
		return nil, stageError(StageRecords, err)
	}

	token, err := s.token.Token(ctx)
	if err != nil {
		return nil, stageError(StageAuthentication, newAuthenticationError(err))
	}

	client := newClient(s.apiURL, s.httpClient, token)
	currentUser, err := client.currentUser(ctx)
	if err != nil {
		return nil, stageError(
			StageAuthentication,
			newAuthenticationError(fmt.Errorf("validate GitHub credential: %w", err)),
		)
	}

	if currentUser.Login == "" {
		return nil, stageError(
			StageAuthentication,
			newAuthenticationError(errors.New("GitHub returned no authenticated user login")),
		)
	}

	upstream, err := client.repository(ctx, upstreamOwner, upstreamName)
	if err != nil {
		return nil, stageError(StageRepository, fmt.Errorf("load %s/%s: %w", upstreamOwner, upstreamName, err))
	}

	if upstream.DefaultBranch == "" {
		return nil, stageError(StageRepository, errors.New("canonical repository has no default branch"))
	}

	baseReference, err := client.reference(ctx, upstreamOwner, upstreamName, upstream.DefaultBranch)
	if err != nil {
		return nil, stageError(StageRepository, fmt.Errorf("resolve upstream branch %q: %w", upstream.DefaultBranch, err))
	}

	if baseReference.Object.SHA == "" {
		return nil, stageError(StageRepository, fmt.Errorf("upstream branch %q has no commit", upstream.DefaultBranch))
	}

	baseCommit, err := client.gitCommit(ctx, upstreamOwner, upstreamName, baseReference.Object.SHA)
	if err != nil {
		return nil, stageError(StageRepository, fmt.Errorf("load upstream commit %s: %w", baseReference.Object.SHA, err))
	}

	if baseCommit.Tree.SHA == "" {
		return nil, stageError(StageRepository, fmt.Errorf("upstream commit %s has no tree", baseReference.Object.SHA))
	}

	pending, err := s.reconcile(ctx, client, baseReference.Object.SHA, records)
	if err != nil {
		return nil, stageError(StageRecords, err)
	}

	if len(pending) == 0 {
		return &modulepublish.Submission{AlreadyPublished: true}, nil
	}

	existing, err := s.findPullRequest(ctx, client, upstream.DefaultBranch, pending, versionPath)
	if err != nil {
		return nil, stageError(StagePullRequest, err)
	}

	if existing != nil {
		return &modulepublish.Submission{URL: existing.HTMLURL, Existing: true}, nil
	}

	fork, err := s.fork(ctx, client, currentUser.Login)
	if err != nil {
		return nil, stageError(StageFork, err)
	}

	forkOwner, forkName, valid := splitRepository(fork.FullName)
	if !valid {
		return nil, stageError(StageFork, fmt.Errorf("GitHub returned invalid fork identity %q", fork.FullName))
	}

	branch := fmt.Sprintf(
		"ferret-publish/%s-%s-v%s-%s",
		publication.Module.Owner,
		publication.Module.Name,
		publication.Version.Version,
		publication.Version.Commit[:8],
	)

	reused, err := s.existingBranch(ctx, client, forkOwner, forkName, branch, baseReference.Object.SHA, pending)
	if err != nil {
		return nil, stageError(StageBranch, err)
	}

	title := fmt.Sprintf("Publish %s/%s@%s", publication.Module.Owner, publication.Module.Name, publication.Version.Version)
	if !reused {
		tree, err := client.createTree(ctx, forkOwner, forkName, baseCommit.Tree.SHA, pending)
		if err != nil {
			return nil, stageError(StageTree, err)
		}

		if tree.SHA == "" {
			return nil, stageError(StageTree, errors.New("GitHub returned no tree identity"))
		}

		commit, err := client.createCommit(ctx, forkOwner, forkName, title, tree.SHA, baseReference.Object.SHA)
		if err != nil {
			return nil, stageError(StageCommit, err)
		}

		if commit.SHA == "" {
			return nil, stageError(StageCommit, errors.New("GitHub returned no commit identity"))
		}

		if err := client.createReference(ctx, forkOwner, forkName, branch, commit.SHA); err != nil {
			if !hasStatus(err, http.StatusUnprocessableEntity) {
				return nil, stageError(StageBranch, err)
			}

			reused, checkErr := s.existingBranch(ctx, client, forkOwner, forkName, branch, baseReference.Object.SHA, pending)
			if checkErr != nil || !reused {
				if checkErr != nil {
					err = errors.Join(err, checkErr)
				}

				return nil, stageError(StageBranch, err)
			}
		}
	}

	body := fmt.Sprintf(
		"Publish `%s/%s@%s`.\n\n- Source: %s\n- Tag: `%s`\n- Commit: `%s`\n\nGenerated by `ferret mod publish`.\n",
		publication.Module.Owner,
		publication.Module.Name,
		publication.Version.Version,
		publication.Module.Source.Repository,
		publication.Version.Tag,
		publication.Version.Commit,
	)
	pull, err := client.createPullRequest(
		ctx,
		upstreamOwner,
		upstreamName,
		title,
		body,
		currentUser.Login+":"+branch,
		upstream.DefaultBranch,
	)
	if err != nil {
		if hasStatus(err, http.StatusUnprocessableEntity) {
			existing, findErr := s.findPullRequest(ctx, client, upstream.DefaultBranch, pending, versionPath)
			if findErr == nil && existing != nil {
				return &modulepublish.Submission{URL: existing.HTMLURL, Existing: true}, nil
			}

			if findErr != nil {
				err = errors.Join(err, findErr)
			}
		}

		return nil, stageError(StagePullRequest, err)
	}

	if pull.HTMLURL == "" {
		return nil, stageError(StagePullRequest, errors.New("GitHub returned no pull request URL"))
	}

	return &modulepublish.Submission{URL: pull.HTMLURL}, nil
}

func (s *Submitter) reconcile(ctx context.Context, client *client, base string, records []record) ([]record, error) {
	pending := make([]record, 0, len(records))

	for _, item := range records {
		content, err := client.content(ctx, upstreamOwner, upstreamName, item.Path, base)
		if err != nil {
			if hasStatus(err, http.StatusNotFound) {
				pending = append(pending, item)

				continue
			}

			return nil, fmt.Errorf("read upstream record %s: %w", item.Path, err)
		}

		if !bytes.Equal(content, item.Content) {
			return nil, fmt.Errorf("%w: %s", ErrImmutableConflict, item.Path)
		}
	}

	return pending, nil
}

func (s *Submitter) findPullRequest(ctx context.Context, client *client, base string, records []record, versionPath string) (*pullRequest, error) {
	pulls, err := client.pullRequests(ctx, upstreamOwner, upstreamName, base)

	if err != nil {
		return nil, fmt.Errorf("list open Barn pull requests: %w", err)
	}

	for index := range pulls {
		pull := &pulls[index]
		files, err := client.pullFiles(ctx, upstreamOwner, upstreamName, pull.Number)
		if err != nil {
			return nil, fmt.Errorf("list files for %s: %w", pull.HTMLURL, err)
		}

		containsVersion := false
		for _, file := range files {
			if file.Filename == versionPath {
				containsVersion = true

				break
			}
		}

		if !containsVersion {
			continue
		}

		if !sameRecordSet(files, records) || pull.Head.Repo == nil {
			return nil, fmt.Errorf("%w: %s", ErrPullConflict, pull.HTMLURL)
		}

		headOwner, headName, valid := splitRepository(pull.Head.Repo.FullName)
		if !valid {
			return nil, fmt.Errorf("pull request %s has invalid head repository %q", pull.HTMLURL, pull.Head.Repo.FullName)
		}

		for _, item := range records {
			content, err := client.content(ctx, headOwner, headName, item.Path, pull.Head.SHA)
			if err != nil {
				return nil, fmt.Errorf("read pull request record %s from %s: %w", item.Path, pull.HTMLURL, err)
			}

			if !bytes.Equal(content, item.Content) {
				return nil, fmt.Errorf("%w: %s", ErrPullConflict, pull.HTMLURL)
			}
		}

		return pull, nil
	}

	return nil, nil
}

func (s *Submitter) fork(ctx context.Context, client *client, login string) (*repository, error) {
	forks, err := client.forks(ctx, upstreamOwner, upstreamName)
	if err != nil {
		return nil, fmt.Errorf("list Barn forks: %w", err)
	}

	for index := range forks {
		if strings.EqualFold(forks[index].Owner.Login, login) {
			return s.waitForFork(ctx, client, forks[index].FullName)
		}
	}

	created, err := client.createFork(ctx, upstreamOwner, upstreamName)
	if err != nil {
		return nil, fmt.Errorf("create personal Barn fork: %w", err)
	}

	return s.waitForFork(ctx, client, created.FullName)
}

func (s *Submitter) waitForFork(ctx context.Context, client *client, fullName string) (*repository, error) {
	owner, name, valid := splitRepository(fullName)
	if !valid {
		return nil, fmt.Errorf("GitHub returned invalid fork identity %q", fullName)
	}

	waitContext, cancel := context.WithTimeout(ctx, forkTimeout)
	defer cancel()

	for {
		fork, err := client.repository(waitContext, owner, name)
		if err == nil {
			if !fork.Permissions.Push {
				return nil, fmt.Errorf("authenticated user cannot write to personal fork %s", fullName)
			}

			return fork, nil
		}

		if !hasStatus(err, http.StatusNotFound) {
			return nil, fmt.Errorf("load personal fork %s: %w", fullName, err)
		}

		timer := time.NewTimer(s.pollInterval)
		select {
		case <-waitContext.Done():
			timer.Stop()

			return nil, fmt.Errorf("wait for personal fork %s: %w", fullName, waitContext.Err())
		case <-timer.C:
		}
	}
}

func (s *Submitter) existingBranch(ctx context.Context, client *client, owner, name, branch, base string, records []record) (bool, error) {
	reference, err := client.reference(ctx, owner, name, branch)
	if err != nil {
		if hasStatus(err, http.StatusNotFound) {
			return false, nil
		}

		return false, fmt.Errorf("read publication branch %q: %w", branch, err)
	}

	details, err := client.commitDetails(ctx, owner, name, reference.Object.SHA)
	if err != nil {
		return false, fmt.Errorf("inspect publication branch %q: %w", branch, err)
	}

	if len(details.Parents) != 1 || details.Parents[0].SHA != base || !sameRecordSet(details.Files, records) {
		return false, fmt.Errorf("%w: %s; delete the branch from %s/%s and retry", ErrBranchConflict, branch, owner, name)
	}

	for _, item := range records {
		content, err := client.content(ctx, owner, name, item.Path, reference.Object.SHA)
		if err != nil {
			return false, fmt.Errorf("read publication branch record %s from %s/%s: %w", item.Path, owner, name, err)
		}

		if !bytes.Equal(content, item.Content) {
			return false, fmt.Errorf("%w: %s; delete the branch from %s/%s and retry", ErrBranchConflict, branch, owner, name)
		}
	}

	return true, nil
}
