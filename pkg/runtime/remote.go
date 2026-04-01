package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/MontFerret/ferret/v2/pkg/source"
)

type (
	remoteVersion struct {
		Worker string `json:"worker"`
		Ferret string `json:"ferret"`
	}

	remoteInfo struct {
		IP      string        `json:"ip"`
		Version remoteVersion `json:"version"`
	}

	remoteQuery struct {
		Text   string                 `json:"text"`
		Params map[string]interface{} `json:"params"`
	}

	Remote struct {
		url    url.URL
		opts   Options
		client *http.Client
	}
)

func NewRemote(url url.URL, opts Options) Runtime {
	rt := new(Remote)
	rt.url = url
	rt.opts = opts
	rt.client = http.DefaultClient

	return rt
}

func (rt *Remote) Version(ctx context.Context) (string, error) {
	data, err := rt.makeRequest(ctx, "GET", "/info", nil)

	if err != nil {
		return "", fmt.Errorf("make request: %w", err)
	}

	info := remoteInfo{}

	b, err := io.ReadAll(data)

	if err != nil {
		return "", fmt.Errorf("read response data: %w", err)
	}

	if err := json.Unmarshal(b, &info); err != nil {
		return "", fmt.Errorf("deserialize response data: %w", err)
	}

	return info.Version.Ferret, nil
}

func (rt *Remote) Run(ctx context.Context, query *source.Source, params map[string]any) (io.ReadCloser, error) {
	body, err := json.Marshal(&remoteQuery{
		Text:   query.Content(),
		Params: params,
	})

	if err != nil {
		return nil, fmt.Errorf("serialize query: %w", err)
	}

	return rt.makeRequest(ctx, "POST", "/", body)
}

func (rt *Remote) RunArtifact(_ context.Context, _ []byte, _ map[string]any) (io.ReadCloser, error) {
	return nil, ErrArtifactRequiresBuiltinRuntime
}

func (rt *Remote) createRequest(ctx context.Context, method, endpoint string, body []byte) (*http.Request, error) {
	var reader io.Reader

	if body != nil {
		reader = bytes.NewReader(body)
	}

	u2, err := url.Parse(endpoint)

	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, rt.url.ResolveReference(u2).String(), reader)

	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if rt.opts.Headers != nil && len(rt.opts.Headers.Data) > 0 {
		for key := range rt.opts.Headers.Data {
			value := rt.opts.Headers.Data.Get(key)
			req.Header.Set(key, value)
		}
	}

	req.Header.Set("Content-Type", "application/json")

	return req, nil
}

func (rt *Remote) makeRequest(ctx context.Context, method, endpoint string, body []byte) (io.ReadCloser, error) {
	req, err := rt.createRequest(ctx, method, endpoint, body)

	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := rt.client.Do(req)

	if err != nil {
		return nil, fmt.Errorf("make HTTP request to remote runtime: %w", err)
	}

	return resp.Body, nil
}
