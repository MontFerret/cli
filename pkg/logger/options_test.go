package logger

import (
	"strings"
	"testing"
)

func TestOptionsValidateRejectsInvalidOutput(t *testing.T) {
	opts := NewDefaultOptions()
	opts.LogOutput = "stdout"

	err := opts.Validate()

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "invalid log output") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewAcceptsZeroValueOptions(t *testing.T) {
	log, err := New(Options{})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if log.Output() == nil {
		t.Fatal("expected default log output")
	}
}

func TestOptionsValidateRejectsEmptyFile(t *testing.T) {
	opts := NewDefaultOptions()
	opts.LogOutput = OutputFile
	opts.LogFilename = ""

	err := opts.Validate()

	if err == nil {
		t.Fatal("expected error")
	}

	if !strings.Contains(err.Error(), "log file cannot be empty") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLevelsIncludesTrace(t *testing.T) {
	for _, level := range Levels() {
		if level == "trace" {
			return
		}
	}

	t.Fatal("expected trace level")
}
