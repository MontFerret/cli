# AGENTS.md

This file is the canonical operating guide for coding agents working in the Ferret CLI repository. It is written for the Ferret v2 CLI only. If repository documentation conflicts with this file, prefer `go.mod`, `Makefile`, `.github/workflows/*`, and `.goreleaser.yml` for toolchain, commands, CI, and release behavior.

## Repo snapshot

- Module path: `github.com/MontFerret/cli/v2`
- Go version in `go.mod`: `1.26.5`
- Binary entrypoint: `ferret/main.go`
- Built binary name: `ferret`
- Core Ferret dependency: `github.com/MontFerret/ferret/v2`
- Registry and publication dependency: `github.com/MontFerret/barn`
- Portable module manifest dependency: `github.com/MontFerret/specs`
- This repository root is the Ferret v2 CLI. Do not mix assumptions from the separate v1 branch.

## Architectural mental model

The CLI turns user intent into Ferret core execution or module-lifecycle workflows.

Primary command flow:

```text
user command -> ferret composition root -> cmd facade -> cmd/internal/<feature> -> pkg service -> output
```

Execution flow:

```text
run/debug/build/check/fmt -> source/options -> runtime/browser/config helpers -> Ferret core API -> diagnostics/results
```

Module-lifecycle flow:

```text
ferret mod -> cmd/internal/mod -> pkg/module -> discovery/install/scaffold/publish -> Barn/Specs/Go/GitHub/filesystem
```

Agents should reason about changes by ownership boundary:

- `ferret/main.go` is the composition root. It creates shared dependencies, assembles the root Cobra command, and owns process-level cancellation and exit behavior.
- `cmd` is a thin public constructor facade. Command implementations and their tests belong in feature-scoped packages under `cmd/internal`.
- Shared execution flags, policies, parameter parsing, and runtime option assembly belong in `cmd/internal/execution` when more than one command consumes them.
- Source resolution belongs in `pkg/source` or `pkg/run`, depending on whether the behavior is input identity/loading or execution preparation.
- Builtin and remote runtime adapters belong in `pkg/runtime`; browser process management belongs in `pkg/browser`.
- Interactive debugger behavior belongs in `pkg/debugger`; command wiring belongs in `cmd/internal/debug`; Ferret core debug-session creation belongs in `pkg/runtime/debug.go`.
- Interactive FQL REPL behavior belongs in `pkg/repl`; command wiring belongs in `cmd/internal/repl`.
- Build artifact planning and paths belong in `pkg/build`; command wiring belongs in `cmd/internal/build` and inspection rendering belongs in `cmd/internal/inspect`.
- Module command UX belongs in `cmd/internal/mod`; reusable workflows belong under `pkg/module`.
- Specs owns portable module manifest schemas and validation. Barn owns Registry APIs and publication preparation. The CLI orchestrates those APIs and owns CLI-specific project mutation, rendering, and GitHub submission.
- Configuration, logging, self-update, and release behavior remain in their owning packages and release files.

The CLI should adapt owning APIs rather than reimplement Ferret language semantics, Specs validation, or Barn Registry preparation.

## Canonical invariants

- The CLI does not own FQL parsing, formatting semantics, compilation, bytecode, VM behavior, optimizer behavior, or core runtime value semantics.
- Commands, flags, aliases, config keys, exit behavior, stdout/stderr intent, and machine-readable output are user-facing API.
- The public `cmd` package remains a thin constructor facade; feature behavior belongs in `cmd/internal/<feature>` or the owning `pkg/*` package.
- Builtin and remote runtime behavior remain explicit and separate. Never silently fall back between them.
- Runtime execution preserves Ferret core behavior instead of compensating for it in CLI-only code.
- Resources started or owned by the CLI are cleaned up deterministically on success, failure, and cancellation where possible.
- Project-mutating module workflows stage and validate changes before committing them and must avoid leaving partial state.
- Module manifest and publication validation reuse Specs and Barn APIs rather than local copies of their rules.
- Machine-readable modes emit only their documented machine output on stdout.
- Registry publication preserves immutable source records and retry safety.
- Do not assume behavior from old design notes, stale documentation, or the v1 CLI unless it is reflected in the current v2 code and tests.

## Package map

Agents should begin with the package whose responsibility owns the requested behavior. Do not infer ownership from historical file names when this map identifies the current boundary.

### Command composition and feature wiring

* `ferret`
    * Owns the binary entrypoint, process-level dependency assembly, signal cancellation, root command, and version injection target.
    * Keep it focused on composition. Do not implement feature workflows here.

* `cmd`
    * Owns the exported command-constructor facade used by the entrypoint and external command-level tests.
    * Keep constructors thin delegations to `cmd/internal/<feature>`.
    * Add facade coverage when constructor wiring or dependency injection changes.

* `cmd/internal/<feature>`
    * Owns Cobra definitions, command hierarchy, flags, aliases, argument validation, help, feature orchestration, and command-specific rendering.
    * Existing feature packages include browser, build, check, config, debug, format, inspect, mod, repl, run, update, and version.
    * Keep callbacks thin: parse CLI intent, validate it, assemble options, call the owning package, and render the result.
    * Put command tests beside the owning feature package rather than growing the public facade package.

* `cmd/internal/execution`
    * Owns execution-related flags and conversion from command/config values to shared source, filesystem, HTTP, parameter, browser, and runtime options.
    * Reuse it across run-like commands instead of duplicating flag parsing or policy construction.

* `cmd/internal/diagnostics`
    * Owns command-facing diagnostic rendering shared by feature commands.
    * Preserve Ferret diagnostic detail and source context rather than replacing them with generic CLI errors.

* `cmd/internal/testutil`
    * Owns test-only command helpers.
    * Do not move production behavior here or expose it as a public API for test convenience.

### Execution and source handling

* `pkg/run`
    * Owns shared run/execute flows after command intent is known.
    * Coordinates source resolution and runtime execution without owning browser process internals or Ferret semantics.

* `pkg/runtime`
    * Owns CLI runtime selection and adapters for builtin, remote, and debug execution.
    * Converts CLI runtime options into Ferret core execution or debug sessions.
    * Keep builtin and remote behavior explicitly separated and propagate context cancellation.
    * Do not implement core Ferret semantics here.

* `pkg/source`
    * Owns source resolution from files, inline input, and other supported CLI source modes.
    * Preserve source identity so Ferret diagnostics and debugger output retain useful paths and locations.

### Browser and environment support

* `pkg/browser`
    * Owns browser discovery, startup, readiness, lifecycle cleanup, and browser-specific flags/options.
    * Keep platform-specific behavior in platform-specific files.
    * Start browsers lazily and only for modes that require browser support.

* `pkg/config`
    * Owns config store behavior, context, initialization, flag binding, filesystem policy, and persistent configuration.
    * Preserve key names and storage semantics unless the task explicitly changes them.
    * Do not mix feature-specific validation into generic config storage.

* `pkg/logger`
    * Owns CLI logging options and logger construction.
    * Logging remains observational; command semantics and control flow must not depend on log output.

### Developer tooling commands

* `pkg/debugger`
    * Owns the interactive debugger REPL, command parsing, dispatch, prompt lifecycle, rendering, and CLI-facing debug-session interface.
    * Consumes Ferret core debug sessions through a small interface seam.
    * Does not own VM semantics, breakpoint binding semantics, runtime value semantics, or DAP behavior.

* `pkg/repl`
    * Owns the interactive FQL shell loop and output behavior.
    * Preserve cancellation, exit, and input behavior carefully because this package is interactive.

* `pkg/build`
    * Owns build artifact planning, output paths, and compiled artifact decisions.
    * Uses Ferret core for compilation and bytecode semantics.

### Module lifecycle

* `pkg/module`
    * Owns the service facade that coordinates discovery, installation, scaffolding, and publication.
    * Keep it as orchestration over narrow workflow interfaces; do not absorb the subpackages' business logic.

* `pkg/module/discovery`
    * Owns CLI discovery projections over Barn's public Registry client.
    * Preserve canonical Registry identity and search behavior across module IDs and descriptions.
    * Do not recreate Barn artifact parsing or validation locally.

* `pkg/module/install`
    * Owns installation into an existing Go application: project discovery, compatible Registry release resolution, composition rewriting, Go toolchain staging, validation, and transactional commit.
    * It may mutate only the project files approved by the install workflow and must restore prior state on commit failure.
    * Module code is installed into the target application; it does not mutate the CLI runtime.

* `pkg/module/scaffold`
    * Owns creation of a new module project from resolved CLI input.
    * Generate in a staging directory, validate the manifest with Specs, and install the scaffold atomically.
    * Do not download or resolve dependencies as a side effect of scaffolding.

* `pkg/module/publish`
    * Owns CLI publication modes and adaptation to Barn's public `pkg/publish.Prepare` API.
    * Barn remains authoritative for manifest, source, tag, README, `go.mod`, identity, and immutability validation.
    * Keep the prepared record model provider-neutral; provider-specific submission belongs below it.

* `pkg/module/publish/github`
    * Owns GitHub authentication, REST Git Data operations, personal-fork recovery, branch reconciliation, and pull-request submission for prepared Barn records.
    * It must not clone Barn, shell out to Git, expose credentials, modify generated `dist/`, or add `publishedAt`.
    * Reuse exact existing publication state and fail explicitly on conflicts or divergent content rather than overwriting it.

### Maintenance and release support

* `pkg/selfupdate`
    * Owns update checks and CLI self-update behavior.
    * Network boundaries must be explicit and errors actionable.
    * Do not replace binaries or installation state without clear command intent.

* `scripts`
    * Owns shell helpers used by the Makefile and release workflows.
    * Keep scripts portable where practical and avoid duplicating logic already owned by Go code or the Makefile.

* `.goreleaser.yml`
    * Owns release packaging behavior.
    * Treat changes here as release-sensitive.

## Where to start by task

- Add or change a command:
    - inspect the owning `cmd/internal/<feature>` package
    - inspect shared patterns in adjacent feature packages
    - place reusable behavior in the appropriate `pkg/*` package
    - update `cmd/commands.go` only when public constructor wiring changes
    - add tests in the owning feature package, plus facade tests when wiring changes

- Add or change a flag:
    - inspect the owning feature package
    - use `cmd/internal/execution` for flags or option conversion shared by execution commands
    - use `pkg/config` only for persistent configuration behavior
    - preserve compatibility and update help and tests

- Change run behavior:
    - inspect `cmd/internal/run`
    - inspect `cmd/internal/execution`
    - inspect `pkg/run`, `pkg/runtime`, and `pkg/source`
    - inspect `pkg/browser` if startup or lifecycle is affected

- Change debug behavior:
    - inspect `cmd/internal/debug` for command validation and startup
    - inspect `pkg/runtime/debug.go` for Ferret core debug-session setup
    - inspect `pkg/debugger` for REPL commands, lifecycle, parsing, and rendering

- Change build or inspect behavior:
    - inspect `cmd/internal/build` or `cmd/internal/inspect`
    - inspect `pkg/build` for artifact planning and path behavior
    - rely on Ferret core for compilation, bytecode, and disassembly semantics

- Change browser behavior:
    - inspect `cmd/internal/browser`, `cmd/internal/execution`, and `pkg/browser`
    - test platform-neutral behavior where possible
    - isolate OS-specific changes in platform-specific files

- Change config behavior:
    - inspect `cmd/internal/config` and `pkg/config`
    - preserve existing key names and storage behavior unless explicitly changed

- Change formatting or checking behavior:
    - inspect `cmd/internal/format` or `cmd/internal/check`
    - call Ferret core formatter/checker APIs instead of reimplementing language logic
    - test CLI error and output shape

- Change module discovery:
    - inspect `cmd/internal/mod`, `pkg/module`, and `pkg/module/discovery`
    - inspect the current Barn public Registry API before changing projections or search behavior

- Change module installation:
    - inspect `cmd/internal/mod` approval behavior and `pkg/module/install`
    - preserve staging, owning-package build validation, idempotency, and rollback behavior
    - test missing dependencies, missing or ambiguous composition, compatible release selection, and commit recovery

- Change module scaffolding:
    - inspect `cmd/internal/mod` wizard behavior and `pkg/module/scaffold`
    - use Specs for manifest constants and validation
    - preserve non-interactive requirements and atomic destination creation

- Change module publication:
    - inspect `cmd/internal/mod`, `pkg/module/publish`, and `pkg/module/publish/github`
    - inspect the current Barn `pkg/publish` API and Specs manifest API
    - preserve non-mutating modes, deterministic output, idempotency, and immutable-record checks

- Change update or release behavior:
    - inspect `cmd/internal/update`, `pkg/selfupdate`, `scripts/release.sh`, `scripts/versions.sh`, `.goreleaser.yml`, the Makefile, and release workflows
    - treat the change as release-sensitive and validate the tag/version flow carefully

## Stability guide

Treat these as relatively stable unless the task explicitly targets them:

- command names, primary aliases, flags, and config keys
- command exit behavior and stdout/stderr intent
- machine-readable output schemas and deterministic ordering
- runtime selection model: builtin versus remote
- browser lifecycle expectations
- module manifest filename and Specs/Barn ownership boundaries
- transactional module install/scaffold expectations
- non-mutating publication modes and immutable publication history
- release binary name and version injection paths

Treat these as implementation-sensitive and verify current code before proposing changes:

- feature command composition under `cmd/internal`
- debugger REPL lifecycle and command dispatch
- browser startup, readiness, and cleanup
- remote runtime request/response behavior
- config file loading and persistence
- source and artifact path resolution
- module install staging and rollback
- publication reconciliation and GitHub recovery
- self-update and release scripts

Do not treat historical discussion, stale README text, old branches, or v1 behavior as authoritative.

## Public command and package boundary rules

- Treat commands, flags, aliases, config keys, exit statuses, stdout/stderr behavior, and documented machine output as CLI API.
- Do not remove or rename public behavior without explicit instruction. Prefer aliases only when compatibility is intended.
- Human-facing results normally go to stdout; diagnostics and errors normally go to stderr.
- Machine-facing modes must not mix progress text with their structured stdout payload.
- Keep command help concise, accurate, and useful in a terminal.
- Treat exported command constructors and exported workflow contracts as API-sensitive even when the CLI is their primary consumer.
- Do not export new symbols merely to share internals or simplify tests. Prefer unexported helpers in the owning package.
- Add meaningful doc comments to necessary exported contracts and describe their stability or lifecycle expectations.
- Call out every intentional backward-incompatible CLI or output change in the final summary.

## Ferret core boundary rules

- Do not reimplement FQL parsing, formatting semantics, compilation, bytecode handling, VM behavior, runtime values, or breakpoint binding in the CLI.
- If behavior belongs in Ferret core, change Ferret separately or adapt to its current public API.
- CLI adapters may add command context to core errors but must preserve specific diagnostics and useful source locations.
- Keep execution adapters thin around Ferret core APIs.
- When a core version bump changes behavior, update CLI tests, output expectations, and documentation to the new contract.
- Do not change FQL language semantics as a side effect of command or adapter refactoring.

## Module lifecycle rules

- Use `github.com/MontFerret/specs/pkg/module` for the canonical manifest filename, parsing, and portable validation.
- Use Barn's public Registry client for discovery and compatible release data; do not parse Registry distribution files directly in CLI code.
- Use Barn `pkg/publish.Prepare` as the authoritative publication-preparation pipeline; do not duplicate its cross-document or Git validation.
- Keep command prompting and rendering in `cmd/internal/mod`, workflow behavior in `pkg/module/*`, and provider transport in `pkg/module/publish/github`.
- Installation must stage dependency and source changes, validate the owning package build, and commit or roll back the complete change set.
- Scaffolding must validate staged output and fail without replacing an existing destination.
- `ferret mod publish --dry-run` and `--print` must not authenticate with or mutate GitHub.
- `--print` emits only its versioned deterministic JSON document on stdout, with stable record ordering and no progress text.
- Resolve GitHub credentials from `GH_TOKEN`, then `GITHUB_TOKEN`, then `gh auth token --hostname github.com`; never print or embed credentials in errors.
- Treat an exact already-published release as a successful no-op before provider authentication or mutation.
- Submit through an authenticated personal fork and focused Git Data operations. Do not require a local Barn checkout.
- Accept only the expected Barn-relative source records. Reject duplicates, unexpected paths, generated `dist/`, and `publishedAt`.
- Reuse exact open pull requests and retry branches; fail closed for immutable conflicts, divergent branches, or mismatched record sets.

## Resource and lifecycle rules

- Document ownership for processes, files, streams, sessions, temporary directories, and other resources that require cleanup.
- Cleanup must be deterministic where an API exposes `Close` or equivalent lifecycle behavior.
- Preserve cleanup on normal return, command error, cancellation, and validation failure where the CLI owns the resource.
- Do not start browsers, network clients, prompts, or provider authentication before the selected mode requires them.
- Propagate command contexts through runtime, browser, Registry, Go toolchain, and GitHub operations.
- Use staged and recoverable filesystem mutation for multi-file workflows; do not leave partial user-project changes after a failed commit.
- Preserve file modes and unrelated user content when rewriting project files.

## Debugger CLI rules

- `cmd/internal/debug` owns CLI argument/flag validation and high-level debug startup.
- `pkg/runtime/debug.go` owns creation of Ferret core debug sessions from CLI runtime options.
- `pkg/debugger` owns interactive command parsing, aliases, dispatch, prompt lifecycle, and terminal rendering.
- The debugger REPL depends on a small session interface, not command or runtime packages.
- Debug requires builtin runtime unless the task explicitly implements and tests another mode.
- Do not add daemon, DAP, or remote-debug assumptions to the local CLI debugger unless explicitly requested.
- Commands fail safely after completion or termination and must not call closed or completed sessions unnecessarily.
- Destructive commands such as breakpoint deletion must not repeat implicitly on empty input.
- Describe limited evaluation accurately when Ferret core exposes only limited evaluation behavior.
- Distinguish requested source locations from resolved breakpoint locations in messages and snippets.
- Debugger support must not add setup or allocation work to debugger-disabled execution paths without measurement and justification.

## Browser lifecycle rules

- Start a browser only when the selected command, runtime, and input require browser support.
- If the CLI starts a browser process, clean it up on normal completion, command error, and cancellation where possible.
- Keep platform-specific behavior in platform-specific files.
- Avoid global mutable browser state unless the package explicitly owns it and tests cover it.
- Readiness errors should name the attempted endpoint or mode and explain the next action.
- Attaching to an existing browser and starting a managed browser are distinct modes; do not silently switch between them.

## Runtime and remote execution rules

- Keep builtin execution local and direct.
- Keep remote execution explicit and network-aware.
- Never silently fall back between builtin and remote runtimes.
- Preserve JSON-aware parameter parsing, including strings that resemble JSON literals.
- Preserve cancellation and context propagation through runtime calls.
- Keep filesystem and HTTP policy construction explicit and shared through the owning execution/config layers.
- Do not make runtime behavior depend on log output.

## Error and diagnostic quality rules

- User-facing errors should be specific, actionable, and retain the best available command, flag, path, runtime, source, or workflow-stage context.
- Preserve wrapped errors where callers or tests rely on error identity.
- Do not replace specific Ferret, Specs, Barn, Go toolchain, or GitHub errors with generic CLI errors.
- Keep diagnostic source spans and labels accurate when core diagnostics provide them.
- For unsupported modes, name the supported alternative.
- For path errors, include the relevant path when safe and useful.
- For authentication errors, explain credential setup without exposing token contents.
- For transactional failures, state whether changes were committed, rolled back, or left untouched.
- Tests for diagnostic changes should verify category, stage, message context, and source location when applicable.

## Go type and file structure rules

These rules are mandatory unless the task explicitly requires otherwise.

- Do not define multiple method-bearing structs in the same `.go` file.
- Prefer declaring a method-bearing struct as a standalone `type Name struct { ... }`.
- A method-bearing struct should usually live in its own file, named after the primary type or responsibility whenever practical, for example:
    - `process.go` for `Process`
    - `installer.go` for `Installer`
    - `publisher.go` for `Publisher`
- Grouped `type ( ... )` declarations are allowed for interfaces, passive data-only structs, and small related helper/value types from one narrow concern.
- A grouped declaration may contain exactly one method-bearing struct when it is the only behavioral type in the file and the other types are passive helpers from the same concern.
- Do not use grouped declarations to hide multiple substantial behavioral types.
- If a helper gains methods and would create a second method-bearing struct in the file, extract it immediately.
- Keep methods with their struct unless there is a strong, explicit reason to split by concern.
- Do not place a new method-bearing struct in an existing file merely because it compiles.

Allowed:

```go
type (
	publicationRecord struct {
		Path    string
		Content string
	}

	Submitter interface {
		Submit(context.Context, *publish.Result) (*Submission, error)
	}
)
```

Avoid:

```go
type (
	Installer struct {
		// ...
	}

	installStage struct {
		// ...
	}
)
```

or 

```go
type Type1 struct {
    // ...
}

type Type2 struct {
    // ...
}
```

Rationale:

- One method-bearing type per file keeps behavioral ownership obvious.
- Standalone behavioral types make diffs and reviews clearer.
- Grouped declarations remain appropriate for passive, closely related contracts and values.

## Function and method ownership rules

These rules are mandatory unless the task explicitly requires otherwise.

- A file centered on a method-bearing type contains the type, its methods, and constructors only.
- Do not mix unrelated package-level helpers into a type-centered file.
- Constructors are the normally allowed package-level functions in type-centered files.
- If logic belongs to the primary type, implement it as a method.
- If logic is genuinely package-level, place it in a helper-focused file.
- Package-level functions are preferred only when there is no natural owning type.
- A file containing methods plus non-constructor package-level functions is usually a structure violation and should be refactored.

## Comment rules for functions and methods

- Do not comment every function or method by default.
- Exported functions and methods should usually have doc comments, especially for command-facing, package-facing, provider-facing, or lifecycle-sensitive contracts.
- Comment unexported functions only when they carry non-obvious invariants, side effects, security constraints, cleanup expectations, recovery behavior, or protocol semantics.
- Explain intent, contract, ownership, invariants, side effects, or lifecycle rather than restating the signature.
- For browser, runtime, module mutation, publication, authentication, and debugger code, prefer comments about safety and recovery semantics over implementation narration.
- Avoid comment wallpaper. Dense, meaningful comments are better than mechanical documentation.

Preferred:

```go
// Close releases resources associated with the browser process.
// It is safe to call multiple times.
func (p *Process) Close() error
```

Preferred for internal code:

```go
// commitInstallChanges restores every committed file if a later replacement fails.
func commitInstallChanges(changes []fileChange) error
```

Avoid:

```go
// Close closes the process.
func (p *Process) Close() error
```

## Go control-flow spacing rules

These rules are mandatory for handwritten Go code.

Blank lines should separate logical units and make control-flow boundaries visually obvious.

### Immediate producer + check

A declaration, assignment, function call, type assertion, lookup, parse operation, or similar statement may remain directly adjacent to a following `if` when the `if` immediately checks or consumes the value produced by that statement.

This includes error checks, boolean/result checks, type assertions, nil checks, bounds checks, and other immediate validation.

Preferred:

```go
res, err := doSome()
if err != nil {
	return err
}
```

Preferred:

```go
named, ok := typeOf.(*types.Named)
if !ok || named.Obj().Pkg() == nil || !w.localPackage(named.Obj().Pkg().Path()) {
	return w.source.errorAt(
		ErrorUnsupportedRegistration,
		expression.Pos(),
		"New selects a module root dynamically",
	)
}
```

Preferred:

```go
value := lookup(name)
if value == nil {
	return ErrNotFound
}
```

Preferred:

```go
count := len(items)
if count == 0 {
	return nil
}
```

The producer and its immediate check form one logical unit and should not be separated by a blank line.

### Separation from preceding logic

If an immediate producer + check unit follows another statement or logical unit, separate it from the preceding code with a blank line.

Preferred:

```go
prepareState()

named, ok := typeOf.(*types.Named)
if !ok {
	return ErrUnsupported
}
```

Avoid:

```go
prepareState()
named, ok := typeOf.(*types.Named)
if !ok {
	return ErrUnsupported
}
```

No leading blank line is required when the producer begins the enclosing block:

```go
func inspect(typeOf types.Type) error {
	named, ok := typeOf.(*types.Named)
	if !ok {
		return ErrUnsupported
	}

	return inspectNamed(named)
}
```

### Consecutive control-flow blocks

Separate independent `if` statements with a blank line.

Avoid:

```go
if foo != nil {
	useFoo(foo)
}
if bar != nil {
	useBar(bar)
}
```

Prefer:

```go
if foo != nil {
	useFoo(foo)
}

if bar != nil {
	useBar(bar)
}
```

This applies even when both conditions are short. Independent control-flow decisions should remain visually distinct.

### Statements after control flow

Add a blank line after a completed `if` block before continuing with a separate statement or logical unit.

Avoid:

```go
if foo == bar {
	doFoo()
}
doSomething()
```

Prefer:

```go
if foo == bar {
	doFoo()
}

doSomething()
```

## Response and code style

When assisting with this repository, avoid large unstructured blocks of prose or code.

Prefer responses that are easy to scan:

- Use short sections with clear headings.
- Use bullets for decisions, trade-offs, and follow-up work.
- Use code blocks only for code, commands, or configuration.
- Prefer focused snippets or diffs over full-file dumps.
- Explain why a change is needed before showing how to implement it.
- Keep code comments useful and minimal.
- Avoid repeating the same context.
- When a change touches multiple files, summarize each file's role first.

The expected tone is practical, concise, and engineering-focused.

## Development practice expectations

Agents must follow repository-specific engineering discipline rather than generic style preferences.

### Core principles

- Preserve correctness first.
- Preserve subsystem boundaries and invariants.
- Prefer the smallest local change that fully solves the task.
- Avoid abstractions, indirection, and refactors unless required for correctness, maintainability, or the requested design.
- Do not optimize by intuition; measure performance-sensitive work.
- Keep behavioral ownership obvious in code structure and naming.
- Distinguish product failures from sandbox, network, toolchain, and dependency failures.

### Mandatory expectations

- Identify the owning subsystem before making a non-trivial change.
- Identify the public or internal contract being preserved or changed.
- Preserve existing behavior unless the task explicitly changes it.
- Add or update tests for every behavior change.
- Add or update benchmarks for every significant performance-sensitive change.
- Run narrow validation first, then broaden in proportion to risk.
- Review the complete resulting diff after implementation and initial validation for every non-trivial task.
- Do not claim tests, benchmarks, or validation that were not actually run.
- Do not treat historical discussions, abandoned directions, v1 behavior, or old branches as authoritative.
- Do not perform unrelated opportunistic refactors unless required for correctness.

### Required workflow for non-trivial changes

Before implementation:

1. Identify the owning subsystem.
2. Identify the contract, invariant, or behavior being preserved or changed.
3. Choose the smallest implementation that fits the current design.
4. Determine whether the change is performance-significant.
5. Run a benchmark baseline first when it is significant.

During and after implementation:

6. Add or update correctness tests.
7. Add or update benchmarks for significant changes.
8. Run the same benchmarks after the change and compare them.
9. Run relevant initial validation.
10. Review the complete resulting diff for correctness, clarity, repository conventions, architecture, organization, tests, and performance where applicable.
11. Correct actual problems found by the review and improve tests when it exposes a behavioral coverage gap.
12. Re-run affected validation and benchmarks, then summarize the final evidence accurately.

### Significant changes

A change is performance-significant when it could reasonably affect:

- command startup latency or allocation
- runtime execution latency or common-path allocation
- browser startup or readiness behavior
- remote runtime request/response cost
- source loading or build artifact generation cost
- module discovery concurrency or large result processing
- module project scanning, rewriting, staging, or preparation cost
- debugger hooks or interactive dispatch on hot paths
- self-update, release, or provider transport processing when performance is part of the change

This usually does not include:

- comments, documentation, or formatting only
- pure renames with no behavior change
- test-only changes
- help-text changes
- narrow refactors that do not affect behavior or hot paths

Correctness-sensitive filesystem, publication, authentication, and release changes remain high risk, but they are not automatically benchmark-significant unless performance may change.

When in doubt about performance impact, treat the change as significant and benchmark it.

### Benchmark workflow for significant changes

For significant changes:

- Run the relevant benchmark before implementation and save the baseline.
- Run the same benchmark after implementation under comparable conditions.
- Compare `ns/op`, `B/op`, and `allocs/op` where applicable.
- Add a focused benchmark when the changed hot path has none.
- Report the exact benchmark commands and summarize the delta.

If benchmarking is impossible because of the environment or the behavior cannot be isolated meaningfully, state that explicitly and do not claim benchmark validation.

## Test placement rules

- Cobra behavior belongs in the owning `cmd/internal/<feature>` package.
- Public constructor and entrypoint dependency wiring belongs in `cmd` facade tests.
- Shared execution flags and policy behavior belongs in `cmd/internal/execution` tests.
- Runtime selection and adapters belong in `pkg/runtime` tests.
- Run/execute preparation belongs in `pkg/run` tests.
- Source resolution belongs in `pkg/source` tests.
- Browser options and lifecycle belong in `pkg/browser` tests, with platform-neutral tests where possible.
- Debugger parsing, lifecycle, and rendering belong in `pkg/debugger` tests.
- Build artifact and path behavior belongs in `pkg/build` tests.
- Config behavior belongs in `pkg/config` tests.
- Module discovery belongs in `pkg/module/discovery` tests with Registry boundaries faked.
- Installation behavior belongs in `pkg/module/install` tests, including transaction rollback and focused integration coverage for the Go toolchain.
- Scaffolding belongs in `pkg/module/scaffold` tests with filesystem assertions and Specs validation.
- Publication modes and Barn adaptation belong in `pkg/module/publish` tests.
- GitHub authentication, pagination, reconciliation, idempotency, and conflict behavior belong in `pkg/module/publish/github` tests with HTTP and command boundaries isolated.
- Self-update network behavior belongs in `pkg/selfupdate` tests with network boundaries mocked.

Prefer testing the owning package directly, then add command-level coverage for user-visible CLI behavior.

## Validation and evidence

When finishing a non-trivial change, report:

- owning subsystem
- files changed
- tests added or updated
- benchmarks added or updated, if applicable
- validation commands run
- benchmark commands and before/after results, if applicable
- notable invariants preserved or intentionally changed
- any environment-limited validation, clearly separated from product failures
- final self-review completed and any resulting corrections or test improvements

For significant changes:

- Tests alone are insufficient.
- Correctness tests and benchmarks are required when the environment permits them.
- Benchmark results must be compared against a baseline.

### Change discipline

- Adapt an existing local pattern before introducing a new architectural pattern.
- Do not add helper layers, wrappers, interfaces, or exports only for aesthetics.
- Do not move code across packages unless the ownership boundary is wrong.
- Keep diffs focused on the requested task.
- Keep any cleanup required for safety tightly scoped and explain why it is necessary.
- Preserve unrelated work in a dirty worktree.

### Comment and documentation discipline

- Add comments for non-obvious semantics, invariants, side effects, ownership, lifecycle, security, and recovery behavior.
- Do not add comment wallpaper.
- Prefer why, contract, and invariant comments over narration.
- Document user-facing, public, provider, filesystem-mutating, and release behavior more carefully than obvious local helpers.

### Decision bias when uncertain

When uncertain:

- verify current code and source-of-truth configuration
- preserve existing behavior
- prefer the smaller local change
- add a focused test
- benchmark if performance might change
- preserve strict output and mutation contracts
- verify ownership before introducing an abstraction or dependency

## Mandatory final self-review

After completing the implementation and initial validation for any non-trivial task, agents must review the complete resulting diff before considering the task finished.

The purpose of this review is to catch problems in the implementation itself, not to generate additional work or redesign unrelated parts of the repository.

Review the final change for:

- **Correctness**
    - Verify that the implementation satisfies the task requirements completely.
    - Look for missing cases, incorrect assumptions, regressions, boundary conditions, and failure paths.
    - Check error handling, cancellation, cleanup, state transitions, ownership, and lifecycle behavior where applicable.
    - Verify that tests exercise the intended contract rather than merely mirroring the implementation.
- **Code clarity and cleanliness**
    - Look for unnecessary complexity, duplication, excessive nesting, awkward control flow, misleading naming, or code that is difficult to reason about.
    - Prefer straightforward and idiomatic Go over clever implementations.
    - Remove implementation artifacts that are no longer necessary after the final design has taken shape.
- **Repository and Go best practices**
    - Verify that the implementation follows the conventions and mandatory structure rules in this file.
    - Check relevant Go practices, error handling, resource ownership, concurrency behavior, and API design.
    - Do not introduce a pattern merely because it is generally fashionable; it must improve this repository specifically.
- **Architecture**
    - Verify that responsibilities remain in the correct package, type, and layer.
    - Check dependency direction and existing architectural boundaries.
    - Look for unwanted coupling, leaked implementation details, misplaced semantics, or abstractions at the wrong level.
    - Verify that shared semantics remain owned by the appropriate subsystem rather than being duplicated by consumers.
- **Code organization and split**
    - Verify that files, types, methods, functions, and packages have clear responsibilities.
    - Check compliance with the Go type/file and function/method ownership rules in this file.
    - Look for files or functions doing too much.
    - Also avoid unnecessary fragmentation where closely related logic has been split into excessive helpers or files.
    - Ensure that the primary execution path remains easy to follow.
- **Tests**
    - Look for meaningful behavioral gaps, especially negative cases and boundary conditions.
    - Check for brittle tests, redundant tests, tests coupled unnecessarily to implementation details, and assertions too weak to catch plausible regressions.
    - For bug fixes, verify that a test would fail without the fix whenever practical.
- **Performance**
    - For significant changes, inspect the final implementation for accidental allocations, repeated work, unnecessary materialization, or additional hot-path overhead.
    - Compare required benchmark results with the baseline.
    - Do not trade clear correctness for speculative micro-optimization.

When the review finds a problem:

1. Fix correctness issues and regressions.
2. Fix meaningful architectural, ownership, lifecycle, or maintainability problems.
3. Simplify unnecessarily complicated code when doing so clearly improves the implementation.
4. Add or improve tests when the review exposes a behavioral coverage gap.
5. Re-run validation affected by the review-driven changes.
6. Re-run relevant benchmarks if a review-driven change affects benchmarked code.

Do not use the self-review as justification for speculative refactoring, unrelated cleanup, API redesign, or stylistic churn.

Distinguish actual problems from optional preferences. Existing code that is already clear, correct, idiomatic, and appropriately structured should be left alone.

The first working implementation is not automatically the final implementation. The task is complete only after implementation, validation, self-review, necessary corrections, and final validation have been performed.

## Tooling prerequisites

- Go must satisfy `go.mod` (`1.26.5` at the time of this guide).
- CI installs Go `>=1.26`; `go.mod` remains authoritative for the repository minimum.
- `make` is optional but is the preferred entrypoint for repository-defined workflows.
- `staticcheck`, `goimports`, and `revive` are required by lint/format flows; install them with `make install-tools`.
- Release work may require GoReleaser plus signing, authentication, or other environment-specific tooling configured outside this repository.
- Module installation integration tests may require the Go toolchain, writable caches, and dependency access.
- Publication or Registry validation may require network access; distinguish network restrictions from code failures.

## Command matrix

- Download dependencies: `make install`
- Broad validation: `go test ./...`
- Repository test target: `make test`
- Lint, including `staticcheck`, `revive`, and `go vet`: `make lint`
- Format Go files and imports: `make fmt`
- Build the CLI binary: `make compile`
- Full local build flow: `make build`
- Release a version tag: `make release vX.Y.Z`

Prefer narrow validation first, then broaden:

- Feature command changes: run `go test ./cmd/internal/<feature>`.
- Package-local changes: run `go test ./pkg/<name>` or the specific nested package.
- Cross-cutting command/runtime changes: run affected package tests, then `go test ./...` or `make test`.
- Module install changes: run focused unit tests before integration tests that invoke the Go toolchain.
- Release-sensitive changes: run `make build` when the toolchain is available and inspect the release workflow/configuration.

## Editing rules

- Treat `go.mod`, `Makefile`, `.github/workflows/*`, and `.goreleaser.yml` as the source of truth for toolchain, validation, CI, and release behavior.
- Do not add parser-generation rules or generated parser artifacts to this repository; parser ownership belongs to Ferret core.
- Do not vendor or copy Ferret core, Specs, or Barn internals into the CLI.
- Keep version injection compatible with the current `make compile` and GoReleaser flags:
    - `main.version`
    - `github.com/MontFerret/cli/v2/pkg/runtime.version`
- When adding or moving Go packages, verify whether the Makefile's explicit `goimports` package list must be updated.
- If release scripts or GoReleaser configuration change, verify the tag/version flow in `scripts/versions.sh`, `scripts/release.sh`, the Makefile, and release workflows.
- Do not run mutating publication, self-update, release, or project-install flows unless the task explicitly authorizes that mutation.

### Validation expectations

- Run the narrowest tests that prove the changed behavior.
- Finish broader changes with the relevant repository-level command.
- Run `make fmt` after formatting-sensitive Go changes.
- Run `make lint` after lint-sensitive code or public behavior changes when tooling is available.
- Run `git diff --check` for documentation and configuration changes.
- If the environment cannot download the required toolchain or dependencies, report which commands could not run and why.
- If network or loopback restrictions block Registry, publication, browser, or installer tests, classify the environment failure separately from product behavior.

### Expectations for non-trivial changes

When proposing or implementing non-trivial changes:

- identify the owning subsystem first
- preserve invariants unless explicitly changing them
- prefer local, comprehensible changes before new abstractions
- distinguish correctness work from performance work
- preserve user-visible compatibility and mutation safety
- avoid unrelated opportunistic refactors

## Secondary references

- `README.md` for product context and user-facing command examples.
- `CHANGELOG.md` for release history.
- `.github/workflows/build.yml` for the primary CI validation path.
- `.github/workflows/release.yaml` and `.goreleaser.yml` for release behavior.
- `scripts/release.sh` and `scripts/versions.sh` for tag and version derivation.
- The Ferret core repository for language, compiler, bytecode, VM, runtime, formatter, and debugger semantics.
- The Specs repository for portable module schemas and validation.
- The Barn repository for Registry artifacts, discovery APIs, publication preparation, and immutable publication rules.
