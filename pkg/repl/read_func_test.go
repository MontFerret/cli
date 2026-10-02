package repl

type readFunc func([]byte) (int, error)

func (f readFunc) Read(data []byte) (int, error) {
	return f(data)
}
