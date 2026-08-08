package mod

import (
	"bufio"
	"io"
)

type scriptedPrompt struct {
	input  *bufio.Reader
	output io.Writer
}

func newScriptedPrompt(input io.Reader, output io.Writer) (prompt, error) {
	return &scriptedPrompt{input: bufio.NewReader(input), output: output}, nil
}

func (prompt *scriptedPrompt) Readline(label string) (string, error) {
	if _, err := io.WriteString(prompt.output, label); err != nil {
		return "", err
	}

	line, err := prompt.input.ReadString('\n')
	if err != nil {
		return "", errPromptCanceled
	}

	return line, nil
}

func (*scriptedPrompt) Close() error {
	return nil
}
