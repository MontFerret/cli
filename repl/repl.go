package repl

import (
	"context"
	"fmt"
	"strings"

	"github.com/chzyer/readline"

	"github.com/MontFerret/cli/runtime"
)

func Start(ctx context.Context, rt runtime.Runtime, params map[string]interface{}) error {
	version, err := rt.Version(ctx)

	if err != nil {
		return err
	}

	fmt.Println(fmt.Sprintf("Welcome to Ferret REPL %s\n", version))
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

		out, err := rt.Run(ctx, query, params)

		if err != nil {
			fmt.Println("Failed to execute the query")
			fmt.Println(err)
			continue
		}

		fmt.Println(string(out))
	}

	return nil
}
