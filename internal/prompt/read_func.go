package prompt

type readFunc func([]byte) (int, error)

func (read readFunc) Read(b []byte) (int, error) {
	return read(b)
}
