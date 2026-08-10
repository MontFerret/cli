package github

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	barnpublish "github.com/MontFerret/barn/pkg/publish"
)

func TestSubmitterRejectsConflictingPullRequest(t *testing.T) {
	publication := testPublication(barnpublish.NewVersion)
	records, versionPath, err := publicationRecords(publication)
	if err != nil {
		t.Fatal(err)
	}
	transport := &scriptedTransport{handle: func(request *http.Request) (*http.Response, error) {
		switch request.URL.Path {
		case "/repos/MontFerret/barn/pulls":
			return jsonResponse(http.StatusOK, []pullRequest{{
				Number: 12, HTMLURL: "https://github.com/MontFerret/barn/pull/12",
				Head: pullHead{SHA: "head", Repo: &repository{FullName: "alice/barn"}},
			}}), nil
		case "/repos/MontFerret/barn/pulls/12/files":
			return jsonResponse(http.StatusOK, []pullFile{
				{Filename: versionPath, Status: "added"},
				{Filename: "README.md", Status: "modified"},
			}), nil
		default:
			t.Fatalf("unexpected GitHub request: %s %s", request.Method, request.URL)

			return nil, nil
		}
	}}
	submitter, err := New(WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatal(err)
	}

	_, err = submitter.findPullRequest(
		context.Background(),
		newClient("https://api.github.test", &http.Client{Transport: transport}, "secret"),
		"main",
		records,
		versionPath,
	)
	if !errors.Is(err, ErrPullConflict) || !strings.Contains(err.Error(), "https://github.com/MontFerret/barn/pull/12") {
		t.Fatalf("unexpected pull request conflict: %v", err)
	}
}

func TestSubmitterPreservesPullRequestContentReadFailure(t *testing.T) {
	publication := testPublication(barnpublish.NewVersion)
	records, versionPath, err := publicationRecords(publication)
	if err != nil {
		t.Fatal(err)
	}
	transport := &scriptedTransport{handle: func(request *http.Request) (*http.Response, error) {
		switch request.URL.Path {
		case "/repos/MontFerret/barn/pulls":
			return jsonResponse(http.StatusOK, []pullRequest{{
				Number: 13, HTMLURL: "https://github.com/MontFerret/barn/pull/13",
				Head: pullHead{SHA: "head", Repo: &repository{FullName: "alice/barn"}},
			}}), nil
		case "/repos/MontFerret/barn/pulls/13/files":
			return jsonResponse(http.StatusOK, []pullFile{{Filename: versionPath, Status: "added"}}), nil
		default:
			if strings.Contains(request.URL.Path, "/repos/alice/barn/contents/") {
				return apiFailure(http.StatusInternalServerError, "content unavailable"), nil
			}

			t.Fatalf("unexpected GitHub request: %s %s", request.Method, request.URL)

			return nil, nil
		}
	}}
	submitter, err := New(WithHTTPClient(&http.Client{Transport: transport}))
	if err != nil {
		t.Fatal(err)
	}

	_, err = submitter.findPullRequest(
		context.Background(),
		newClient("https://api.github.test", &http.Client{Transport: transport}, "secret"),
		"main",
		records,
		versionPath,
	)
	if err == nil || errors.Is(err, ErrPullConflict) ||
		!strings.Contains(err.Error(), "read pull request record "+versionPath) ||
		!strings.Contains(err.Error(), "HTTP 500") {
		t.Fatalf("unexpected pull request content failure: %v", err)
	}
}

func TestSubmitterRecoversPullRequestCreationRace(t *testing.T) {
	publication := testPublication(barnpublish.NewVersion)
	versionFile := publication.Files[0]
	pullLookups := 0
	mutations := make([]string, 0, 4)
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
			pullLookups++
			if pullLookups == 1 {
				return jsonResponse(http.StatusOK, []pullRequest{}), nil
			}

			return jsonResponse(http.StatusOK, []pullRequest{{
				Number: 44, HTMLURL: "https://github.com/MontFerret/barn/pull/44",
				Head: pullHead{SHA: "branch-commit", Repo: &repository{FullName: "alice/barn"}},
			}}), nil
		case path == "/repos/MontFerret/barn/pulls/44/files":
			return jsonResponse(http.StatusOK, []pullFile{{Filename: versionFile.Path, Status: "added"}}), nil
		case strings.Contains(path, "/repos/alice/barn/contents/"):
			return jsonResponse(http.StatusOK, encodedContent(versionFile.Content)), nil
		case path == "/repos/MontFerret/barn/forks":
			return jsonResponse(http.StatusOK, []repository{{FullName: "alice/barn", Owner: user{Login: "alice"}}}), nil
		case path == "/repos/alice/barn":
			return jsonResponse(http.StatusOK, repository{FullName: "alice/barn", Permissions: permissions{Push: true}}), nil
		case strings.HasPrefix(path, "/repos/alice/barn/git/ref/heads/"):
			return apiFailure(http.StatusNotFound, "Not Found"), nil
		case path == "/repos/alice/barn/git/trees":
			mutations = append(mutations, "tree")

			return jsonResponse(http.StatusCreated, createdTree{SHA: "new-tree"}), nil
		case path == "/repos/alice/barn/git/commits":
			mutations = append(mutations, "commit")

			return jsonResponse(http.StatusCreated, createdCommit{SHA: "branch-commit"}), nil
		case path == "/repos/alice/barn/git/refs":
			mutations = append(mutations, "reference")

			return jsonResponse(http.StatusCreated, gitReference{}), nil
		case path == "/repos/MontFerret/barn/pulls" && request.Method == http.MethodPost:
			mutations = append(mutations, "pull")

			return apiFailure(http.StatusUnprocessableEntity, "pull request already exists"), nil
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
	if result.URL != "https://github.com/MontFerret/barn/pull/44" || !result.Existing || pullLookups != 2 || strings.Join(mutations, ",") != "tree,commit,reference,pull" {
		t.Fatalf("unexpected race recovery: result=%#v lookups=%d mutations=%v", result, pullLookups, mutations)
	}
}
