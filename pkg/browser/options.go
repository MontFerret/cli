package browser

import (
	"fmt"
	"os"
	"path/filepath"
)

const defaultUserDir = ".ferret-browser"

var getwd = os.Getwd

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

func (opts Options) ToFlags() ([]string, error) {
	flags := make([]string, 0, len(headlessFlags)+5)

	if opts.Headless {
		flags = append(flags, headlessFlags...)
	}

	userDir := opts.UserDir

	if userDir == "" {
		cwd, err := getwd()

		if err != nil {
			return nil, fmt.Errorf("resolve browser user data dir: %w", err)
		}

		userDir = filepath.Join(cwd, defaultUserDir)
	}

	flags = append(flags, fmt.Sprintf("--user-data-dir=%s", userDir))

	if opts.Address != "" {
		flags = append(flags, fmt.Sprintf("--remote-debugging-address=%s", opts.Address))
	}

	flags = append(flags, fmt.Sprintf("--remote-debugging-port=%d", opts.Port))

	return flags, nil
}
