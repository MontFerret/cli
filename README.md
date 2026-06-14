# Ferret CLI

<p align="center">
	<a href="https://goreportcard.com/report/github.com/MontFerret/cli/v2">
		<img alt="Go Report Status" src="https://goreportcard.com/badge/github.com/MontFerret/cli/v2">
	</a>
	<a href="https://github.com/MontFerret/cli/actions">
		<img alt="Build Status" src="https://github.com/MontFerret/cli/workflows/build/badge.svg">
	</a>
	<a href="https://discord.gg/kzet32U">
		<img alt="Discord Chat" src="https://img.shields.io/discord/501533080880676864.svg">
	</a>
	<a href="https://github.com/MontFerret/cli/releases">
		<img alt="Ferret release" src="https://img.shields.io/github/release/MontFerret/cli.svg">
	</a>
	<a href="https://opensource.org/licenses/Apache-2.0">
		<img alt="Apache-2.0 License" src="http://img.shields.io/badge/license-Apache-brightgreen.svg">
	</a>
</p>

<p align="center">
	<img alt="Ferret" src="https://raw.githubusercontent.com/MontFerret/cli/master/assets/logo.svg" width="360px" />
</p>

> This branch contains the CLI for Ferret v2. For the stable v1 CLI, see the [`v1` branch](https://github.com/MontFerret/cli/tree/v1).

## What is this?

Ferret CLI is the command-line interface for [Ferret](https://github.com/MontFerret/ferret), a declarative query language and runtime for structured data extraction, browser automation, and data workflows.

Use it to run FQL scripts, format and check source files, inspect compiled bytecode, manage browser sessions, and debug local scripts.

Full documentation lives at [ferretlang.org](https://ferretlang.org/).

## Installation

Download a release from the [releases page](https://github.com/MontFerret/cli/releases), or install from source:

```bash
go install github.com/MontFerret/cli/v2/ferret@latest
```

Shell installer:

```bash
curl https://raw.githubusercontent.com/MontFerret/cli/master/install.sh | sh
```

## Quick start

Run the REPL:

```bash
ferret repl
```

Run an inline expression:

```bash
ferret run --eval 'RETURN "Hello, Ferret!"'
```

Run a script:

```bash
ferret run example.fql
```

Pass parameters:

```bash
ferret run example.fql --param url=https://example.com --param limit=10
```

Parameter values are parsed as JSON when possible. Values that are not valid JSON are passed as strings.

```bash
ferret run example.fql --param active=true
ferret run example.fql --param tags='["news","tech"]'
ferret run example.fql --param code='"123"'
```

Use parameters in FQL with `@name`:

```fql
LET page = DOCUMENT(@url)

RETURN ELEMENT(page, "title").innerText
```

## Common commands

```bash
ferret run script.fql       # Run a script
ferret exec script.fql      # Alias for run
ferret repl                 # Start the interactive shell
ferret check script.fql     # Check syntax and semantics
ferret fmt script.fql       # Format source
ferret build script.fql     # Compile to a bytecode artifact
ferret inspect script.fql   # Print compiled program details
ferret debug script.fql     # Start the interactive debugger
ferret browser open         # Start a managed browser
ferret config list          # Show configuration
ferret version              # Show version information
```

Run `ferret [command] --help` for command-specific options.

## Browser usage

Ferret can use Chrome or Chromium through the Chrome DevTools Protocol.

Open a managed browser:

```bash
ferret browser open
```

Run a script with a visible browser:

```bash
ferret run --browser-open script.fql
```

Run with a headless browser:

```bash
ferret run --browser-headless script.fql
```

Use an existing browser endpoint:

```bash
ferret run --browser-address http://127.0.0.1:9222 script.fql
```

## Debugging

Start the debugger for a local source file:

```bash
ferret debug script.fql
```

Useful debugger commands:

```text
break 12        Set a breakpoint
breakpoints     List breakpoints
continue        Resume execution
step            Step into
next            Step over
out             Step out
where           Show stack trace
locals          Show local variables
print <expr>    Evaluate a safe debug expression
quit            Exit
```

The debugger currently supports local source scripts with the builtin runtime. Compiled artifacts, remote debugging, DAP, conditional breakpoints, hit-count breakpoints, and logpoints are not supported yet.

## Configuration

Configuration values can come from command-line flags, environment variables, or the config file.

Priority order:

1. Command-line flags
2. Environment variables, for example `FERRET_RUNTIME`
3. Config file
4. Defaults

Config file locations:

- Linux/macOS: `~/.config/ferret/config.yaml`
- Windows: `%APPDATA%\ferret\config.yaml`

Examples:

```bash
ferret config set runtime builtin
ferret config set browser-address http://127.0.0.1:9222
ferret config get browser-address
ferret config list
```

## Development

Build and test locally:

```bash
git clone https://github.com/MontFerret/cli.git
cd cli

make compile
make test
```

Common development commands:

```bash
make fmt
make lint
make build
```

## Contributing

Issues and pull requests are welcome. Before opening a pull request, run the formatter, linter, and test suite.

## License

Apache-2.0
