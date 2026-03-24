package logger

import "github.com/rs/zerolog"

type Options struct {
	Level       zerolog.Level
	LogFilename string
	LogMaxSize  int
	LogMaxAge   int
}

func NewDefaultOptions() Options {
	return Options{
		Level:       zerolog.InfoLevel,
		LogFilename: "ferret.log",
		LogMaxSize:  10, // Mb
		LogMaxAge:   30, // Days
	}
}
