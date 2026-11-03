package output

import (
	"bytes"
	"context"
	"errors"
	"io"
	"testing"

	"github.com/MontFerret/cli/v2/internal/testutil/encoded"
)

func TestWriteChunksAndNewlines(t *testing.T) {
	for _, test := range []struct {
		name    string
		chunks  [][]byte
		newline bool
		want    string
	}{
		{"chunks", [][]byte{[]byte("a"), {}, []byte("b")}, false, "ab"},
		{"newline", [][]byte{[]byte("a"), []byte("b")}, true, "ab\n"},
		{"existing newline", [][]byte{[]byte("a\n")}, true, "a\n"},
		{"empty", [][]byte{{}}, true, ""},
		{"absent", nil, true, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			src := &encoded.Output{Chunks: test.chunks}
			var dst bytes.Buffer
			terminal, failure := Write(t.Context(), &dst, src, test.newline)
			if terminal != nil || failure != nil || dst.String() != test.want || src.Closed != 1 {
				t.Fatalf("data=%q terminal=%v failure=%v closed=%d", dst.String(), terminal, failure, src.Closed)
			}
		})
	}
}

func TestWritePartialOutputAndFailures(t *testing.T) {
	executionErr, closeErr := errors.New("execution failed"), errors.New("cleanup failed")
	src := &encoded.Output{Chunks: [][]byte{[]byte("available")}, Err: executionErr, CloseErr: closeErr}
	var dst bytes.Buffer
	terminal, failure := Write(t.Context(), &dst, src, true)
	if !errors.Is(terminal, executionErr) || !errors.Is(failure, closeErr) || dst.String() != "available\n" || src.Closed != 1 {
		t.Fatalf("data=%q terminal=%v failure=%v closed=%d", dst.String(), terminal, failure, src.Closed)
	}
}

func TestWriteWriterFailureAndShortWrites(t *testing.T) {
	writeErr := errors.New("write failed")
	for _, failure := range []error{writeErr, nil} {
		src := &encoded.Output{Chunks: [][]byte{[]byte("abc")}}
		_, err := Write(t.Context(), writeFunc(func([]byte) (int, error) { return 0, failure }), src, true)
		want := failure
		if want == nil {
			want = io.ErrShortWrite
		}

		if !errors.Is(err, want) || src.Closed != 1 {
			t.Fatalf("error=%v closed=%d", err, src.Closed)
		}
	}

	var dst bytes.Buffer
	src := &encoded.Output{Chunks: [][]byte{[]byte("abc")}}
	_, err := Write(context.Background(), writeFunc(func(data []byte) (int, error) { return dst.Write(data[:1]) }), src, false)
	if err != nil || dst.String() != "abc" {
		t.Fatalf("data=%q error=%v", dst.String(), err)
	}
}

func TestWriteDoesNotRepeatRetainedCleanupFailure(t *testing.T) {
	failure := errors.New("retained cleanup failure")
	src := &encoded.Output{Err: failure, CloseErr: failure}
	var dst bytes.Buffer
	err, fatal := Write(t.Context(), &dst, src, false)
	if !errors.Is(fatal, failure) || err.Error() != failure.Error() {
		t.Fatalf("error=%v fatal=%v", err, fatal)
	}
}
