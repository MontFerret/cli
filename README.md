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
	<a href="https://opensource.org/licenses/Apache-2.0">
		<img alt="Apache-2.0 License" src="http://img.shields.io/badge/license-Apache-brightgreen.svg">
	</a>
</p>

<p align="center">
<img alt="lab" src="https://raw.githubusercontent.com/MontFerret/cli/master/assets/logo.svg" style="margin-left: auto; margin-right: auto;" width="450px" height="430px" />
</p>

---

> **📢 Notice:** This branch contains the upcoming **CLI for Ferret v2**. For the stable v1 release, please visit [CLI v1](https://github.com/MontFerret/cli/tree/v1).

---

## About Ferret CLI

Ferret CLI is a command-line interface for the [Ferret](https://github.com/MontFerret/ferret) web scraping system. Ferret uses its own query language called **FQL (Ferret Query Language)** - a SQL-like language designed specifically for web scraping, browser automation, and data extraction tasks.

## Table of Contents

- [About Ferret CLI](#about-ferret-cli)
- [What is FQL?](#what-is-fql)
- [Key Features](#key-features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Commands Overview](#commands-overview)
- [Configuration](#configuration)
- [Browser Management](#browser-management)
- [Advanced Usage](#advanced-usage)
- [Examples](#examples)
- [Troubleshooting](#troubleshooting)
- [Development](#development)
- [Contributors](#contributors)

### What is FQL?

FQL (Ferret Query Language) is a declarative language that combines the familiar syntax of SQL with powerful web automation capabilities. It allows you to:

- Navigate web pages and interact with elements
- Extract data from HTML documents
- Handle dynamic content and JavaScript-heavy sites  
- Manage browser sessions and cookies
- Perform complex data transformations
- Execute parallel scraping operations

### Key Features

- 🚀 **Fast and Efficient**: Built-in concurrency and optimized execution
- 🌐 **Browser Automation**: Full Chrome/Chromium browser control
- 🔄 **Dynamic Content**: Handle SPAs and JavaScript-heavy sites
- 📊 **Data Processing**: Built-in functions for data manipulation
- 🛠️ **Flexible Runtime**: Run locally or on remote workers
- 💾 **Session Management**: Persistent cookies and browser state
- 🔧 **Configuration**: Extensive customization options

Documentation is available [at our website](https://www.montferret.dev/docs/introduction/).

## Installation

### Binary

You can download the latest binaries from [here](https://github.com/MontFerret/cli/releases).

### Source (Go >= 1.18)
```bash
go install github.com/MontFerret/cli/ferret@latest
```

### Shell
```shell
curl https://raw.githubusercontent.com/MontFerret/cli/master/install.sh | sh
```

## Quick Start

### Your First FQL Query

The simplest way to get started is with the interactive REPL:

```bash
ferret repl
Welcome to Ferret REPL

Please use `exit` or `Ctrl-D` to exit this program.
>>> RETURN "Hello, Ferret!"
"Hello, Ferret!"
```

### Basic Web Scraping

Create a simple script (`example.fql`) to scrape a webpage:

```fql
// Navigate to a website and extract data
LET page = DOCUMENT("https://news.ycombinator.com")

FOR item IN ELEMENTS(page, ".submission")
    LET title = ELEMENT(item, ".title")
    RETURN {
        title: title.innerText,
        url: title.href
    }
```

Run the script:

```bash
ferret run example.fql
```

> **Note:** `exec` is an alias for `run` — both work interchangeably.

### Browser Automation

For JavaScript-heavy sites, use browser automation:

```bash
# Open browser window for debugging
ferret run --browser-open my-script.fql

# Run headlessly for production
ferret run --browser-headless my-script.fql
```

Example browser automation script:

```fql
// Browser automation example
LET page = DOCUMENT("https://example.com", { driver: "cdp" })
CLICK(page, "#search-button")
WAIT_ELEMENT(page, "#results")
RETURN ELEMENTS(page, ".result-item")
```

### Query Parameters

Pass dynamic values to your scripts:

```bash
ferret run -p 'url:"https://example.com"' -p 'limit:10' my-script.fql
```

Use parameters in your FQL script:

```fql
LET page = DOCUMENT(@url)  // Use the url parameter
LET items = ELEMENTS(page, ".item")
RETURN items
```

### Inline Evaluation

Run a quick FQL expression without a file:

```bash
ferret run --eval 'RETURN 2 + 2'
```

### Remote Runtime

Execute scripts on remote Ferret workers:

```bash
ferret run --runtime 'https://my-worker.com' my-script.fql
```

## Commands Overview

```
Usage:
  ferret [flags]
  ferret [command]

Available Commands:
  browser     Manage Ferret browsers
  build       Compile FQL scripts into bytecode artifacts
  check       Check FQL scripts for syntax and semantic errors
  config      Manage Ferret configs
  fmt         Format FQL scripts
  inspect     Compile and disassemble a FQL script
  repl        Launch interactive FQL shell
  run         Run a FQL script (alias: exec)
  update      Update Ferret CLI
  version     Show the CLI version information

Flags:
  -h, --help               help for ferret
  -l, --log-level string   Set the logging level ("debug"|"info"|"warn"|"error"|"fatal") (default "info")

Use "ferret [command] --help" for more information about a command.
```

### run / exec

Run a FQL script, a compiled artifact file, or an inline expression. To launch the interactive REPL, use the `ferret repl` command.

```bash
ferret run [script]
ferret exec [script]   # alias
```

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--runtime` | `-r` | Runtime type (`"builtin"` or a remote worker URL) | `builtin` |
| `--proxy` | `-x` | Proxy server address | |
| `--user-agent` | `-a` | User-Agent header | |
| `--browser-address` | `-d` | CDP debugger address | `http://127.0.0.1:9222` |
| `--browser-open` | `-B` | Open a visible browser for execution | `false` |
| `--browser-headless` | `-b` | Open a headless browser for execution | `false` |
| `--browser-cookies` | `-c` | Keep cookies between queries | `false` |
| `--param` | `-p` | Query parameter (`key:value`, repeatable) | |
| `--eval` | `-e` | Inline FQL expression (cannot be used with file args) | |

Compiled artifacts are auto-detected by content, so files produced by `ferret build` work even when they do not use a `.fqlc` filename. Artifact execution currently requires the builtin runtime.

### repl

Launch the interactive FQL shell. Supports command history, multiline input (toggle with `%`), and all runtime flags.

```bash
ferret repl
```

Accepts the same runtime and `--param` flags as `run` (everything except `--eval`).

### check

Compile one or more FQL scripts without executing them. Reports syntax and semantic errors.

```bash
ferret check [files...]
```

### build

Compile one or more FQL scripts into serialized bytecode artifacts.

```bash
ferret build [files...]
```

| Flag | Short | Description |
|------|-------|-------------|
| `--output` | `-o` | Output file path for a single input, or output directory for multiple inputs |

Without `--output`, each input writes a sibling artifact with the same base name and a `.fqlc` extension.

### fmt

Format FQL scripts. By default, files are overwritten in place.

```bash
ferret fmt [files...]
```

| Flag | Description | Default |
|------|-------------|---------|
| `--dry-run` | Print formatted output to stdout instead of overwriting | `false` |
| `--print-width` | Maximum line length | `80` |
| `--tab-width` | Indentation size | `4` |
| `--single-quote` | Use single quotes instead of double quotes | `false` |
| `--bracket-spacing` | Add spaces inside brackets | `true` |
| `--case-mode` | Keyword case: `upper`, `lower`, or `ignore` | `upper` |

### inspect

Compile a FQL script and display its disassembled bytecode. Useful for debugging and understanding script compilation.

```bash
ferret inspect [script]
```

| Flag | Description |
|------|-------------|
| `--eval` / `-e` | Inline FQL expression |
| `--bytecode` | Show only bytecode instructions |
| `--constants` | Show only the constant pool |
| `--functions` | Show only function definitions |
| `--summary` | Show a high-level program summary |
| `--spans` | Show debug source spans |

When no filter flags are provided, the full disassembly is printed.

## Configuration

Ferret CLI can be configured using the `config` command or configuration files.

### Setting Configuration Values

```bash
# Set the CDP browser address
ferret config set browser-address "http://localhost:9222"

# Set user agent
ferret config set user-agent "MyBot 1.0"

# Set default runtime
ferret config set runtime "builtin"
```

### Viewing Configuration

```bash
# List all configuration values
ferret config list

# Get a specific value
ferret config get browser-address
```

### Configuration File Locations

Configuration files are stored in:
- **Linux/macOS**: `~/.config/ferret/config.yaml`
- **Windows**: `%APPDATA%\ferret\config.yaml`

### Configuration Priority

Values are resolved in this order (highest to lowest):

1. Command-line flags
2. Environment variables (prefixed with `FERRET_`, e.g. `FERRET_RUNTIME`)
3. Configuration file
4. Defaults

### Available Configuration Options

| Key | Description | Default |
|-----|-------------|---------|
| `log-level` | Logging level | `info` |
| `runtime` | Runtime type (`builtin` or a remote URL) | `builtin` |
| `browser-address` | Chrome DevTools Protocol address | `http://127.0.0.1:9222` |
| `browser-cookies` | Keep cookies between queries | `false` |
| `browser-open` | Open a visible browser for execution | `false` |
| `browser-headless` | Open a headless browser for execution | `false` |
| `proxy` | Proxy server address | |
| `user-agent` | Custom User-Agent header | |

## Browser Management

### Starting a Browser Instance

```bash
# Open a new browser instance
ferret browser open

# Open in headless mode
ferret browser open --headless

# Open on a custom debugging port
ferret browser open --port 9223

# Start in background and print the process ID
ferret browser open --detach

# Specify a custom user data directory
ferret browser open --user-dir /tmp/ferret-profile
```

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--detach` | `-d` | Start in background, print PID | `false` |
| `--headless` | | Launch in headless mode | `false` |
| `--port` | `-p` | Remote debugging port | `9222` |
| `--user-dir` | | Browser user data directory | |

### Closing Browser Instances

```bash
# Close the default browser
ferret browser close

# Close a specific browser by PID
ferret browser close 12345
```

## Advanced Usage

### Complex Data Extraction

```fql
// E-commerce product scraping with error handling
LET page = DOCUMENT("https://shop.example.com/products")
LET products = (
    FOR product IN ELEMENTS(page, ".product-card")
        LET name = ELEMENT(product, ".product-name")
        LET price = ELEMENT(product, ".price")
        LET image = ELEMENT(product, ".product-image")
        
        // Handle missing elements gracefully
        RETURN name != NONE ? {
            name: TRIM(name.innerText),
            price: REGEX_MATCH(price.innerText, /\$[\d.]+/)[0],
            image: image.src,
            url: CONCAT("https://shop.example.com", product.href)
        } : NONE
)
// Filter out null results  
LET validProducts = (
    FOR product IN products
        FILTER product != NONE
        RETURN product
)
RETURN validProducts
```

### Working with Forms

```fql
// Login form automation
LET page = DOCUMENT("https://example.com/login", { driver: "cdp" })

// Fill in form fields
INPUT(page, "#username", "myuser")
INPUT(page, "#password", "mypassword")  

// Submit form and wait for navigation
CLICK(page, "#login-button")
WAIT_NAVIGATION(page)

// Extract user data after login
RETURN {
    loggedIn: ELEMENT(page, ".user-menu") != NONE,
    username: ELEMENT(page, ".username").innerText
}
```

### Parallel Processing

```fql
// Scrape multiple pages in parallel
LET urls = [
    "https://news.ycombinator.com",
    "https://reddit.com/r/programming", 
    "https://dev.to"
]

LET results = (
    FOR url IN urls
        LET page = DOCUMENT(url)
        RETURN {
            url: url,
            title: ELEMENT(page, "title").innerText,
            headlines: (
                FOR headline IN ELEMENTS(page, "h1, h2, h3")
                RETURN headline.innerText
            )
        }
)

RETURN results
```

### Working with APIs

```fql
// Combine web scraping with API calls
LET page = DOCUMENT("https://github.com/trending")
LET repos = ELEMENTS(page, ".Box-row")

LET details = (
    FOR repo IN repos[0:5]
        LET repoName = ELEMENT(repo, "h1 a").innerText
        LET apiUrl = CONCAT("https://api.github.com/repos/", repoName)
        
        // Make API call
        LET apiData = DOCUMENT(apiUrl, { driver: "http" })
        
        RETURN {
            name: repoName,
            description: ELEMENT(repo, "p").innerText,
            stars: apiData.stargazers_count,
            language: apiData.language
        }
)

RETURN details
```

## Examples

### Web Scraping Examples

<details>
<summary>📊 Extract table data</summary>

```fql
// Extract data from HTML tables
LET page = DOCUMENT("https://example.com/data-table")
LET table = ELEMENT(page, "table")
LET headers = (
    FOR header IN ELEMENTS(table, "thead th")
    RETURN header.innerText
)
LET rows = ELEMENTS(table, "tbody tr")

LET data = (
    FOR row IN rows
        LET cells = (
            FOR cell IN ELEMENTS(row, "td")
            RETURN cell.innerText
        )
        LET record = {}
        
        FOR i IN RANGE(0, LENGTH(headers))
            SET_KEY(record, headers[i], cells[i])
        
        RETURN record
)

RETURN data
```
</details>



<details>
<summary>📱 Mobile viewport simulation</summary>

```fql
// Test mobile-responsive sites
LET page = DOCUMENT("https://example.com", {
    driver: "cdp",
    viewport: {
        width: 375,
        height: 667,
        mobile: true
    },
    userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 14_7_1 like Mac OS X)"
})

// Check mobile-specific elements
LET mobileMenu = ELEMENT(page, ".mobile-menu")
LET desktopMenu = ELEMENT(page, ".desktop-menu")

RETURN {
    isMobile: mobileMenu != NONE,
    isDesktop: desktopMenu != NONE,
    viewport: {
        width: page.viewport.width,
        height: page.viewport.height
    }
}
```
</details>

## Troubleshooting

### Common Issues

**Browser connection failed**
```bash
# Check if Chrome is running with remote debugging
google-chrome --remote-debugging-port=9222

# Or use Ferret's browser management
ferret browser open
```

**Script execution timeout**
```fql
// Increase timeouts for slow pages
LET page = DOCUMENT("https://slow-site.com", {
    driver: "cdp", 
    timeout: 30000  // 30 seconds
})
```

**Element not found errors**
```fql
// Use WAIT_ELEMENT for dynamic content
LET page = DOCUMENT("https://spa.example.com", { driver: "cdp" })
WAIT_ELEMENT(page, "#dynamic-content", 10000)
LET element = ELEMENT(page, "#dynamic-content")
```

**Memory issues with large datasets**
```fql
// Process data in chunks using supported syntax
LET items = ELEMENTS(page, ".item")
LET batchSize = 100

FOR i IN RANGE(0, LENGTH(items), batchSize)
    FOR item IN items
        // Process individual items...
        RETURN item.innerText
```

### Debug Mode

Enable debug logging for troubleshooting:

```bash
ferret run --log-level debug my-script.fql
```

### Performance Tips

1. **Use CSS selectors efficiently**: Specific selectors are faster than broad ones
2. **Minimize DOM queries**: Store elements in variables when reusing
3. **Use headless mode**: `--browser-headless` is faster for production
4. **Implement timeouts**: Always set appropriate timeouts for reliability
5. **Handle errors gracefully**: Use conditional logic to handle missing elements

## Development

### Building from Source

```bash
# Clone the repository
git clone https://github.com/MontFerret/cli.git
cd cli

# Install dependencies
go mod download

# Build the binary
make compile

# Run tests
make test
```

### Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b my-new-feature`
3. Make your changes and add tests
4. Run the test suite: `make test`
5. Submit a pull request

### Development Commands

```bash
# Install development tools
make install-tools

# Format code
make fmt

# Run linters
make lint

# Run all checks
make build
```

## Contributors
<a href="https://github.com/MontFerret/ferret/graphs/contributors">
  <img src="https://contrib.rocks/image?repo=MontFerret/cli" />
</a>
