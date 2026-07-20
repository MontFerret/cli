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

## HTTP policy

Ferret's builtin runtime blocks localhost, loopback, private-network, and link-local HTTP access by default. Grant only the access a script needs; for example, a script that intentionally calls a local development service requires an explicit opt-in:

```bash
ferret run \
  --policy-http-allow-localhost \
  --policy-http-default-headers='{"X-Trace":"local"}' \
  script.fql
```

HTTP policy options are available on `run`, `repl`, and `debug` and apply only to the builtin runtime. Supplying one with a remote runtime is a configuration error. They configure Ferret HTTP integrations such as `IO::NET::HTTP` and `NET::REST`; the existing `--proxy` and `--user-agent` options continue to configure HTML/browser drivers.

List values accept repeated flags or comma-separated values. Default headers use a JSON object with string values. Only values explicitly supplied through a flag, environment variable, or config file override Ferret's secure defaults. Numeric zero retains the Ferret default; use the dedicated `no-timeout` or `unlimited-*` option to disable a limit.

| Flag and config key | Environment variable | Default | Behavior |
| --- | --- | --- | --- |
| `policy-http-allowed-schemes` | `FERRET_POLICY_HTTP_ALLOWED_SCHEMES` | `http,https` | Allowed URL schemes |
| `policy-http-allowed-methods` | `FERRET_POLICY_HTTP_ALLOWED_METHODS` | `GET,HEAD,POST,PUT,PATCH,DELETE,OPTIONS` | Allowed HTTP methods |
| `policy-http-allowed-hosts` | `FERRET_POLICY_HTTP_ALLOWED_HOSTS` | unrestricted | Exact allowed hosts or `host:port` values |
| `policy-http-blocked-hosts` | `FERRET_POLICY_HTTP_BLOCKED_HOSTS` | none | Exact blocked hosts or `host:port` values |
| `policy-http-allow-localhost` | `FERRET_POLICY_HTTP_ALLOW_LOCALHOST` | `false` | Allow localhost and loopback addresses |
| `policy-http-allow-private-networks` | `FERRET_POLICY_HTTP_ALLOW_PRIVATE_NETWORKS` | `false` | Allow private-network addresses |
| `policy-http-allow-link-local` | `FERRET_POLICY_HTTP_ALLOW_LINK_LOCAL` | `false` | Allow link-local addresses |
| `policy-http-default-headers` | `FERRET_POLICY_HTTP_DEFAULT_HEADERS` | none | Default request headers as a JSON string map |
| `policy-http-blocked-request-headers` | `FERRET_POLICY_HTTP_BLOCKED_REQUEST_HEADERS` | none | Block requests containing these header names |
| `policy-http-timeout` | `FERRET_POLICY_HTTP_TIMEOUT` | `30s` | Overall HTTP timeout |
| `policy-http-no-timeout` | `FERRET_POLICY_HTTP_NO_TIMEOUT` | `false` | Explicitly disable the overall timeout |
| `policy-http-max-request-size` | `FERRET_POLICY_HTTP_MAX_REQUEST_SIZE` | `16777216` | Maximum request-body size in bytes |
| `policy-http-unlimited-request-size` | `FERRET_POLICY_HTTP_UNLIMITED_REQUEST_SIZE` | `false` | Explicitly disable the request-body limit |
| `policy-http-max-response-size` | `FERRET_POLICY_HTTP_MAX_RESPONSE_SIZE` | `16777216` | Maximum response-body size in bytes |
| `policy-http-unlimited-response-size` | `FERRET_POLICY_HTTP_UNLIMITED_RESPONSE_SIZE` | `false` | Explicitly disable the response-body limit |
| `policy-http-max-response-header-size` | `FERRET_POLICY_HTTP_MAX_RESPONSE_HEADER_SIZE` | `1048576` | Maximum response-header size in bytes |
| `policy-http-follow-redirects` | `FERRET_POLICY_HTTP_FOLLOW_REDIRECTS` | `true` | Follow HTTP redirects |
| `policy-http-max-redirects` | `FERRET_POLICY_HTTP_MAX_REDIRECTS` | `10` | Maximum redirects to follow |

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
ferret config set policy-http-allow-localhost true
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
