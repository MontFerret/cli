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

## About Ferret CLI

Ferret CLI is a command-line interface for the [Ferret](https://github.com/MontFerret/ferret) web scraping system. Ferret uses its own query language called **FQL (Ferret Query Language)** - a SQL-like language designed specifically for web scraping, browser automation, and data extraction tasks.

## Table of Contents

- [About Ferret CLI](#about-ferret-cli)
- [What is FQL?](#what-is-fql)
- [Key Features](#key-features)
- [Installation](#installation)
- [Quick Start](#quick-start)
- [Options](#options)
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

## Quick start

### Your First FQL Query

The simplest way to get started is with the interactive REPL:

```bash
ferret exec
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
LET items = (
    FOR item IN ELEMENTS(page, ".athing")
        LET title = ELEMENT(item, ".storylink")
        RETURN {
            title: title.innerText,
            url: title.href
        }
)
RETURN items[0:5]  // Return first 5 items
```

Run the script:

```bash
ferret exec example.fql
```

### Script execution
```bash
ferret exec my-script.fql
```

### Browser Automation

For JavaScript-heavy sites, use browser automation:

```bash
# Open browser window for debugging
ferret exec --browser-open my-script.fql

# Run headlessly for production
ferret exec --browser-headless my-script.fql
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
ferret exec -p 'url:"https://example.com"' -p 'limit:10' my-script.fql
```

Use parameters in your FQL script:

```fql
LET page = DOCUMENT(@url)  // Use the url parameter
LET items = ELEMENTS(page, ".item")
RETURN items[0:@limit]     // Use the limit parameter
```

### Remote Runtime

Execute scripts on remote Ferret workers:

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
  update      Update Ferret CLI
  version     Show the CLI version information

Flags:
  -h, --help               help for ferret
  -l, --log-level string   Set the logging level ("debug"|"info"|"warn"|"error"|"fatal") (default "info")

Use "ferret [command] --help" for more information about a command.

```

## Configuration

Ferret CLI can be configured using the `config` command or configuration files.

### Setting Configuration Values

```bash
# Set a global configuration value
ferret config set browser.address "http://localhost:9222"

# Set user agent
ferret config set browser.userAgent "MyBot 1.0"

# Set default runtime
ferret config set runtime.type "builtin"
```

### Viewing Configuration

```bash
# List all configuration values
ferret config list

# Get a specific value  
ferret config get browser.address
```

### Configuration File Locations

Configuration files are stored in:
- **Linux/macOS**: `~/.config/ferret/config.yaml`
- **Windows**: `%APPDATA%\ferret\config.yaml`

### Available Configuration Options

| Key | Description | Default |
|-----|-------------|---------|
| `browser.address` | Chrome DevTools Protocol address | `http://127.0.0.1:9222` |
| `browser.userAgent` | Default User-Agent header | System default |
| `browser.cookies` | Keep cookies between queries | `false` |
| `runtime.type` | Runtime type (builtin/url) | `builtin` |
| `log.level` | Logging level | `info` |

## Browser Management  

### Starting a Browser Instance

```bash
# Open a new browser instance
ferret browser open

# Open with specific debugging address
ferret browser open --address "http://localhost:9223"
```

### Closing Browser Instances

```bash
# Close browser
ferret browser close

# Close specific browser by address
ferret browser close --address "http://localhost:9223"
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
RETURN products[* FILTER CURRENT != NONE]
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
            headlines: ELEMENTS(page, "h1, h2, h3")[*].innerText
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
LET headers = ELEMENTS(table, "thead th")[*].innerText
LET rows = ELEMENTS(table, "tbody tr")

LET data = (
    FOR row IN rows
        LET cells = ELEMENTS(row, "td")[*].innerText
        LET record = {}
        
        FOR i, header IN ENUMERATE(headers)
            LET record[header] = cells[i]
        
        RETURN record
)

RETURN data
```
</details>

<details>
<summary>🔍 Search and pagination</summary>

```fql
// Handle paginated search results
LET page = DOCUMENT("https://example.com/search", { driver: "cdp" })

// Perform search
INPUT(page, "#search-input", @searchTerm)
CLICK(page, "#search-button")
WAIT_ELEMENT(page, ".search-results")

LET allResults = []
LET currentPage = 1
LET maxPages = 5

WHILE currentPage <= maxPages
    // Extract results from current page
    LET results = ELEMENTS(page, ".search-result")
    LET pageData = (
        FOR result IN results
            RETURN {
                title: ELEMENT(result, ".title").innerText,
                url: ELEMENT(result, ".link").href,
                snippet: ELEMENT(result, ".snippet").innerText
            }
    )
    
    LET allResults = APPEND(allResults, pageData, true)
    
    // Try to go to next page
    LET nextButton = ELEMENT(page, ".next-page")
    IF nextButton != NONE
        CLICK(page, ".next-page")
        WAIT(2000)  // Wait for page load
        LET currentPage = currentPage + 1
    ELSE
        BREAK
    END
END

RETURN FLATTEN(allResults)
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
// Process data in chunks
LET items = ELEMENTS(page, ".item")
LET batchSize = 100

FOR batch IN RANGE(0, LENGTH(items), batchSize)
    LET chunk = items[batch:batch+batchSize]
    // Process chunk...
END
```

### Debug Mode

Enable debug logging for troubleshooting:

```bash
ferret exec --log-level debug my-script.fql
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
