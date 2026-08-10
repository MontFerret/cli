package github

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"sort"
	"testing"

	barnpublish "github.com/MontFerret/barn/pkg/publish"
	registryspec "github.com/MontFerret/specs/pkg/registry"
)

func jsonResponse(status int, value any) *http.Response {
	data, _ := json.Marshal(value)

	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(data)),
	}
}

func apiFailure(status int, message string) *http.Response {
	return jsonResponse(status, map[string]string{"message": message})
}

func encodedContent(data []byte) map[string]string {
	return map[string]string{"encoding": "base64", "content": base64.StdEncoding.EncodeToString(data)}
}

func testPublication(kind barnpublish.Kind) *barnpublish.Result {
	module := &registryspec.ModuleManifest{
		Owner: "acme", Name: "widget",
		Source: registryspec.Source{Repository: "https://github.com/acme/widget", Path: "modules/widget"},
	}
	version := &registryspec.VersionRecord{
		Version: "1.2.3", Tag: "modules/widget/v1.2.3", Commit: "abcdef0123456789abcdef0123456789abcdef01",
	}
	modulePath := "registry/modules/acme/widget/manifest.json"
	versionPath := "registry/modules/acme/widget/versions/v1.2.3.json"
	files := []barnpublish.File{{Path: versionPath, Content: []byte("{\"version\":\"1.2.3\"}\n")}}
	if kind == barnpublish.NewModule {
		files = append(files, barnpublish.File{Path: modulePath, Content: []byte("{\"owner\":\"acme\"}\n")})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })

	return &barnpublish.Result{Kind: kind, Module: module, Version: version, Files: files}
}

func readJSONBody(t *testing.T, request *http.Request, target any) {
	t.Helper()
	if err := json.NewDecoder(request.Body).Decode(target); err != nil {
		t.Fatalf("decode %s %s body: %v", request.Method, request.URL, err)
	}
}
