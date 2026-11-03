package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/MontFerret/api"
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
)

// Remote adapts HTTP Worker execution and metadata to api.Runtime. Responses
// stream their encoded bytes through caller-owned output. Worker does not expose
// reusable compilation or debugging; use builtin or Wire for compilation.
type Remote struct {
	url    url.URL
	opts   Options
	client *http.Client
}

var _ api.Runtime = (*Remote)(nil)

func newRemote(url url.URL, opts Options) *Remote {
	rt := new(Remote)
	rt.url = url
	rt.opts = opts
	rt.client = http.DefaultClient

	return rt
}

// Version reports the Worker's Ferret version unchanged and closes its response.
func (rt *Remote) Version(ctx context.Context) (version api.Version, err error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}

	response, err := rt.makeRequest(ctx, "GET", "/info", nil)
	if err != nil {
		return "", fmt.Errorf("make request: %w", err)
	}

	defer func() {
		if closeErr := response.Body.Close(); closeErr != nil {
			err = errors.Join(err, fmt.Errorf("close response data: %w", closeErr))
		}
	}()

	info := remoteInfo{}

	b, err := io.ReadAll(response.Body)
	if err != nil {
		return "", fmt.Errorf("read response data: %w", err)
	}

	if err := json.Unmarshal(b, &info); err != nil {
		return "", fmt.Errorf("deserialize response data: %w", err)
	}

	return api.Version(info.Version.Ferret), nil
}

// Run submits source and merged parameters to Worker. It owns the response body
// and preserves available output alongside response read or cleanup failures.
func (rt *Remote) Run(ctx context.Context, src api.Source, setters ...api.SessionOption) (api.Output, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	opts, err := newRemoteSessionOptions(setters)
	if err != nil {
		return nil, fmt.Errorf("configure HTTP Worker execution: %w", err)
	}

	body, err := json.Marshal(&remoteQuery{
		Text:   src.Content,
		Params: opts.params,
	})
	if err != nil {
		return nil, fmt.Errorf("serialize query: %w", err)
	}

	requestCtx, cancel := context.WithCancel(ctx)
	response, err := rt.makeRequest(requestCtx, "POST", "/", body)
	if err != nil {
		cancel()
		return nil, err
	}

	return newHTTPOutput(ctx, cancel, response), nil
}

// Compile rejects reusable compilation because Worker only exposes execution.
func (rt *Remote) Compile(ctx context.Context, _ api.Source, _ ...api.PlanOption) (api.Plan, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return nil, fmt.Errorf("HTTP Worker runtime does not support compilation; use builtin or Wire: %w", errors.ErrUnsupported)
}

// CompileDebug rejects debug compilation without making a Worker request.
func (rt *Remote) CompileDebug(ctx context.Context, _ api.Source, _ ...api.PlanOption) (api.Plan, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	return nil, fmt.Errorf("HTTP Worker runtime does not support debug compilation; use builtin for CLI debugging: %w", errors.ErrUnsupported)
}

// Close is an idempotent no-op: Remote borrows the shared HTTP client and closes
// metadata responses within Version; execution outputs own their responses.
// Close leaves the adapter and client usable.
func (rt *Remote) Close() error {
	return nil
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

func (rt *Remote) makeRequest(ctx context.Context, method, endpoint string, body []byte) (*http.Response, error) {
	req, err := rt.createRequest(ctx, method, endpoint, body)

	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := rt.client.Do(req)

	if err != nil {
		return nil, fmt.Errorf("make HTTP request to remote runtime: %w", err)
	}

	return resp, nil
}
