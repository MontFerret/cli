package logger

import (
	"io"

	"github.com/natefinch/lumberjack"
	"github.com/rs/zerolog"
)

type Logger struct {
	logger     *zerolog.Logger
	fileWriter *lumberjack.Logger
}

func New(opts Options) *Logger {
	output := &lumberjack.Logger{
		Filename: opts.LogFilename,
		MaxSize:  opts.LogMaxSize,
		MaxAge:   opts.LogMaxAge,
	}

	l := zerolog.New(output).Level(opts.Level).With().Timestamp().Logger()

	logger := new(Logger)
	logger.fileWriter = output
	logger.logger = &l

	return logger
}

func (l *Logger) Log() *zerolog.Logger {
	return l.logger
}

func (l *Logger) Output() io.Writer {
	return l.fileWriter
}
