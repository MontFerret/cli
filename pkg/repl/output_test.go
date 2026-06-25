package repl

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"
	"testing/iotest"
)

func TestWriteResultAddsTrailingNewline(t *testing.T) {
	var out bytes.Buffer

	err := writeResult(&out, io.NopCloser(strings.NewReader("1")))

	if err != nil {
		t.Fatal(err)
	}

	if out.String() != "1\n" {
		t.Fatalf("expected output %q, got %q", "1\n", out.String())
	}
}

func TestWriteResultPreservesExistingTrailingNewline(t *testing.T) {
	var out bytes.Buffer

	err := writeResult(&out, io.NopCloser(strings.NewReader("1\n")))

	if err != nil {
		t.Fatal(err)
	}

	if out.String() != "1\n" {
		t.Fatalf("expected output %q, got %q", "1\n", out.String())
	}
}

func TestWriteResultKeepsEmptyOutputEmpty(t *testing.T) {
	var out bytes.Buffer

	err := writeResult(&out, io.NopCloser(strings.NewReader("")))

	if err != nil {
		t.Fatal(err)
	}

	if out.String() != "" {
		t.Fatalf("expected empty output, got %q", out.String())
	}
}

func TestWriteResultPropagatesReadError(t *testing.T) {
	readErr := errors.New("read failed")
	var out bytes.Buffer

	err := writeResult(&out, io.NopCloser(iotest.ErrReader(readErr)))

	if !errors.Is(err, readErr) {
		t.Fatalf("expected read error %v, got %v", readErr, err)
	}
}

func TestWriteResultPropagatesWriteError(t *testing.T) {
	writeErr := errors.New("write failed")

	err := writeResult(writerFunc(func([]byte) (int, error) {
		return 0, writeErr
	}), io.NopCloser(strings.NewReader("1")))

	if !errors.Is(err, writeErr) {
		t.Fatalf("expected write error %v, got %v", writeErr, err)
	}
}

func TestWriteResultClosesSource(t *testing.T) {
	var out bytes.Buffer
	src := &trackingReadCloser{Reader: strings.NewReader("1")}

	err := writeResult(&out, src)

	if err != nil {
		t.Fatal(err)
	}

	if !src.closed {
		t.Fatal("expected source to be closed")
	}
}

type trackingReadCloser struct {
	*strings.Reader
	closed bool
}

func (rc *trackingReadCloser) Close() error {
	rc.closed = true

	return nil
}

type writerFunc func([]byte) (int, error)

func (fn writerFunc) Write(data []byte) (int, error) {
	return fn(data)
}

func TestWriteResult_CloseError(t *testing.T) {
	t.Run("join errors when both fail", func(t *testing.T) {
		mainErr := errors.New("main error")
		closeErr := errors.New("close error")
		src := &errorCloseReader{Reader: strings.NewReader("data"), closeErr: closeErr}

		err := writeResult(writerFunc(func(_ []byte) (int, error) {
			return 0, mainErr
		}), src)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, mainErr) {
			t.Errorf("expected main error, got %v", err)
		}
		if !errors.Is(err, closeErr) {
			t.Errorf("expected close error, got %v", err)
		}
	})

	t.Run("return close error when main succeeds", func(t *testing.T) {
		closeErr := errors.New("close error")
		src := &errorCloseReader{Reader: strings.NewReader("data"), closeErr: closeErr}

		var buf bytes.Buffer
		err := writeResult(&buf, src)

		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !errors.Is(err, closeErr) {
			t.Errorf("expected close error, got %v", err)
		}
		if buf.String() != "data\n" {
			t.Errorf("expected data\\n, got %q", buf.String())
		}
	})
}

func TestWriteAll_ZeroWrite(t *testing.T) {
	err := writeAll(writerFunc(func(_ []byte) (int, error) {
		return 0, nil
	}), []byte("data"))

	if !errors.Is(err, io.ErrShortWrite) {
		t.Errorf("expected io.ErrShortWrite, got %v", err)
	}
}

type errorCloseReader struct {
	io.Reader
	closeErr error
}

func (e *errorCloseReader) Close() error {
	return e.closeErr
}
