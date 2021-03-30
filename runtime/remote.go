package runtime

import (
	"bytes"
	"context"
	"encoding/json"
	"github.com/pkg/errors"
	"io"
	"net/http"
	"net/url"
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
		return "", err
	}

	info := remoteInfo{}

	if err := json.Unmarshal(data, &info); err != nil {
		return "", errors.Wrap(err, "deserialize response data")
	}

	return info.Version.Ferret, nil
}

func (rt *Remote) Run(ctx context.Context, query string, params map[string]interface{}) ([]byte, error) {
	body, err := json.Marshal(&remoteQuery{
		Text:   query,
		Params: params,
	})

	if err != nil {
		return nil, errors.Wrap(err, "serialize query")
	}

	return rt.makeRequest(ctx, "POST", "/", body)
}

func (rt *Remote) createRequest(ctx context.Context, method, endpoint string, body []byte) (*http.Request, error) {
	var reader io.Reader = nil

	if body != nil {
		reader = bytes.NewReader(body)
	}

	u2, err := url.Parse(endpoint)

	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, method, rt.url.ResolveReference(u2).String(), reader)

	if err != nil {
		return nil, err
	}

	if rt.opts.Headers != nil {
		rt.opts.Headers.ForEach(func(value []string, key string) bool {
			for _, v := range value {
				req.Header.Add(key, v)
			}

			return true
		})
	}

	req.Header.Set("Content-Type", "application/jsonw")

	return req, nil
}

func (rt *Remote) makeRequest(ctx context.Context, method, endpoint string, body []byte) ([]byte, error) {
	req, err := rt.createRequest(ctx, method, endpoint, body)

	if err != nil {
		return nil, errors.Wrap(err, "create request")
	}

	resp, err := rt.client.Do(req)

	if err != nil {
		return nil, errors.Wrap(err, "make HTTP request to remote runtime")
	}

	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)

	if err != nil {
		return nil, errors.Wrap(err, "read response data")
	}

	return data, nil
}
