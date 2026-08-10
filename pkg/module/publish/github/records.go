package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strings"

	barnpublish "github.com/MontFerret/barn/pkg/publish"
)

type record struct {
	Path    string
	Content []byte
}

func publicationRecords(publication *barnpublish.Result) ([]record, string, error) {
	if publication == nil || publication.Module == nil || publication.Version == nil {
		return nil, "", fmt.Errorf("prepared publication is incomplete")
	}

	if len(publication.Version.Commit) < 8 {
		return nil, "", fmt.Errorf("prepared publication commit is incomplete")
	}

	if publication.Kind != barnpublish.NewVersion {
		return nil, "", fmt.Errorf("prepared publication kind %q is unsupported", publication.Kind)
	}

	moduleRoot := path.Join("registry", "modules", publication.Module.Owner, publication.Module.Name)
	manifestPath := path.Join(moduleRoot, "manifest.json")
	versionPath := path.Join(moduleRoot, "versions", "v"+publication.Version.Version+".json")

	allowed := map[string]bool{versionPath: true}
	allowed[manifestPath] = true
	records := make([]record, 0, len(publication.Files))
	seen := make(map[string]struct{}, len(publication.Files))

	for _, file := range publication.Files {
		clean := path.Clean(file.Path)
		if clean != file.Path || strings.HasPrefix(clean, "../") || !allowed[clean] {
			return nil, "", fmt.Errorf("prepared publication contains unexpected path %q", file.Path)
		}

		if _, exists := seen[clean]; exists {
			return nil, "", fmt.Errorf("prepared publication contains duplicate path %q", clean)
		}

		seen[clean] = struct{}{}
		if bytes.Contains(file.Content, []byte(`"publishedAt"`)) {
			var document map[string]json.RawMessage
			if err := json.Unmarshal(file.Content, &document); err != nil {
				return nil, "", fmt.Errorf("parse prepared record %s: %w", clean, err)
			}

			if _, exists := document["publishedAt"]; exists {
				return nil, "", fmt.Errorf("prepared publication must not contain publishedAt in %s", clean)
			}
		}

		records = append(records, record{Path: clean, Content: append([]byte{}, file.Content...)})
	}
	if len(records) != len(allowed) {
		return nil, "", fmt.Errorf("prepared publication contains %d records, want %d", len(records), len(allowed))
	}

	sort.Slice(records, func(i, j int) bool { return records[i].Path < records[j].Path })

	return records, versionPath, nil
}

func sameRecordSet(files []pullFile, records []record) bool {
	if len(files) != len(records) {
		return false
	}

	expected := make(map[string]struct{}, len(records))
	for _, item := range records {
		expected[item.Path] = struct{}{}
	}

	for _, file := range files {
		if file.Status != "added" {
			return false
		}

		if _, exists := expected[file.Filename]; !exists {
			return false
		}

		delete(expected, file.Filename)
	}

	return len(expected) == 0
}
