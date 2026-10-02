package run

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"

	"github.com/spf13/cobra"
)

func TestRunOutputWithError(t *testing.T) {
	for _, test := range []struct {
		output io.ReadCloser
		want   string
	}{
		{nil, ""}, {io.NopCloser(strings.NewReader("")), ""}, {io.NopCloser(strings.NewReader("available")), "available"},
	} {
		cmd := &cobra.Command{}
		var stdout, stderr bytes.Buffer
		cmd.SetOut(&stdout)
		cmd.SetErr(&stderr)
		failure := errors.New("execution or cleanup failed")
		err := writeResult(cmd, test.output, failure)
		if !errors.Is(err, failure) || !strings.Contains(stderr.String(), failure.Error()) {
			t.Fatalf("output=%q stderr=%q error=%v", stdout.String(), stderr.String(), err)
		}

		if stdout.String() != test.want {
			t.Fatalf("output changed: %q", stdout.String())
		}
	}
}

func TestRunSuccessfulEmptyOutput(t *testing.T) {
	cmd := &cobra.Command{}
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	if err := writeResult(cmd, io.NopCloser(strings.NewReader("")), nil); err != nil || stdout.Len() != 0 || stderr.Len() != 0 {
		t.Fatalf("empty result: stdout=%q stderr=%q error=%v", stdout.String(), stderr.String(), err)
	}
}

func TestRunJoinsExecutionCopyAndReaderCloseErrors(t *testing.T) {
	executionErr := errors.New("execution failed")
	copyErr := errors.New("reading output failed")
	closeErr := errors.New("closing output failed")
	output := &failingCloseReader{
		Reader: io.MultiReader(strings.NewReader("available"), iotest.ErrReader(copyErr)),
		err:    closeErr,
	}
	cmd := &cobra.Command{}
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)

	err := writeResult(cmd, output, executionErr)
	for _, want := range []error{executionErr, copyErr, closeErr} {
		if !errors.Is(err, want) || !strings.Contains(stderr.String(), want.Error()) {
			t.Fatalf("failure lost: %v; stderr=%q", err, stderr.String())
		}
	}

	if stdout.String() != "available" {
		t.Fatalf("available output lost: %q", stdout.String())
	}
}
