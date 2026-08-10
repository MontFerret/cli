package github

import (
	"strings"
	"testing"

	barnpublish "github.com/MontFerret/barn/pkg/publish"
)

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
