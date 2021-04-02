package browser

import "fmt"

type Options struct {
	Detach   bool
	Headless bool
	Address  string
	Port     uint64
	UserDir  string
}

func NewDefaultOptions() Options {
	return Options{
		Headless: false,
		Address:  "",
		Port:     9222,
		UserDir:  "",
	}
}

func (opts Options) ToURL() string {
	url := opts.Address

	if url == "" {
		url = "http://127.0.0.1"
	}

	return fmt.Sprintf("%s:%d", url, opts.Port)
}

func (opts Options) ToFlags() []string {
	flags := make([]string, 0, len(headlessFlags)+5)

	if opts.Headless {
		flags = append(flags, headlessFlags...)
	}

	if opts.UserDir != "" {
		flags = append(flags, fmt.Sprintf("--user-data-dir=%s", opts.UserDir))
	}

	if opts.Address != "" {
		flags = append(flags, fmt.Sprintf("--remote-debugging-address=%s", opts.Address))
	}

	flags = append(flags, fmt.Sprintf("--remote-debugging-port=%d", opts.Port))

	return flags
}
