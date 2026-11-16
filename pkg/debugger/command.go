package debugger

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	apidebug "github.com/MontFerret/api/debugger"
	"github.com/MontFerret/api/source"
)

type CommandName string

const (
	CommandEmpty       CommandName = ""
	CommandHelp        CommandName = "help"
	CommandBreak       CommandName = "break"
	CommandDelete      CommandName = "delete"
	CommandBreakpoints CommandName = "breakpoints"
	CommandContinue    CommandName = "continue"
	CommandStep        CommandName = "step"
	CommandNext        CommandName = "next"
	CommandOut         CommandName = "out"
	CommandPause       CommandName = "pause"
	CommandWhere       CommandName = "where"
	CommandLocals      CommandName = "locals"
	CommandPrint       CommandName = "print"
	CommandQuit        CommandName = "quit"
)

type Command struct {
	Name              CommandName
	Argument          string
	Location          source.Location
	BreakpointOptions apidebug.BreakpointOptions
	BreakpointID      apidebug.BreakpointID
}

var aliases = map[string]CommandName{
	"b":    CommandBreak,
	"d":    CommandDelete,
	"bp":   CommandBreakpoints,
	"bl":   CommandBreakpoints,
	"c":    CommandContinue,
	"s":    CommandStep,
	"n":    CommandNext,
	"w":    CommandWhere,
	"bt":   CommandWhere,
	"l":    CommandLocals,
	"p":    CommandPrint,
	"e":    CommandPrint,
	"eval": CommandPrint,
	"q":    CommandQuit,
}

func ParseCommand(input string) (Command, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return Command{}, nil
	}

	name := input
	argument := ""
	if idx := strings.IndexFunc(input, unicode.IsSpace); idx >= 0 {
		name = input[:idx]
		argument = strings.TrimSpace(input[idx:])
	}

	commandName := CommandName(strings.ToLower(name))
	if alias, ok := aliases[string(commandName)]; ok {
		commandName = alias
	}

	command := Command{Name: commandName, Argument: argument}

	switch commandName {
	case CommandHelp, CommandBreakpoints, CommandContinue, CommandStep, CommandNext, CommandOut, CommandPause, CommandWhere, CommandLocals, CommandQuit:
		if argument != "" {
			return Command{}, fmt.Errorf("%s does not accept arguments", commandName)
		}
	case CommandBreak:
		location, options, err := parseBreakpoint(argument)
		if err != nil {
			return Command{}, err
		}

		command.Location = location
		command.BreakpointOptions = options
	case CommandDelete:
		id, err := parsePositiveNumber(argument, "usage: delete <breakpoint-id>")
		if err != nil {
			return Command{}, err
		}

		command.BreakpointID = apidebug.BreakpointID(id)
	case CommandPrint:
		if argument == "" {
			return Command{}, fmt.Errorf("usage: print <expression>")
		}
	default:
		return Command{}, fmt.Errorf("unknown command: %s", name)
	}

	return command, nil
}

const breakpointUsage = "usage: break [--exact|--next|--in-function] <line>[:<column>] or <file>:<line>[:<column>]"

func parseBreakpoint(argument string) (source.Location, apidebug.BreakpointOptions, error) {
	var location source.Location
	options := apidebug.BreakpointOptions{BindingMode: apidebug.BreakpointBindNextExecutableInSource}
	tokens := strings.Fields(argument)

	if len(tokens) == 0 {
		return location, options, errors.New(breakpointUsage)
	}

	locationText := ""
	modeSet := false

	for _, token := range tokens {
		var mode apidebug.BreakpointBindingMode
		switch token {
		case "--exact":
			mode = apidebug.BreakpointBindExact
		case "--next":
			mode = apidebug.BreakpointBindNextExecutableInSource
		case "--in-function":
			mode = apidebug.BreakpointBindNextExecutableInFunction
		default:
			if strings.HasPrefix(token, "--") {
				return location, options, fmt.Errorf("unknown break option: %s", token)
			}

			if locationText != "" {
				return location, options, errors.New(breakpointUsage)
			}

			locationText = token

			continue
		}

		if modeSet {
			return location, options, errors.New("break binding options are mutually exclusive")
		}

		modeSet = true
		options.BindingMode = mode
	}

	if locationText == "" {
		return location, options, errors.New(breakpointUsage)
	}

	location, err := parseBreakpointLocation(locationText)
	if err != nil {
		return source.Location{}, options, err
	}

	return location, options, nil
}

func parseBreakpointLocation(value string) (source.Location, error) {
	var location source.Location
	lastColon := strings.LastIndex(value, ":")

	if lastColon < 0 {
		line, err := parsePositiveNumber(value, breakpointUsage)
		if err != nil {
			return location, err
		}

		location.Line = line

		return location, nil
	}

	last, err := parsePositiveNumber(value[lastColon+1:], breakpointUsage)
	if err != nil {
		return location, err
	}

	prefix := value[:lastColon]
	if prefix == "" {
		return location, errors.New(breakpointUsage)
	}

	previousColon := strings.LastIndex(prefix, ":")
	lineText := prefix

	if previousColon >= 0 {
		lineText = prefix[previousColon+1:]
	}

	if line, numeric, err := parseOptionalPositiveNumber(lineText); numeric {
		if err != nil {
			return location, errors.New(breakpointUsage)
		}

		location.Line = line
		location.Column = last

		if previousColon >= 0 {
			location.SourceName = prefix[:previousColon]

			if location.SourceName == "" {
				return source.Location{}, errors.New(breakpointUsage)
			}
		}

		return location, nil
	}

	location.SourceName = prefix
	location.Line = last

	return location, nil
}

func parseOptionalPositiveNumber(value string) (int, bool, error) {
	number, err := strconv.Atoi(value)
	if err != nil {
		return 0, false, nil
	}

	if number <= 0 {
		return 0, true, errors.New("number must be positive")
	}

	return number, true, nil
}

func parsePositiveNumber(value, usage string) (int, error) {
	number, err := strconv.Atoi(value)
	if err != nil || number <= 0 {
		return 0, errors.New(usage)
	}

	return number, nil
}
