package logger

import (
	"fmt"
	"github.com/rs/zerolog"
	"strings"
)

func Levels() []string {
	return []string{
		zerolog.DebugLevel.String(),
		zerolog.InfoLevel.String(),
		zerolog.WarnLevel.String(),
		zerolog.ErrorLevel.String(),
		zerolog.FatalLevel.String(),
	}
}

func LevelsFmt() string {
	return fmt.Sprintf("\"%s\"", strings.Join(Levels(), `"|"`))
}

func ToLevel(input string) zerolog.Level {
	lvl, err := zerolog.ParseLevel(strings.ReplaceAll(strings.ToLower(input), " ", ""))

	if err != nil {
		return zerolog.InfoLevel
	}

	return lvl
}
