package migration

import (
	"cmp"
	"fmt"
	"slices"
	"unicode/utf8"
)

type fqlSourceEdit struct {
	start int
	end   int
	text  string
}

// Edits use byte offsets in the original source. Applying them from right to left
// lets loop insertions and nested call renames share one parse without shifting spans.
func applyFQLEdits(content string, edits []fqlSourceEdit) (string, error) {
	slices.SortFunc(edits, func(a, b fqlSourceEdit) int {
		return cmp.Compare(b.start, a.start)
	})

	size := len(content)
	boundary := len(content)
	for i, edit := range edits {
		if edit.start < 0 || edit.end < edit.start || edit.end > boundary ||
			(i > 0 && edit.start == edits[i-1].start) {
			return "", fmt.Errorf("apply Ferret source edits: invalid or overlapping span")
		}

		if (edit.start < len(content) && !utf8.RuneStart(content[edit.start])) ||
			(edit.end < len(content) && !utf8.RuneStart(content[edit.end])) {
			return "", fmt.Errorf("apply Ferret source edits: span splits a UTF-8 rune")
		}

		size += len(edit.text) - (edit.end - edit.start)
		boundary = edit.start
	}

	output := make([]byte, size)
	read, write := len(content), size
	for _, edit := range edits {
		write -= read - edit.end
		copy(output[write:], content[edit.end:read])
		write -= len(edit.text)
		copy(output[write:], edit.text)
		read = edit.start
	}

	copy(output[:write], content[:read])

	return string(output), nil
}
