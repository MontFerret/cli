package repl

type writeFunc func([]byte) (int, error)

func (f writeFunc) Write(data []byte) (int, error) {
	return f(data)
}
