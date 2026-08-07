package registryclient

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// Client navigates and decodes the public Ferret registry distribution.
type Client struct {
	base       *url.URL
	httpClient HTTPClient
	maxBody    int64
}

// New constructs a registry client. A nil HTTP client uses a bounded default.
func New(baseURL string, httpClient HTTPClient) (*Client, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("parse registry base URL: %w", err)
	}

	if !base.IsAbs() || base.Host == "" || (base.Scheme != "https" && base.Scheme != "http") {
		return nil, fmt.Errorf("registry base URL must be an absolute HTTP or HTTPS URL")
	}

	if base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return nil, fmt.Errorf("registry base URL must not contain credentials, a query, or a fragment")
	}

	if !strings.HasSuffix(base.Path, "/") {
		base.Path += "/"
	}

	if httpClient == nil {
		httpClient = &http.Client{Timeout: defaultTimeout}
	}

	return &Client{base: base, httpClient: httpClient, maxBody: maxResponseSize}, nil
}

// Root fetches the registry root index.
func (c *Client) Root(ctx context.Context) (*RootIndex, error) {
	var document rootDocument
	if _, err := c.fetch(ctx, c.base.String(), false, &document); err != nil {
		return nil, err
	}

	if err := validateRootDocument(&document); err != nil {
		return nil, err
	}

	if _, err := c.resolve(document.Artifacts["modules"]); err != nil {
		return nil, err
	}

	return &RootIndex{SchemaVersion: document.SchemaVersion, ModulesHref: document.Artifacts["modules"]}, nil
}

// Catalog fetches the module catalog discovered through the registry root.
func (c *Client) Catalog(ctx context.Context) (*Catalog, error) {
	root, err := c.Root(ctx)
	if err != nil {
		return nil, err
	}

	var document catalogDocument
	if _, err := c.fetch(ctx, root.ModulesHref, false, &document); err != nil {
		return nil, err
	}

	if err := validateCatalogDocument(&document); err != nil {
		return nil, err
	}

	result := &Catalog{Modules: make([]ModuleReference, len(document.Modules))}
	for i, item := range document.Modules {
		if _, err := c.resolve(item.Href); err != nil {
			return nil, err
		}

		result.Modules[i] = ModuleReference(item)
	}

	return result, nil
}

// Module fetches one module document by a catalog-provided link.
func (c *Client) Module(ctx context.Context, href string) (*Module, error) {
	var document moduleDocument
	if _, err := c.fetch(ctx, href, true, &document); err != nil {
		return nil, err
	}

	if err := validateModuleDocument(&document); err != nil {
		return nil, err
	}

	result := &Module{
		ID:          document.ID,
		Owner:       document.Owner,
		Name:        document.Name,
		Description: document.Description,
		License:     document.License,
		Latest:      document.Latest,
		Versions:    make([]ModuleVersionReference, len(document.Versions)),
	}

	for i, item := range document.Versions {
		if _, err := c.resolve(item.Href); err != nil {
			return nil, err
		}

		result.Versions[i] = ModuleVersionReference(item)
	}

	return result, nil
}

// Version fetches one immutable module version document.
func (c *Client) Version(ctx context.Context, href string) (*Version, error) {
	var document versionDocument
	documentURL, err := c.fetch(ctx, href, true, &document)
	if err != nil {
		return nil, err
	}

	if err := validateVersionDocument(&document); err != nil {
		return nil, err
	}

	documentation, err := documentURL.Parse(document.Content["documentation"])
	if err != nil {
		return nil, fmt.Errorf("%w: resolve documentation link: %v", ErrMalformed, err)
	}
	if documentation.Scheme != documentURL.Scheme || documentation.Host != documentURL.Host {
		return nil, fmt.Errorf("%w: documentation link points outside the registry origin", ErrMalformed)
	}

	return &Version{
		ID:        document.ID,
		Version:   document.Version,
		Namespace: document.Namespace,
		Ferret:    document.Ferret,
		Source: Source{
			Repository: document.Source.Repository,
			Path:       document.Source.Path,
			Commit:     document.Source.Commit,
		},
		Documentation: documentation.String(),
	}, nil
}

func (c *Client) fetch(ctx context.Context, href string, notFound bool, destination any) (*url.URL, error) {
	target, err := c.resolve(href)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, target.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: create request for %s: %v", ErrUnavailable, target, err)
	}

	request.Header.Set("Accept", "application/json")
	response, err := c.httpClient.Do(request)

	if err != nil {
		return nil, fmt.Errorf("%w: fetch %s: %v", ErrUnavailable, target, err)
	}

	defer response.Body.Close()

	if response.Request != nil && response.Request.URL != nil && (response.Request.URL.Scheme != c.base.Scheme || response.Request.URL.Host != c.base.Host) {
		return nil, fmt.Errorf("%w: registry request redirected outside the registry origin", ErrMalformed)
	}

	if response.StatusCode == http.StatusNotFound && notFound {
		return nil, fmt.Errorf("%w: %s", ErrNotFound, target)
	}

	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("%w: fetch %s: HTTP %s", ErrUnavailable, target, response.Status)
	}

	data, err := io.ReadAll(io.LimitReader(response.Body, c.maxBody+1))
	if err != nil {
		return nil, fmt.Errorf("%w: read %s: %v", ErrUnavailable, target, err)
	}

	if int64(len(data)) > c.maxBody {
		return nil, fmt.Errorf("%w: response from %s exceeds %d bytes", ErrMalformed, target, c.maxBody)
	}

	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(destination); err != nil {
		return nil, fmt.Errorf("%w: decode %s: %v", ErrMalformed, target, err)
	}

	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("response contains more than one JSON value")
		}
		return nil, fmt.Errorf("%w: decode %s: %v", ErrMalformed, target, err)
	}

	return target, nil
}

func (c *Client) resolve(href string) (*url.URL, error) {
	reference, err := url.Parse(href)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid registry link %q: %v", ErrMalformed, href, err)
	}

	if reference.User != nil || reference.RawQuery != "" || reference.Fragment != "" {
		return nil, fmt.Errorf("%w: registry link %q contains credentials, a query, or a fragment", ErrMalformed, href)
	}

	target := c.base.ResolveReference(reference)
	if target.Scheme != c.base.Scheme || target.Host != c.base.Host {
		return nil, fmt.Errorf("%w: registry link %q points outside the registry origin", ErrMalformed, href)
	}

	return target, nil
}
