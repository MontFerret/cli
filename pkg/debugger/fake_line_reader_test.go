package debugger

import "io"

type lineResult struct {
	line string
	err  error
}

type fakeLineReader struct {
	results []lineResult
	index   int
}

func (f *fakeLineReader) Readline() (string, error) {
	if f.index >= len(f.results) {
		return "", io.EOF
	}

	result := f.results[f.index]
	f.index++
	return result.line, result.err
}
