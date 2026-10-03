package runtime

type closeFunc func() error

func (f closeFunc) Close() error {
	return f()
}
