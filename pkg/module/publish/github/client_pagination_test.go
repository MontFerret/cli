package github

import (
	"context"
	"net/http"
	"testing"
)

func TestClientPaginatesGitHubCollections(t *testing.T) {
	t.Run("pull requests", func(t *testing.T) {
		pages := 0
		transport := &scriptedTransport{handle: func(request *http.Request) (*http.Response, error) {
			pages++
			if request.URL.Query().Get("page") == "1" {
				return jsonResponse(http.StatusOK, make([]pullRequest, 100)), nil
			}

			return jsonResponse(http.StatusOK, []pullRequest{{Number: 101}}), nil
		}}
		client := newClient("https://api.github.test", &http.Client{Transport: transport}, "secret")

		items, err := client.pullRequests(context.Background(), upstreamOwner, upstreamName, "main")
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 101 || items[100].Number != 101 || pages != 2 {
			t.Fatalf("unexpected pagination result: len=%d pages=%d", len(items), pages)
		}
	})

	t.Run("pull request files", func(t *testing.T) {
		pages := 0
		transport := &scriptedTransport{handle: func(request *http.Request) (*http.Response, error) {
			pages++
			if request.URL.Query().Get("page") == "1" {
				return jsonResponse(http.StatusOK, make([]pullFile, 100)), nil
			}

			return jsonResponse(http.StatusOK, []pullFile{{Filename: "last.json"}}), nil
		}}
		client := newClient("https://api.github.test", &http.Client{Transport: transport}, "secret")

		items, err := client.pullFiles(context.Background(), upstreamOwner, upstreamName, 7)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 101 || items[100].Filename != "last.json" || pages != 2 {
			t.Fatalf("unexpected pagination result: len=%d pages=%d", len(items), pages)
		}
	})

	t.Run("forks", func(t *testing.T) {
		pages := 0
		transport := &scriptedTransport{handle: func(request *http.Request) (*http.Response, error) {
			pages++
			if request.URL.Query().Get("page") == "1" {
				return jsonResponse(http.StatusOK, make([]repository, 100)), nil
			}

			return jsonResponse(http.StatusOK, []repository{{FullName: "alice/barn"}}), nil
		}}
		client := newClient("https://api.github.test", &http.Client{Transport: transport}, "secret")

		items, err := client.forks(context.Background(), upstreamOwner, upstreamName)
		if err != nil {
			t.Fatal(err)
		}
		if len(items) != 101 || items[100].FullName != "alice/barn" || pages != 2 {
			t.Fatalf("unexpected pagination result: len=%d pages=%d", len(items), pages)
		}
	})
}
