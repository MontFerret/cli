package browser

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestOptions_ToURL_Default(t *testing.T) {
	opts := NewDefaultOptions()
	url := opts.ToURL()

	if url != "http://127.0.0.1:9222" {
		t.Errorf("expected http://127.0.0.1:9222, got %s", url)
	}
}

func TestOptions_ToURL_CustomAddress(t *testing.T) {
	opts := Options{
		Address: "http://localhost",
		Port:    9333,
	}

	url := opts.ToURL()

	if url != "http://localhost:9333" {
		t.Errorf("expected http://localhost:9333, got %s", url)
	}
}

func TestOptions_ToFlags_Headless(t *testing.T) {
	opts := Options{
		Headless: true,
		UserDir:  "/tmp/chrome",
		Port:     9222,
	}

	flags, err := opts.ToFlags()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	joined := strings.Join(flags, " ")

	if !strings.Contains(joined, "--headless") {
		t.Error("expected --headless flag")
	}

	if !strings.Contains(joined, "--remote-debugging-port=9222") {
		t.Error("expected --remote-debugging-port=9222")
	}
}

func TestOptions_ToFlags_UserDir(t *testing.T) {
	opts := Options{
		UserDir: "/tmp/chrome",
		Port:    9222,
	}

	flags, err := opts.ToFlags()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	joined := strings.Join(flags, " ")

	if !strings.Contains(joined, "--user-data-dir=/tmp/chrome") {
		t.Errorf("expected --user-data-dir=/tmp/chrome in %q", joined)
	}
}

func TestOptions_ToFlags_DefaultUserDir(t *testing.T) {
	wd := t.TempDir()
	t.Chdir(wd)

	flags, err := (Options{Port: 9222}).ToFlags()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := filepath.Join(wd, defaultUserDir)
	joined := strings.Join(flags, " ")

	if !strings.Contains(joined, "--user-data-dir="+expected) {
		t.Errorf("expected --user-data-dir=%s in %q", expected, joined)
	}
}

func TestOptions_ToFlags_PreservesExplicitUserDir(t *testing.T) {
	flags, err := (Options{UserDir: "/tmp/chrome"}).ToFlags()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	joined := strings.Join(flags, " ")

	if !strings.Contains(joined, "--user-data-dir=/tmp/chrome") {
		t.Fatalf("expected explicit user dir to be preserved, got %q", joined)
	}
}

func TestOptions_ToFlags_GetwdError(t *testing.T) {
	prev := getwd
	getwd = func() (string, error) {
		return "", errors.New("boom")
	}
	t.Cleanup(func() {
		getwd = prev
	})

	_, err := (Options{}).ToFlags()

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "resolve browser user data dir") {
		t.Fatalf("expected wrapped error, got %v", err)
	}
}

func TestOptions_ToFlags_NoHeadless(t *testing.T) {
	opts := Options{
		UserDir: "/tmp/chrome",
		Port:    9222,
	}

	flags, err := opts.ToFlags()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	joined := strings.Join(flags, " ")

	if strings.Contains(joined, "--headless") {
		t.Error("unexpected --headless flag")
	}
}

func TestOptions_ToFlags_RemoteAddress(t *testing.T) {
	opts := Options{
		Address: "192.168.1.1",
		UserDir: "/tmp/chrome",
		Port:    9222,
	}

	flags, err := opts.ToFlags()

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	joined := strings.Join(flags, " ")

	if !strings.Contains(joined, "--remote-debugging-address=192.168.1.1") {
		t.Errorf("expected --remote-debugging-address in %q", joined)
	}
}
