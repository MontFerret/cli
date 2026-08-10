package github

import (
	"net/http"
	"strings"
	"testing"
)

func submissionFailureTransport(t *testing.T, failure string) *scriptedTransport {
	t.Helper()

	return &scriptedTransport{handle: func(request *http.Request) (*http.Response, error) {
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
			if failure == "fork" {
				return apiFailure(http.StatusInternalServerError, "fork failed"), nil
			}

			return jsonResponse(http.StatusOK, []repository{{FullName: "alice/barn", Owner: user{Login: "alice"}}}), nil
		case path == "/repos/alice/barn":
			return jsonResponse(http.StatusOK, repository{FullName: "alice/barn", Permissions: permissions{Push: true}}), nil
		case strings.HasPrefix(path, "/repos/alice/barn/git/ref/heads/"):
			return apiFailure(http.StatusNotFound, "Not Found"), nil
		case path == "/repos/alice/barn/git/trees":
			if failure == "tree" {
				return apiFailure(http.StatusInternalServerError, "tree failed"), nil
			}

			return jsonResponse(http.StatusCreated, createdTree{SHA: "new-tree"}), nil
		case path == "/repos/alice/barn/git/commits":
			if failure == "commit" {
				return apiFailure(http.StatusInternalServerError, "commit failed"), nil
			}

			return jsonResponse(http.StatusCreated, createdCommit{SHA: "new-commit"}), nil
		case path == "/repos/alice/barn/git/refs":
			if failure == "reference" {
				return apiFailure(http.StatusInternalServerError, "reference failed"), nil
			}

			return jsonResponse(http.StatusCreated, gitReference{}), nil
		case path == "/repos/MontFerret/barn/pulls" && request.Method == http.MethodPost:
			return apiFailure(http.StatusInternalServerError, "pull request failed"), nil
		default:
			t.Fatalf("unexpected GitHub request: %s %s", request.Method, request.URL)

			return nil, nil
		}
	}}
}
