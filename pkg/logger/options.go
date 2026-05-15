package logger

import (
	"fmt"
	"strings"

	"github.com/rs/zerolog"
)

const (
	OutputStderr = "stderr"
	OutputFile   = "file"
	OutputNone   = "none"
)

type Options struct {
	Level        zerolog.Level
	LogOutput    string
	LogOutputSet bool
	LogFilename  string
	LogMaxSize   int
	LogMaxAge    int
}

func NewDefaultOptions() Options {
	return Options{
		Level:       zerolog.InfoLevel,
		LogOutput:   OutputStderr,
		LogFilename: "ferret.log",
		LogMaxSize:  10, // Mb
		LogMaxAge:   30, // Days
	}
}

func Outputs() []string {
	return []string{
		OutputStderr,
		OutputFile,
		OutputNone,
	}
}

func OutputsFmt() string {
	return fmt.Sprintf("\"%s\"", strings.Join(Outputs(), `"|"`))
}

func NormalizeOutput(input string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(input)), " ", "")
}

func (opts Options) Validate() error {
	switch NormalizeOutput(opts.LogOutput) {
	case OutputStderr, OutputNone:
		return nil
	case OutputFile:
		if strings.TrimSpace(opts.LogFilename) == "" {
			return fmt.Errorf("log file cannot be empty when log output is %q", OutputFile)
		}

		return nil
	default:
		return fmt.Errorf("invalid log output %q (expected %s)", opts.LogOutput, OutputsFmt())
	}
}

func (opts Options) Enabled() bool {
	opts = NormalizeOptions(opts)

	return NormalizeOutput(opts.LogOutput) != OutputNone
}

func NormalizeOptions(opts Options) Options {
	defaults := NewDefaultOptions()

	if opts == (Options{}) {
		return defaults
	}

	if opts.LogOutput == "" && !opts.LogOutputSet {
		opts.LogOutput = defaults.LogOutput
	}

	if opts.LogFilename == "" && NormalizeOutput(opts.LogOutput) != OutputFile {
		opts.LogFilename = defaults.LogFilename
	}

	if opts.LogMaxSize == 0 {
		opts.LogMaxSize = defaults.LogMaxSize
	}

	if opts.LogMaxAge == 0 {
		opts.LogMaxAge = defaults.LogMaxAge
	}

	return opts
}
