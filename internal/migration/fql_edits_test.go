package migration

import "testing"

func TestApplyFQLEdits(t *testing.T) {
	content := "πabc"
	edits := []fqlSourceEdit{
		{start: 0, end: 0, text: "("},
		{start: 3, end: 4, text: "long"},
		{start: 5, end: 5, text: ")"},
	}
	got, err := applyFQLEdits(content, edits)
	if err != nil {
		t.Fatal(err)
	}

	if got != "(πalongc)" {
		t.Fatalf("edits used shifted offsets: %q", got)
	}
}

func TestApplyFQLEditsRejectsInvalidSpans(t *testing.T) {
	tests := [][]fqlSourceEdit{
		{{start: -1, end: 0}},
		{{start: 3, end: 2}},
		{{start: 0, end: 6}},
		{{start: 6, end: 6}},
		{{start: 1, end: 2}},
		{{start: 0, end: 1}},
		{{start: 2, end: 4}, {start: 3, end: 5}},
		{{start: 2, end: 5}, {start: 3, end: 3}},
		{{start: 2, end: 2}, {start: 2, end: 2}},
	}
	for _, edits := range tests {
		if _, err := applyFQLEdits("πabc", edits); err == nil {
			t.Fatalf("accepted invalid edits: %#v", edits)
		}
	}
}
