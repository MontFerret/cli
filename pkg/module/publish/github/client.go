package github

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

const apiVersion = "2026-03-10"

type client struct {
	baseURL    string
	httpClient *http.Client
	token      string
}

func newClient(baseURL string, httpClient *http.Client, token string) *client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}

	return &client{baseURL: strings.TrimRight(baseURL, "/"), httpClient: httpClient, token: token}
}

func (c *client) currentUser(ctx context.Context) (*user, error) {
	result := new(user)
	if err := c.do(ctx, http.MethodGet, apiPath("user"), nil, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *client) repository(ctx context.Context, owner, name string) (*repository, error) {
	result := new(repository)
	if err := c.do(ctx, http.MethodGet, apiPath("repos", owner, name), nil, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *client) reference(ctx context.Context, owner, name, branch string) (*gitReference, error) {
	result := new(gitReference)
	if err := c.do(ctx, http.MethodGet, apiPath("repos", owner, name, "git", "ref", "heads", branch), nil, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *client) gitCommit(ctx context.Context, owner, name, sha string) (*gitCommit, error) {
	result := new(gitCommit)
	if err := c.do(ctx, http.MethodGet, apiPath("repos", owner, name, "git", "commits", sha), nil, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *client) commitDetails(ctx context.Context, owner, name, sha string) (*commitDetails, error) {
	result := new(commitDetails)
	if err := c.do(ctx, http.MethodGet, apiPath("repos", owner, name, "commits", sha), nil, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *client) content(ctx context.Context, owner, name, filename, ref string) ([]byte, error) {
	query := url.Values{"ref": []string{ref}}
	result := new(contentResponse)
	if err := c.do(ctx, http.MethodGet, contentPath(owner, name, filename)+"?"+query.Encode(), nil, result); err != nil {
		return nil, err
	}

	if result.Encoding != "base64" {
		return nil, fmt.Errorf("GitHub returned unsupported content encoding %q", result.Encoding)
	}

	data, err := base64.StdEncoding.DecodeString(strings.ReplaceAll(result.Content, "\n", ""))
	if err != nil {
		return nil, fmt.Errorf("decode GitHub content for %s: %w", filename, err)
	}

	return data, nil
}

func (c *client) pullRequests(ctx context.Context, owner, name, base string) ([]pullRequest, error) {
	var result []pullRequest
	for page := 1; ; page++ {
		query := url.Values{
			"base":     []string{base},
			"page":     []string{strconv.Itoa(page)},
			"per_page": []string{"100"},
			"state":    []string{"open"},
		}

		var batch []pullRequest
		if err := c.do(ctx, http.MethodGet, apiPath("repos", owner, name, "pulls")+"?"+query.Encode(), nil, &batch); err != nil {
			return nil, err
		}

		result = append(result, batch...)
		if len(batch) < 100 {
			return result, nil
		}
	}
}

func (c *client) pullFiles(ctx context.Context, owner, name string, number int) ([]pullFile, error) {
	var result []pullFile

	for page := 1; ; page++ {
		query := url.Values{"page": []string{strconv.Itoa(page)}, "per_page": []string{"100"}}

		var batch []pullFile
		if err := c.do(ctx, http.MethodGet, apiPath("repos", owner, name, "pulls", strconv.Itoa(number), "files")+"?"+query.Encode(), nil, &batch); err != nil {
			return nil, err
		}

		result = append(result, batch...)
		if len(batch) < 100 {
			return result, nil
		}
	}
}

func (c *client) forks(ctx context.Context, owner, name string) ([]repository, error) {
	var result []repository

	for page := 1; ; page++ {
		query := url.Values{"page": []string{strconv.Itoa(page)}, "per_page": []string{"100"}}
		var batch []repository
		if err := c.do(ctx, http.MethodGet, apiPath("repos", owner, name, "forks")+"?"+query.Encode(), nil, &batch); err != nil {
			return nil, err
		}

		result = append(result, batch...)
		if len(batch) < 100 {
			return result, nil
		}
	}
}

func (c *client) createFork(ctx context.Context, owner, name string) (*repository, error) {
	result := new(repository)
	body := map[string]any{"default_branch_only": true}

	if err := c.do(ctx, http.MethodPost, apiPath("repos", owner, name, "forks"), body, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *client) createTree(ctx context.Context, owner, name, baseTree string, files []record) (*createdTree, error) {
	entries := make([]map[string]string, len(files))

	for index, file := range files {
		entries[index] = map[string]string{
			"path": file.Path, "mode": "100644", "type": "blob", "content": string(file.Content),
		}
	}

	result := new(createdTree)
	body := map[string]any{"base_tree": baseTree, "tree": entries}

	if err := c.do(ctx, http.MethodPost, apiPath("repos", owner, name, "git", "trees"), body, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *client) createCommit(ctx context.Context, owner, name, message, tree, parent string) (*createdCommit, error) {
	result := new(createdCommit)
	body := map[string]any{"message": message, "tree": tree, "parents": []string{parent}}

	if err := c.do(ctx, http.MethodPost, apiPath("repos", owner, name, "git", "commits"), body, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *client) createReference(ctx context.Context, owner, name, branch, sha string) error {
	body := map[string]string{"ref": "refs/heads/" + branch, "sha": sha}

	return c.do(ctx, http.MethodPost, apiPath("repos", owner, name, "git", "refs"), body, new(gitReference))
}

func (c *client) createPullRequest(ctx context.Context, owner, name, title, body, head, base string) (*pullRequest, error) {
	result := new(pullRequest)
	request := map[string]any{
		"title": title, "body": body, "head": head, "base": base, "maintainer_can_modify": true,
	}

	if err := c.do(ctx, http.MethodPost, apiPath("repos", owner, name, "pulls"), request, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (c *client) do(ctx context.Context, method, path string, body, output any) error {
	var input io.Reader

	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode GitHub request: %w", err)
		}

		input = bytes.NewReader(data)
	}

	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, input)
	if err != nil {
		return fmt.Errorf("create GitHub request: %w", err)
	}

	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("Authorization", "Bearer "+c.token)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("User-Agent", "ferret-cli")
	request.Header.Set("X-GitHub-Api-Version", apiVersion)

	response, err := c.httpClient.Do(request)
	if err != nil {
		return fmt.Errorf("send GitHub request: %w", err)
	}

	defer response.Body.Close()

	data, err := io.ReadAll(io.LimitReader(response.Body, 4<<20))
	if err != nil {
		return fmt.Errorf("read GitHub response: %w", err)
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		var document struct {
			Message string `json:"message"`
		}

		_ = json.Unmarshal(data, &document)

		return &apiError{
			StatusCode:  response.StatusCode,
			Message:     document.Message,
			Permissions: response.Header.Get("X-Accepted-GitHub-Permissions"),
		}
	}

	if output == nil || len(data) == 0 {
		return nil
	}

	if err := json.Unmarshal(data, output); err != nil {
		return fmt.Errorf("decode GitHub response: %w", err)
	}

	return nil
}
