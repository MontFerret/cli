package github

import (
	"bytes"
	"strings"
	"testing"

	barnpublish "github.com/MontFerret/barn/pkg/publish"
)

func TestPublicationRecordsAcceptBarnRecordShapes(t *testing.T) {
	for _, test := range []struct {
		name      string
		kind      barnpublish.Kind
		wantPaths []string
	}{
		{
			name: "new module",
			kind: barnpublish.NewModule,
			wantPaths: []string{
				"registry/modules/acme/widget/manifest.json",
				"registry/modules/acme/widget/versions/v1.2.3.json",
			},
		},
		{
			name:      "new version",
			kind:      barnpublish.NewVersion,
			wantPaths: []string{"registry/modules/acme/widget/versions/v1.2.3.json"},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			publication := testPublication(test.kind)
			records, versionPath, err := publicationRecords(publication)
			if err != nil {
				t.Fatal(err)
			}
			if versionPath != "registry/modules/acme/widget/versions/v1.2.3.json" || len(records) != len(test.wantPaths) {
				t.Fatalf("unexpected publication records: versionPath=%q records=%#v", versionPath, records)
			}

			for index, wantPath := range test.wantPaths {
				if records[index].Path != wantPath {
					t.Fatalf("record %d path = %q, want %q", index, records[index].Path, wantPath)
				}

				var wantContent []byte
				for _, file := range publication.Files {
					if file.Path == wantPath {
						wantContent = file.Content
						break
					}
				}
				if !bytes.Equal(records[index].Content, wantContent) {
					t.Fatalf("record %s did not preserve prepared bytes", wantPath)
				}
			}
		})
	}
}

func TestPublicationRecordsRejectUnexpectedAndStampedFiles(t *testing.T) {
	for _, test := range []struct {
		name    string
		mutate  func(*barnpublish.Result)
		message string
	}{
		{
			name: "unexpected path",
			mutate: func(publication *barnpublish.Result) {
				publication.Files[1].Path = "dist/modules/acme/widget.json"
			},
			message: "unexpected path",
		},
		{
			name: "publication timestamp",
			mutate: func(publication *barnpublish.Result) {
				publication.Files[1].Content = []byte("{\"publishedAt\":\"2026-08-09T00:00:00Z\"}\n")
			},
			message: "must not contain publishedAt",
		},
		{
			name: "duplicate path",
			mutate: func(publication *barnpublish.Result) {
				publication.Files = append(publication.Files, publication.Files[0])
			},
			message: "duplicate path",
		},
		{
			name: "unknown kind",
			mutate: func(publication *barnpublish.Result) {
				publication.Kind = barnpublish.Kind("future-kind")
			},
			message: "unsupported",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			publication := testPublication(barnpublish.NewModule)
			test.mutate(publication)

			_, _, err := publicationRecords(publication)
			if err == nil || !strings.Contains(err.Error(), test.message) {
				t.Fatalf("unexpected validation error: %v", err)
			}
		})
	}
}
