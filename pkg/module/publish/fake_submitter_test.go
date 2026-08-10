package publish

import (
	"context"

	barnpublish "github.com/MontFerret/barn/pkg/publish"
)

type fakeSubmitter struct {
	submission *Submission
	prepared   *barnpublish.Result
	err        error
	calls      int
}

func (f *fakeSubmitter) Submit(_ context.Context, prepared *barnpublish.Result) (*Submission, error) {
	f.calls++
	f.prepared = prepared

	return f.submission, f.err
}
