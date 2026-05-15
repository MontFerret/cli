package logger

import (
	"io"
	"os"
	"time"

	"github.com/natefinch/lumberjack"
	"github.com/rs/zerolog"
)

type Logger struct {
	logger     *zerolog.Logger
	output     io.Writer
	fileWriter *lumberjack.Logger
}

func New(opts Options) (*Logger, error) {
	opts = NormalizeOptions(opts)

	if err := opts.Validate(); err != nil {
		return nil, err
	}

	logger := new(Logger)
	output := newOutput(opts, logger)

	if output == nil {
		l := zerolog.Nop()
		logger.logger = &l

		return logger, nil
	}

	l := zerolog.New(output).Level(opts.Level).With().Timestamp().Logger()

	logger.output = output
	logger.logger = &l

	return logger, nil
}

func (l *Logger) Log() *zerolog.Logger {
	return l.logger
}

func (l *Logger) Output() io.Writer {
	if l == nil {
		return nil
	}

	return l.output
}

func (l *Logger) Close() error {
	if l == nil || l.fileWriter == nil {
		return nil
	}

	return l.fileWriter.Close()
}

func newOutput(opts Options, logger *Logger) io.Writer {
	switch NormalizeOutput(opts.LogOutput) {
	case OutputStderr:
		return zerolog.ConsoleWriter{
			Out:        os.Stderr,
			TimeFormat: time.RFC3339,
		}
	case OutputFile:
		output := &lumberjack.Logger{
			Filename: opts.LogFilename,
			MaxSize:  opts.LogMaxSize,
			MaxAge:   opts.LogMaxAge,
		}
		logger.fileWriter = output

		return output
	default:
		return nil
	}
}
