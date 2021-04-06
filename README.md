# Ferret CLI

<p align="center">
	<a href="https://goreportcard.com/report/github.com/MontFerret/cli">
		<img alt="Go Report Status" src="https://goreportcard.com/badge/github.com/MontFerret/cli">
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
	<a href="http://opensource.org/licenses/MIT">
		<img alt="MIT License" src="http://img.shields.io/badge/license-MIT-brightgreen.svg">
	</a>
</p>

Documentation is available [at our website](https://www.montferret.dev/docs/introduction/).

## Installation

### Binary
You can download the latest binaries from [here](https://github.com/MontFerret/cli/releases).

### Source
* Go >=1.16

```bash
go get https://github.com/MontFerret/cli
```

## Quick start

### REPL

```bash
ferret exec
Welcome to Ferret REPL 0.14.1

Please use `exit` or `Ctrl-D` to exit this program.
```

### Script execution
```bash
ferret exec my-script.fql
```

### With browser

```bash
ferret exec --browser-open my-script.fql
```

#### As headless

```bash
ferret exec --browser-headless my-script.fql
```

### Query parameters

```bash
ferret exec -p 'foo:"bar"' -p 'qaz:"baz"' my-script.fql
```

### With remote runtime (worker)
```bash
ferret exec --runtime 'https://my-worker.com' my-script.fql
```

## Options

```bash
Usage:
  ferret [flags]
  ferret [command]

Available Commands:
  browser     Manage Ferret browsers
  config      Manage Ferret configs
  exec        Execute a FQL script or launch REPL
  help        Help about any command
  version     Show the CLI version information

Flags:
  -h, --help               help for ferret
  -l, --log-level string   Set the logging level ("debug"|"info"|"warn"|"error"|"fatal") (default "info")

Use "ferret [command] --help" for more information about a command.

```