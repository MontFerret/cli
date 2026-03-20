package repl

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/chzyer/readline"

	"github.com/MontFerret/ferret/v2/pkg/file"

	"github.com/MontFerret/cli/pkg/runtime"
)

func Start(ctx context.Context, opts runtime.Options, params map[string]interface{}) error {
	rt, err := runtime.New(opts)

	if err != nil {
		return err
	}

	version, err := rt.Version(ctx)

	if err != nil {
		return err
	}

	fmt.Printf("Welcome to Ferret REPL %s\n", version)
	fmt.Println("Please use `exit` or `Ctrl-D` to exit this program.")

	rl, err := readline.NewEx(&readline.Config{
		Prompt:          "> ",
		InterruptPrompt: "^C",
		EOFPrompt:       "exit",
	})

	if err != nil {
		return err
	}

	defer rl.Close()

	var commands []string
	var multiline bool

	ctx, cancel := context.WithCancel(context.Background())

	exit := func() {
		cancel()
	}

	for {
		line, err := rl.Readline()

		if err != nil {
			break
		}

		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		if strings.HasPrefix(line, "%") {
			line = line[1:]

			multiline = !multiline
		}

		if multiline {
			commands = append(commands, line)
			continue
		}

		commands = append(commands, line)
		query := strings.TrimSpace(strings.Join(commands, "\n"))
		commands = make([]string, 0, 10)

		if query == "" {
			continue
		}

		if query == "exit" {
			exit()

			break
		}

		out, err := rt.Run(ctx, file.NewAnonymousSource(query), params)

		if err != nil {
			fmt.Println("Failed to execute the query")
			fmt.Println(err)
			continue
		}

		_, err = io.Copy(os.Stdout, out)

		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			break
		}
	}

	return nil
}
