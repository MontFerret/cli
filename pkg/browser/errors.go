package browser

import "errors"

var (
	ErrBinNotFound  = errors.New("no compatible browser was found")
	ErrProcNotFound = errors.New("process not found")
)
