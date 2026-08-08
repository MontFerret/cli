package mod

import (
	"bytes"
	"strings"
	"testing"
)

func TestReadlinePromptUsesConfiguredStreams(t *testing.T) {
	output := new(bytes.Buffer)
	prompt, err := newReadlinePrompt(strings.NewReader("acme/sqlite\n"), output)
	if err != nil {
		t.Fatal(err)
	}
	defer prompt.Close()

	value, err := prompt.Readline("Name: ")
	if err != nil {
		t.Fatal(err)
	}
	if value != "acme/sqlite" || !strings.Contains(output.String(), "Name: ") {
		t.Fatalf("unexpected prompt result: value=%q output=%q", value, output.String())
	}
}
