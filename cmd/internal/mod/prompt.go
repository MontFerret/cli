package mod

import (
	"errors"
	"io"
	"os"
	"sync"

	"github.com/chzyer/readline"
)

type readlinePrompt struct {
	instance *readline.Instance
	close    sync.Once
	err      error
}

func newReadlinePrompt(input io.Reader, output io.Writer) (prompt, error) {
	config := &readline.Config{
		Prompt:                 "",
		HistoryLimit:           -1,
		DisableAutoSaveHistory: true,
		InterruptPrompt:        "\n",
		EOFPrompt:              "\n",
		Stdin:                  io.NopCloser(input),
		Stdout:                 output,
		Stderr:                 output,
		FuncIsTerminal:         readline.DefaultIsTerminal,
	}

	if _, ok := input.(*os.File); !ok {
		config.ForceUseInteractive = true
		config.FuncMakeRaw = func() error { return nil }
		config.FuncExitRaw = func() error { return nil }
		config.FuncOnWidthChanged = func(func()) {}
		config.FuncGetWidth = func() int { return 80 }
	}

	instance, err := readline.NewEx(config)
	if err != nil {
		return nil, err
	}

	return &readlinePrompt{instance: instance}, nil
}

func (prompt *readlinePrompt) Readline(label string) (string, error) {
	prompt.instance.SetPrompt(label)
	value, err := prompt.instance.Readline()
	if errors.Is(err, readline.ErrInterrupt) || errors.Is(err, io.EOF) {
		return "", errPromptCanceled
	}

	return value, err
}

func (prompt *readlinePrompt) Close() error {
	prompt.close.Do(func() {
		prompt.err = prompt.instance.Close()
	})

	return prompt.err
}
