package github

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	barnpublish "github.com/MontFerret/barn/pkg/publish"
)

func TestSubmitterCreatesFocusedForkPullRequest(t *testing.T) {
	publication := testPublication(barnpublish.NewModule)
	expectedRecords := make(map[string]string, len(publication.Files))
	for _, file := range publication.Files {
		expectedRecords[file.Path] = string(file.Content)
	}
	mutations := make([]string, 0, 4)
	treePaths := make([]string, 0, 2)
	transport := &scriptedTransport{handle: func(request *http.Request) (*http.Response, error) {
		path := request.URL.Path
		switch {
		case request.Method == http.MethodGet && path == "/user":
			return jsonResponse(http.StatusOK, user{Login: "alice"}), nil
		case request.Method == http.MethodGet && path == "/repos/MontFerret/barn":
			return jsonResponse(http.StatusOK, repository{FullName: "MontFerret/barn", DefaultBranch: "main"}), nil
		case request.Method == http.MethodGet && strings.HasPrefix(path, "/repos/MontFerret/barn/git/ref/heads/main"):
			return jsonResponse(http.StatusOK, gitReference{Object: gitObject{SHA: "base-sha"}}), nil
		case request.Method == http.MethodGet && path == "/repos/MontFerret/barn/git/commits/base-sha":
			return jsonResponse(http.StatusOK, gitCommit{SHA: "base-sha", Tree: gitObject{SHA: "base-tree"}}), nil
		case request.Method == http.MethodGet && strings.Contains(path, "/repos/MontFerret/barn/contents/"):
			return apiFailure(http.StatusNotFound, "Not Found"), nil
		case request.Method == http.MethodGet && path == "/repos/MontFerret/barn/pulls":
			return jsonResponse(http.StatusOK, []pullRequest{}), nil
		case request.Method == http.MethodGet && path == "/repos/MontFerret/barn/forks":
			return jsonResponse(http.StatusOK, []repository{}), nil
		case request.Method == http.MethodPost && path == "/repos/MontFerret/barn/forks":
			mutations = append(mutations, "fork")

			return jsonResponse(http.StatusAccepted, repository{FullName: "alice/renamed-barn", Owner: user{Login: "alice"}}), nil
		case request.Method == http.MethodGet && path == "/repos/alice/renamed-barn":
			return jsonResponse(http.StatusOK, repository{FullName: "alice/renamed-barn", Permissions: permissions{Push: true}}), nil
		case request.Method == http.MethodGet && strings.HasPrefix(path, "/repos/alice/renamed-barn/git/ref/heads/"):
			return apiFailure(http.StatusNotFound, "Not Found"), nil
		case request.Method == http.MethodPost && path == "/repos/alice/renamed-barn/git/trees":
			mutations = append(mutations, "tree")
			var body struct {
				BaseTree string `json:"base_tree"`
				Tree     []struct {
					Path    string `json:"path"`
					Content string `json:"content"`
				} `json:"tree"`
			}
			readJSONBody(t, request, &body)
			if body.BaseTree != "base-tree" || len(body.Tree) != 2 {
				t.Fatalf("unexpected tree request: %#v", body)
			}
			for _, entry := range body.Tree {
				treePaths = append(treePaths, entry.Path)
				if strings.HasPrefix(entry.Path, "dist/") || strings.Contains(entry.Content, "publishedAt") {
					t.Fatalf("unsafe tree entry: %#v", entry)
				}
				if expectedRecords[entry.Path] != entry.Content {
					t.Fatalf("tree entry did not preserve prepared bytes: %#v", entry)
				}
			}

			return jsonResponse(http.StatusCreated, createdTree{SHA: "new-tree"}), nil
		case request.Method == http.MethodPost && path == "/repos/alice/renamed-barn/git/commits":
			mutations = append(mutations, "commit")
			var body struct {
				Message string   `json:"message"`
				Tree    string   `json:"tree"`
				Parents []string `json:"parents"`
			}
			readJSONBody(t, request, &body)
			if body.Message != "Publish acme/widget@1.2.3" || body.Tree != "new-tree" || len(body.Parents) != 1 || body.Parents[0] != "base-sha" {
				t.Fatalf("unexpected commit request: %#v", body)
			}

			return jsonResponse(http.StatusCreated, createdCommit{SHA: "new-commit"}), nil
		case request.Method == http.MethodPost && path == "/repos/alice/renamed-barn/git/refs":
			mutations = append(mutations, "reference")

			return jsonResponse(http.StatusCreated, gitReference{}), nil
		case request.Method == http.MethodPost && path == "/repos/MontFerret/barn/pulls":
			mutations = append(mutations, "pull")
			var body map[string]any
			readJSONBody(t, request, &body)
			pullBody, _ := body["body"].(string)
			if body["title"] != "Publish acme/widget@1.2.3" ||
				body["head"] != "alice:ferret-publish/acme-widget-v1.2.3-abcdef01" ||
				body["base"] != "main" ||
				!strings.Contains(pullBody, "https://github.com/acme/widget") ||
				!strings.Contains(pullBody, "`modules/widget/v1.2.3`") ||
				!strings.Contains(pullBody, "`abcdef0123456789abcdef0123456789abcdef01`") {
				t.Fatalf("unexpected pull request body: %#v", body)
			}

			return jsonResponse(http.StatusCreated, pullRequest{HTMLURL: "https://github.com/MontFerret/barn/pull/123"}), nil
		default:
			t.Fatalf("unexpected GitHub request: %s %s", request.Method, request.URL)

			return nil, nil
		}
	}}
	token := &fakeTokenProvider{token: "secret"}
	submitter, err := New(
		WithHTTPClient(&http.Client{Transport: transport}),
		WithTokenProvider(token),
		WithPollInterval(time.Millisecond),
	)
	if err != nil {
		t.Fatal(err)
	}

	result, err := submitter.Submit(context.Background(), publication)
	if err != nil {
		t.Fatal(err)
	}
	if result.URL != "https://github.com/MontFerret/barn/pull/123" || result.Existing || result.AlreadyPublished {
		t.Fatalf("unexpected submission result: %#v", result)
	}
	if strings.Join(mutations, ",") != "fork,tree,commit,reference,pull" || len(treePaths) != 2 || token.calls != 1 {
		t.Fatalf("unexpected submission effects: mutations=%v paths=%v tokenCalls=%d", mutations, treePaths, token.calls)
	}
}

func TestSubmitterReturnsExistingMatchingPullRequestWithoutMutation(t *testing.T) {
	publication := testPublication(barnpublish.NewVersion)
	versionFile := publication.Files[0]
	mutations := 0
	transport := &scriptedTransport{handle: func(request *http.Request) (*http.Response, error) {
		path := request.URL.Path
		if request.Method != http.MethodGet {
			mutations++
		}
		switch {
		case path == "/user":
			return jsonResponse(http.StatusOK, user{Login: "alice"}), nil
		case path == "/repos/MontFerret/barn":
			return jsonResponse(http.StatusOK, repository{DefaultBranch: "main"}), nil
		case strings.HasPrefix(path, "/repos/MontFerret/barn/git/ref/heads/main"):
			return jsonResponse(http.StatusOK, gitReference{Object: gitObject{SHA: "base"}}), nil
		case path == "/repos/MontFerret/barn/git/commits/base":
			return jsonResponse(http.StatusOK, gitCommit{Tree: gitObject{SHA: "tree"}}), nil
		case strings.Contains(path, "/repos/MontFerret/barn/contents/"):
			return apiFailure(http.StatusNotFound, "Not Found"), nil
		case path == "/repos/MontFerret/barn/pulls":
			return jsonResponse(http.StatusOK, []pullRequest{{
				Number: 7, HTMLURL: "https://github.com/MontFerret/barn/pull/7",
				Head: pullHead{SHA: "head", Repo: &repository{FullName: "bob/barn"}},
			}}), nil
		case path == "/repos/MontFerret/barn/pulls/7/files":
			return jsonResponse(http.StatusOK, []pullFile{{Filename: versionFile.Path, Status: "added"}}), nil
		case strings.Contains(path, "/repos/bob/barn/contents/"):
			return jsonResponse(http.StatusOK, encodedContent(versionFile.Content)), nil
		default:
			t.Fatalf("unexpected GitHub request: %s %s", request.Method, request.URL)

			return nil, nil
		}
	}}
	submitter, err := New(
		WithHTTPClient(&http.Client{Transport: transport}),
		WithTokenProvider(&fakeTokenProvider{token: "secret"}),
	)
	if err != nil {
		t.Fatal(err)
	}

	result, err := submitter.Submit(context.Background(), publication)
	if err != nil {
		t.Fatal(err)
	}
	if result.URL != "https://github.com/MontFerret/barn/pull/7" || !result.Existing || mutations != 0 {
		t.Fatalf("unexpected existing submission: result=%#v mutations=%d", result, mutations)
	}
}

func TestSubmitterTreatsIdenticalUpstreamRecordsAsPublished(t *testing.T) {
	publication := testPublication(barnpublish.NewVersion)
	versionFile := publication.Files[0]
	mutations := 0
	transport := &scriptedTransport{handle: func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodGet {
			mutations++
		}
		switch {
		case request.URL.Path == "/user":
			return jsonResponse(http.StatusOK, user{Login: "alice"}), nil
		case request.URL.Path == "/repos/MontFerret/barn":
			return jsonResponse(http.StatusOK, repository{DefaultBranch: "main"}), nil
		case strings.HasPrefix(request.URL.Path, "/repos/MontFerret/barn/git/ref/heads/main"):
			return jsonResponse(http.StatusOK, gitReference{Object: gitObject{SHA: "base"}}), nil
		case request.URL.Path == "/repos/MontFerret/barn/git/commits/base":
			return jsonResponse(http.StatusOK, gitCommit{Tree: gitObject{SHA: "tree"}}), nil
		case strings.Contains(request.URL.Path, "/repos/MontFerret/barn/contents/"):
			return jsonResponse(http.StatusOK, encodedContent(versionFile.Content)), nil
		default:
			t.Fatalf("unexpected GitHub request: %s %s", request.Method, request.URL)

			return nil, nil
		}
	}}
	submitter, err := New(
		WithHTTPClient(&http.Client{Transport: transport}),
		WithTokenProvider(&fakeTokenProvider{token: "secret"}),
	)
	if err != nil {
		t.Fatal(err)
	}

	result, err := submitter.Submit(context.Background(), publication)
	if err != nil {
		t.Fatal(err)
	}
	if !result.AlreadyPublished || mutations != 0 {
		t.Fatalf("unexpected published result: result=%#v mutations=%d", result, mutations)
	}
}

func TestSubmitterRejectsDivergentRetryBranch(t *testing.T) {
	publication := testPublication(barnpublish.NewVersion)
	transport := &scriptedTransport{handle: func(request *http.Request) (*http.Response, error) {
		path := request.URL.Path
		switch {
		case path == "/user":
			return jsonResponse(http.StatusOK, user{Login: "alice"}), nil
		case path == "/repos/MontFerret/barn":
			return jsonResponse(http.StatusOK, repository{DefaultBranch: "main"}), nil
		case strings.HasPrefix(path, "/repos/MontFerret/barn/git/ref/heads/main"):
			return jsonResponse(http.StatusOK, gitReference{Object: gitObject{SHA: "base"}}), nil
		case path == "/repos/MontFerret/barn/git/commits/base":
			return jsonResponse(http.StatusOK, gitCommit{Tree: gitObject{SHA: "tree"}}), nil
		case strings.Contains(path, "/repos/MontFerret/barn/contents/"):
			return apiFailure(http.StatusNotFound, "Not Found"), nil
		case path == "/repos/MontFerret/barn/pulls":
			return jsonResponse(http.StatusOK, []pullRequest{}), nil
		case request.Method == http.MethodGet && path == "/repos/MontFerret/barn/forks":
			return jsonResponse(http.StatusOK, []repository{{FullName: "alice/barn", Owner: user{Login: "alice"}}}), nil
		case path == "/repos/alice/barn":
			return jsonResponse(http.StatusOK, repository{FullName: "alice/barn", Permissions: permissions{Push: true}}), nil
		case strings.HasPrefix(path, "/repos/alice/barn/git/ref/heads/"):
			return jsonResponse(http.StatusOK, gitReference{Object: gitObject{SHA: "branch"}}), nil
		case path == "/repos/alice/barn/commits/branch":
			return jsonResponse(http.StatusOK, commitDetails{Parents: []gitObject{{SHA: "old-base"}}}), nil
		default:
			t.Fatalf("unexpected GitHub request: %s %s", request.Method, request.URL)

			return nil, nil
		}
	}}
	submitter, err := New(
		WithHTTPClient(&http.Client{Transport: transport}),
		WithTokenProvider(&fakeTokenProvider{token: "secret"}),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = submitter.Submit(context.Background(), publication)
	if !errors.Is(err, ErrBranchConflict) || !strings.Contains(err.Error(), "delete the branch") {
		t.Fatalf("unexpected branch conflict: %v", err)
	}
}

func TestSubmitterReusesExactRetryBranch(t *testing.T) {
	publication := testPublication(barnpublish.NewVersion)
	versionFile := publication.Files[0]
	mutations := make([]string, 0, 1)
	transport := &scriptedTransport{handle: func(request *http.Request) (*http.Response, error) {
		path := request.URL.Path
		switch {
		case path == "/user":
			return jsonResponse(http.StatusOK, user{Login: "alice"}), nil
		case path == "/repos/MontFerret/barn":
			return jsonResponse(http.StatusOK, repository{DefaultBranch: "main"}), nil
		case strings.HasPrefix(path, "/repos/MontFerret/barn/git/ref/heads/main"):
			return jsonResponse(http.StatusOK, gitReference{Object: gitObject{SHA: "base"}}), nil
		case path == "/repos/MontFerret/barn/git/commits/base":
			return jsonResponse(http.StatusOK, gitCommit{Tree: gitObject{SHA: "tree"}}), nil
		case strings.Contains(path, "/repos/MontFerret/barn/contents/"):
			return apiFailure(http.StatusNotFound, "Not Found"), nil
		case path == "/repos/MontFerret/barn/pulls" && request.Method == http.MethodGet:
			return jsonResponse(http.StatusOK, []pullRequest{}), nil
		case path == "/repos/MontFerret/barn/forks":
			return jsonResponse(http.StatusOK, []repository{{FullName: "alice/barn", Owner: user{Login: "alice"}}}), nil
		case path == "/repos/alice/barn":
			return jsonResponse(http.StatusOK, repository{FullName: "alice/barn", Permissions: permissions{Push: true}}), nil
		case strings.HasPrefix(path, "/repos/alice/barn/git/ref/heads/"):
			return jsonResponse(http.StatusOK, gitReference{Object: gitObject{SHA: "branch"}}), nil
		case path == "/repos/alice/barn/commits/branch":
			return jsonResponse(http.StatusOK, commitDetails{
				Parents: []gitObject{{SHA: "base"}},
				Files:   []pullFile{{Filename: versionFile.Path, Status: "added"}},
			}), nil
		case strings.Contains(path, "/repos/alice/barn/contents/"):
			return jsonResponse(http.StatusOK, encodedContent(versionFile.Content)), nil
		case path == "/repos/MontFerret/barn/pulls" && request.Method == http.MethodPost:
			mutations = append(mutations, "pull")

			return jsonResponse(http.StatusCreated, pullRequest{HTMLURL: "https://github.com/MontFerret/barn/pull/9"}), nil
		default:
			t.Fatalf("unexpected GitHub request: %s %s", request.Method, request.URL)

			return nil, nil
		}
	}}
	submitter, err := New(
		WithHTTPClient(&http.Client{Transport: transport}),
		WithTokenProvider(&fakeTokenProvider{token: "secret"}),
	)
	if err != nil {
		t.Fatal(err)
	}

	result, err := submitter.Submit(context.Background(), publication)
	if err != nil {
		t.Fatal(err)
	}
	if result.URL != "https://github.com/MontFerret/barn/pull/9" || strings.Join(mutations, ",") != "pull" {
		t.Fatalf("unexpected exact branch retry: result=%#v mutations=%v", result, mutations)
	}
}

func TestSubmitterAuthenticationFailureMakesNoGitHubRequest(t *testing.T) {
	want := errors.New("not authenticated")
	requests := 0
	transport := &scriptedTransport{handle: func(*http.Request) (*http.Response, error) {
		requests++

		return nil, errors.New("unexpected request")
	}}
	submitter, err := New(
		WithHTTPClient(&http.Client{Transport: transport}),
		WithTokenProvider(&fakeTokenProvider{err: want}),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = submitter.Submit(context.Background(), testPublication(barnpublish.NewVersion))
	var submissionError *Error
	if !errors.Is(err, want) || !errors.As(err, &submissionError) || submissionError.Stage != StageAuthentication || requests != 0 {
		t.Fatalf("unexpected authentication result: err=%v requests=%d", err, requests)
	}
}

func TestSubmitterRejectsConflictingUpstreamRecordBeforeMutation(t *testing.T) {
	mutations := 0
	transport := &scriptedTransport{handle: func(request *http.Request) (*http.Response, error) {
		if request.Method != http.MethodGet {
			mutations++
		}
		switch {
		case request.URL.Path == "/user":
			return jsonResponse(http.StatusOK, user{Login: "alice"}), nil
		case request.URL.Path == "/repos/MontFerret/barn":
			return jsonResponse(http.StatusOK, repository{DefaultBranch: "main"}), nil
		case strings.HasPrefix(request.URL.Path, "/repos/MontFerret/barn/git/ref/heads/main"):
			return jsonResponse(http.StatusOK, gitReference{Object: gitObject{SHA: "base"}}), nil
		case request.URL.Path == "/repos/MontFerret/barn/git/commits/base":
			return jsonResponse(http.StatusOK, gitCommit{Tree: gitObject{SHA: "tree"}}), nil
		case strings.Contains(request.URL.Path, "/repos/MontFerret/barn/contents/"):
			return jsonResponse(http.StatusOK, encodedContent([]byte("different\n"))), nil
		default:
			t.Fatalf("unexpected GitHub request: %s %s", request.Method, request.URL)

			return nil, nil
		}
	}}
	submitter, err := New(
		WithHTTPClient(&http.Client{Transport: transport}),
		WithTokenProvider(&fakeTokenProvider{token: "secret"}),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = submitter.Submit(context.Background(), testPublication(barnpublish.NewVersion))
	if !errors.Is(err, ErrImmutableConflict) || mutations != 0 {
		t.Fatalf("unexpected immutable conflict: err=%v mutations=%d", err, mutations)
	}
}

func TestSubmitterPreservesOperationFailureStages(t *testing.T) {
	for _, test := range []struct {
		failure string
		stage   Stage
	}{
		{failure: "fork", stage: StageFork},
		{failure: "tree", stage: StageTree},
		{failure: "commit", stage: StageCommit},
		{failure: "reference", stage: StageBranch},
		{failure: "pull", stage: StagePullRequest},
	} {
		t.Run(test.failure, func(t *testing.T) {
			submitter, err := New(
				WithHTTPClient(&http.Client{Transport: submissionFailureTransport(t, test.failure)}),
				WithTokenProvider(&fakeTokenProvider{token: "secret"}),
			)
			if err != nil {
				t.Fatal(err)
			}

			_, err = submitter.Submit(context.Background(), testPublication(barnpublish.NewVersion))
			var submissionError *Error
			if !errors.As(err, &submissionError) || submissionError.Stage != test.stage {
				t.Fatalf("unexpected failure stage: %v", err)
			}
		})
	}
}

func TestSubmitterStopsWaitingForForkWhenContextIsCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	transport := &scriptedTransport{handle: func(request *http.Request) (*http.Response, error) {
		path := request.URL.Path
		switch {
		case path == "/user":
			return jsonResponse(http.StatusOK, user{Login: "alice"}), nil
		case path == "/repos/MontFerret/barn":
			return jsonResponse(http.StatusOK, repository{DefaultBranch: "main"}), nil
		case strings.HasPrefix(path, "/repos/MontFerret/barn/git/ref/heads/main"):
			return jsonResponse(http.StatusOK, gitReference{Object: gitObject{SHA: "base"}}), nil
		case path == "/repos/MontFerret/barn/git/commits/base":
			return jsonResponse(http.StatusOK, gitCommit{Tree: gitObject{SHA: "tree"}}), nil
		case strings.Contains(path, "/repos/MontFerret/barn/contents/"):
			return apiFailure(http.StatusNotFound, "Not Found"), nil
		case path == "/repos/MontFerret/barn/pulls":
			return jsonResponse(http.StatusOK, []pullRequest{}), nil
		case request.Method == http.MethodGet && path == "/repos/MontFerret/barn/forks":
			return jsonResponse(http.StatusOK, []repository{}), nil
		case request.Method == http.MethodPost && path == "/repos/MontFerret/barn/forks":
			return jsonResponse(http.StatusAccepted, repository{FullName: "alice/barn"}), nil
		case path == "/repos/alice/barn":
			cancel()

			return apiFailure(http.StatusNotFound, "Not Found"), nil
		default:
			t.Fatalf("unexpected GitHub request: %s %s", request.Method, request.URL)

			return nil, nil
		}
	}}
	submitter, err := New(
		WithHTTPClient(&http.Client{Transport: transport}),
		WithTokenProvider(&fakeTokenProvider{token: "secret"}),
		WithPollInterval(time.Hour),
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = submitter.Submit(ctx, testPublication(barnpublish.NewVersion))
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context cancellation, got %v", err)
	}
}
