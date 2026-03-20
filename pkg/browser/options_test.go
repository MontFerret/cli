package browser

import (
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
		Port:     9222,
	}

	flags := opts.ToFlags()
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

	flags := opts.ToFlags()
	joined := strings.Join(flags, " ")

	if !strings.Contains(joined, "--user-data-dir=/tmp/chrome") {
		t.Errorf("expected --user-data-dir=/tmp/chrome in %q", joined)
	}
}

func TestOptions_ToFlags_NoHeadless(t *testing.T) {
	opts := Options{
		Port: 9222,
	}

	flags := opts.ToFlags()
	joined := strings.Join(flags, " ")

	if strings.Contains(joined, "--headless") {
		t.Error("unexpected --headless flag")
	}
}

func TestOptions_ToFlags_RemoteAddress(t *testing.T) {
	opts := Options{
		Address: "192.168.1.1",
		Port:    9222,
	}

	flags := opts.ToFlags()
	joined := strings.Join(flags, " ")

	if !strings.Contains(joined, "--remote-debugging-address=192.168.1.1") {
		t.Errorf("expected --remote-debugging-address in %q", joined)
	}
}
