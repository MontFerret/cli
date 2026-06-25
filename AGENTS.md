# AGENTS.md

This file is the canonical operating guide for coding agents working in the Ferret CLI repository. It is written for the Ferret v2 CLI only. If repository documentation conflicts with this file, prefer `Makefile`, `go.mod`, `.github/workflows/*`, and `.goreleaser.yml` for commands, toolchain, CI, and release behavior.

## Repo snapshot

- Module path: `github.com/MontFerret/cli/v2`
- Go version in `go.mod`: `1.26.1`
- Binary entrypoint: `ferret/main.go`
- Built binary name: `ferret`
- Core Ferret dependency: `github.com/MontFerret/ferret/v2`
- This repository is the CLI/tooling layer for Ferret v2. Do not assume ownership of core language, compiler, VM, bytecode, parser, or runtime semantics unless the behavior is explicitly implemented in this repository.

## Architectural mental model

The CLI repository wraps Ferret v2 core capabilities into user-facing commands.

Primary flow:

```text
user command -> Cobra command -> CLI option parsing -> source/runtime/browser/config helpers -> Ferret core API -> output/debug/rendering
```

Agents should reason about changes by ownership boundary:

- Command shape, flags, aliases, help text, and user-facing command behavior usually begin in `cmd`.
- Script/source resolution usually belongs in `pkg/source` or the command package that owns the input mode.
- Runtime selection, builtin-vs-remote behavior, parameter passing, and execution setup usually belong in `pkg/runtime` or `pkg/run`.
- Browser process setup, lifecycle, and browser flags belong in `pkg/browser`.
- Interactive debugger REPL behavior belongs in `pkg/debugger`; command-level debug wiring belongs in `cmd/debug.go`; Ferret-core debug session creation belongs in `pkg/runtime/debug.go`.
- Interactive FQL REPL behavior belongs in `pkg/repl`; command wiring belongs in `cmd/repl.go`.
- Build artifact planning and path behavior belongs in `pkg/build`; command wiring belongs in `cmd/build.go`.
- Configuration persistence and config flag binding belong in `pkg/config`.
- Logging behavior belongs in `pkg/logger`.
- Self-update behavior belongs in `pkg/selfupdate` and `cmd/update.go`.
- Release packaging belongs in `.goreleaser.yml`, `scripts/release.sh`, and the Makefile release target.

The CLI should call into Ferret core rather than reimplementing language semantics locally.

## Canonical invariants

- The CLI does not own FQL language semantics.
- The CLI does not own Ferret compiler, VM, bytecode, parser, optimizer, or core runtime value semantics.
- User-facing command behavior must remain predictable, scriptable, and stable unless a task explicitly changes it.
- CLI errors should be actionable and should preserve the best available context: command, input path, runtime type, flag value, or source location.
- Runtime execution must preserve Ferret core behavior instead of compensating for core behavior in CLI-only code.
- Browser resources must be cleaned up deterministically where this repository starts or owns them.
- Remote runtime behavior and builtin runtime behavior must remain clearly separated.
- Debugger-disabled execution paths must not pay for debugger-specific setup.
- Do not assume behavior from the v1 CLI branch unless it is reflected in this repository.

## Package map

Agents should begin with the package whose responsibility owns the requested behavior. Do not infer ownership from file names alone when a package in this map already describes the intended boundary.

### Command layer

* `cmd`
    * Owns Cobra command definitions, command hierarchy, CLI help, flag registration, argument validation, and high-level command orchestration.
    * Keep command files thin. They should parse CLI intent, validate inputs, assemble options, call the owning package, and render command-level errors.
    * Do not bury business logic in Cobra callbacks when it naturally belongs in `pkg/*`.
    * Do not change command names, aliases, flags, or output formats casually; these are user-facing API.

* `ferret`
    * Owns the binary entrypoint and version injection target.
    * Keep this package minimal. It should wire and execute the root command, not implement command behavior.

### Execution and source handling

* `pkg/run`
    * Owns shared run/execute flows and execution input resolution used by commands such as `run`.
    * Prefer this package when changing how CLI execution is prepared, resolved, or dispatched after command-level options are known.
    * Do not place browser-specific process management here; delegate to `pkg/browser`.

* `pkg/runtime`
    * Owns CLI runtime selection and runtime adapters for builtin and remote execution.
    * Owns conversion from CLI runtime options to Ferret core execution/debug sessions.
    * Keep builtin and remote behavior explicitly separated.
    * Do not implement core Ferret semantics here; call Ferret core APIs.
    * Debug-session creation for the CLI belongs here, but interactive debugger behavior does not.

* `pkg/source`
    * Owns CLI source resolution from files, inline input, stdin-like flows, or command-specific source modes.
    * Use this package when behavior depends on how the CLI identifies or loads user-provided source.
    * Preserve source identity so Ferret core diagnostics and debugger output can point back to useful paths.

### Browser and environment support

* `pkg/browser`
    * Owns browser process discovery, startup, wait/readiness behavior, lifecycle cleanup, and browser-specific flags/options.
    * Platform-specific files should keep OS-specific behavior isolated.
    * Browser startup should be lazy or conditional when possible; avoid starting browsers for commands that do not need one.
    * Ensure cleanup paths work on success, failure, cancellation, and early validation errors where this package owns resources.

* `pkg/config`
    * Owns config store behavior, config context, config initialization, config flag binding, and persistent CLI configuration.
    * Preserve key names and config semantics unless the task explicitly changes them.
    * Avoid mixing command-specific validation into generic config storage.

* `pkg/logger`
    * Owns CLI logging options and logger construction.
    * Logging should remain observational.
    * Do not make command semantics, runtime behavior, or control flow depend on log output.

### Developer tooling commands

* `pkg/debugger`
    * Owns the interactive debugger REPL, debugger command parsing, command dispatch, rendering, and the CLI-facing debugger session interface.
    * It should consume Ferret core debug sessions through a small interface seam.
    * It should not own Ferret VM semantics, breakpoint binding semantics, runtime value semantics, or protocol-level DAP behavior.
    * Keep REPL lifecycle behavior explicit: not started, paused, completed, terminated, closed, or equivalent local states.
    * User-facing debugger output should be stable, readable, and friendly for terminal use.

* `pkg/repl`
    * Owns interactive FQL shell behavior.
    * Keep command wiring in `cmd/repl.go`; keep interactive loop behavior here.
    * Preserve cancellation, exit, and user input behavior carefully because this package is interactive.

* `pkg/build`
    * Owns build artifact planning, artifact path handling, and compiled output decisions for CLI build flows.
    * Do not implement compiler semantics here. Use Ferret core APIs for compilation.
    * Keep path behavior explicit and well-tested.

### Maintenance and release support

* `pkg/selfupdate`
    * Owns CLI self-update behavior and update checks.
    * Network behavior must be explicit and error messages must be actionable.
    * Do not silently replace binaries or mutate installation state without clear command intent.

* `scripts`
    * Owns shell helpers used by Makefile or release flows.
    * Keep scripts portable where practical and avoid duplicating logic that already exists in Makefile or Go code.

* `.goreleaser.yml`
    * Owns release packaging behavior.
    * Treat changes here as release-sensitive.

## Where to start by task

- Add or change a command:
    - inspect the relevant file in `cmd`
    - check existing command/flag patterns
    - place reusable behavior in the appropriate `pkg/*` package
    - add command-level tests in `cmd`

- Add or change a flag:
    - inspect `cmd/flags.go` and the owning command file
    - inspect the owning options type in `pkg/*`
    - update help text and tests
    - preserve backward compatibility unless explicitly changing the flag

- Change run behavior:
    - inspect `cmd/run.go`
    - inspect `pkg/run`
    - inspect `pkg/runtime`
    - inspect `pkg/browser` if browser startup or lifecycle is affected

- Change debug behavior:
    - inspect `cmd/debug.go` for command wiring
    - inspect `pkg/runtime/debug.go` for Ferret core debug session setup
    - inspect `pkg/debugger` for REPL commands, lifecycle, parsing, and rendering
    - add focused tests in `cmd`, `pkg/runtime`, and/or `pkg/debugger` based on the changed layer

- Change build or inspect behavior:
    - inspect `cmd/build.go` or `cmd/inspect.go`
    - inspect `pkg/build` for artifact planning/path behavior
    - rely on Ferret core for compilation/disassembly semantics

- Change browser behavior:
    - inspect `cmd/browser.go`
    - inspect `pkg/browser`
    - test platform-neutral behavior where possible
    - isolate platform-specific changes in OS-specific files

- Change config behavior:
    - inspect `cmd/config.go`
    - inspect `pkg/config`
    - preserve existing key names and storage behavior unless explicitly changed

- Change formatting/checking behavior:
    - inspect `cmd/format.go` or `cmd/check.go`
    - call Ferret core formatter/checker APIs rather than reimplementing language logic
    - add tests for CLI behavior and error/output shape

- Change update/release behavior:
    - inspect `cmd/update.go`, `pkg/selfupdate`, `scripts/release.sh`, `.goreleaser.yml`, and `Makefile`
    - treat this as release-sensitive and validate carefully

## Stability guide

Treat these as relatively stable unless the task explicitly targets them:

- command names and primary aliases
- user-facing flags and config keys
- command exit behavior
- output formats that users may script around
- runtime selection model: builtin vs remote
- browser lifecycle expectations
- release binary name and version injection path

Treat these as implementation-sensitive and verify current code before proposing changes:

- debugger REPL lifecycle and command dispatch
- browser startup/wait/cleanup behavior
- remote runtime request/response behavior
- config file loading and persistence
- path resolution for source files and build artifacts
- self-update and release scripts

Do not treat historical discussion, stale README text, v1 behavior, or old branches as authoritative.

## Public command and compatibility rules

- Treat commands, flags, aliases, config keys, exit status behavior, and machine-readable output as CLI API.
- Do not remove or rename public flags without explicit instruction.
- Prefer adding aliases over replacing existing names.
- Preserve existing stdout/stderr intent. User-facing results should usually go to stdout; diagnostics and errors should usually go to stderr.
- Keep command help clear and short enough to be useful in a terminal.
- When adding new output intended for humans, avoid making existing scripted output harder to parse.
- Any intentional backward-incompatible CLI behavior change must be called out explicitly in the final summary.

## Ferret core boundary rules

- Do not reimplement FQL parsing, formatting, compilation, bytecode handling, VM behavior, runtime value semantics, or debugger breakpoint binding in the CLI.
- If a behavior belongs in Ferret core, change Ferret core separately or adapt to the existing core API.
- The CLI may translate core errors into clearer command-level messages, but it should not obscure specific diagnostics.
- Keep CLI adapters thin around Ferret core APIs.
- When a core Ferret version bump changes behavior, update CLI tests and docs to reflect the new contract.

## Debugger CLI rules

- `cmd/debug.go` owns CLI argument/flag validation and high-level debug startup.
- `pkg/runtime/debug.go` owns creation of the Ferret core debug session from CLI runtime options.
- `pkg/debugger` owns interactive debugger UX: command parsing, aliases, dispatch, prompt lifecycle, and terminal rendering.
- The debugger REPL should depend on a small session interface, not directly on command or runtime packages.
- Do not add daemon, DAP, or remote debug assumptions to the local CLI debugger unless the task explicitly requests it.
- Debugger commands should fail safely after completion or termination and should not call into a closed/completed session unnecessarily.
- Destructive commands such as deleting breakpoints must not be repeated implicitly by empty input.
- Evaluation commands should be clearly described as safe/limited when Ferret core exposes limited evaluation behavior.
- Keep source snippets and breakpoint messages accurate: requested location and bound location are distinct concepts.

## Browser lifecycle rules

- Commands should only start a browser when the selected runtime/input actually needs browser support.
- If the CLI starts a browser process, cleanup must be deterministic on normal completion, command error, and cancellation where possible.
- Platform-specific browser behavior belongs in platform-specific files.
- Avoid global mutable browser state unless the current package already owns it and tests cover it.
- Browser wait/readiness errors should explain what the user can do next.

## Runtime and remote execution rules

- Keep builtin runtime behavior local and direct.
- Keep remote runtime behavior explicit and network-aware.
- Do not silently fall back from remote to builtin or builtin to remote.
- Debug currently requires builtin runtime unless explicitly implemented otherwise.
- Parameter parsing should preserve JSON-aware behavior and test ambiguous cases such as strings that look like JSON values.
- Cancellation and context propagation should be preserved through runtime calls.

## Error and diagnostic quality rules

- User-facing errors should be specific and actionable.
- Prefer errors that name the unsupported mode and the supported alternative.
- Preserve wrapped errors where callers or tests rely on error identity.
- Avoid replacing useful Ferret core diagnostics with generic CLI errors.
- For path errors, include the relevant path when safe and useful.
- For unsupported debug modes, tell the user what to do instead, for example use the original `.fql` source file or builtin runtime.

## Go type and file structure rules

These rules are mandatory unless the task explicitly requires otherwise.

- Do not define multiple method-bearing structs in the same `.go` file.
- Prefer declaring a method-bearing struct as a standalone `type Name struct { ... }`.
- A method-bearing struct should usually live in its own file, named after the primary type or responsibility whenever practical.
- Grouped `type ( ... )` declarations are allowed for interfaces, passive data-only structs, and other small related helper/value types that belong to the same narrow concern.
- A grouped `type ( ... )` block may also contain exactly one method-bearing struct when:
    - it is the only behavioral type in the file, and
    - the other grouped types are passive helper/value types from the same narrow concern.
- Do not use grouped `type ( ... )` declarations to hide multiple substantial behavioral types.
- If a helper struct later gains methods and would create more than one method-bearing struct in the file, extract it into its own file immediately.
- Methods for a struct should live in the same file as the struct unless there is a strong, explicit reason to split by concern.
- Do not place a new method-bearing struct into an existing file just because the code compiles.

Allowed:

```go
type (
	commandResult struct {
		Exit bool
	}

	commandHandler interface {
		Handle(context.Context) commandResult
	}
)
```

Avoid:

```go
type (
	Runner struct {
		// ...
	}

	debugPrompt struct {
		// ...
	}
)
```

## Function and method ownership rules

These rules are mandatory unless the task explicitly requires otherwise.

- A file centered on a method-bearing type should contain the type, its methods, and its constructors only.
- Do not mix package-level helper functions into a file that already contains methods for a primary type.
- In type-centered files, constructor functions are the only normally allowed package-level functions.
- If logic conceptually belongs to the primary type, implement it as a method.
- If logic does not belong to the type and must remain a package-level function, place it in a separate helper-focused file.
- Package-level functions are preferred only when there is no natural owning type or when the behavior is genuinely package-level.
- If a file contains both methods and non-constructor package-level functions, that is usually a structure violation and should be refactored.

## Comment rules for functions and methods

- Do not add comments to every function or method by default.
- Exported functions and methods should usually have doc comments, especially in command-facing, package-facing, or extension-facing code.
- Unexported functions and methods should be commented only when they carry non-obvious behavior, invariants, side effects, ownership rules, cleanup expectations, or protocol/lifecycle constraints.
- Comments must explain intent, contract, invariants, side effects, or lifecycle behavior.
- Prefer comments that explain why the code exists, what must remain true, or how the method is meant to be used.
- Do not write comments that merely restate the method name or signature.
- Avoid comment wallpaper. Dense, meaningful comments are preferred over mechanically documenting obvious code.

Preferred:

```go
// Close releases resources associated with the browser process.
// It is safe to call multiple times.
func (p *Process) Close() error
```

Avoid:

```go
// Close closes the process.
func (p *Process) Close() error
```

## Response and code style

When assisting with this repository, avoid large unstructured blocks of prose or code.

Prefer responses that are easy to scan:

- Use short sections with clear headings.
- Use bullet points for decisions, trade-offs, and follow-up work.
- Use code blocks only for actual code, commands, or configuration.
- Prefer focused snippets or diffs over full-file dumps.
- Explain why a change is needed before showing how to implement it.
- Keep comments in code useful and minimal.
- Avoid repeating the same context in multiple places.
- When the change touches multiple files, summarize the role of each file first.

The expected tone is practical, concise, and engineering-focused.

## Development practice expectations

Agents must follow repository-specific engineering discipline rather than generic style preferences.

### Core principles

- Preserve correctness first.
- Preserve subsystem boundaries and invariants.
- Prefer the smallest local change that fully solves the task.
- Avoid introducing abstractions, indirection, or refactors unless they are necessary for correctness, maintainability, or an explicitly requested design change.
- Do not optimize by intuition alone; use measurements for performance-sensitive work.
- Keep behavioral ownership obvious in code structure, naming, and file layout.

### Mandatory expectations

- Identify the owning subsystem before making a non-trivial change.
- Preserve existing behavior unless the task explicitly requires changing it.
- Add or update tests for any behavior change.
- Add or update benchmarks for any significant change.
- Run the narrowest relevant validation first, then broaden as appropriate.
- Do not claim tests, benchmarks, or validation were completed unless they were actually run.
- Do not treat historical discussions, abandoned directions, v1 behavior, or old branches as authoritative over current code and repository guidance.
- Do not perform opportunistic refactors unrelated to the requested task unless they are required for correctness.

### Required workflow for non-trivial changes

Before making a non-trivial change, agents must:

1. Identify the owning subsystem.
2. Identify the contract, invariant, or behavior being preserved or changed.
3. Choose the smallest reasonable implementation that fits the existing design.
4. Determine whether the change is significant.
5. Add or update correctness tests.
6. Add or update benchmarks if the change is significant.
7. Run relevant validation and summarize the results accurately.

### Significant changes

A change is significant when it could reasonably affect:

- command startup latency
- runtime execution latency
- browser startup/wait behavior
- allocation patterns on common command paths
- remote runtime request/response behavior
- build artifact generation cost
- debugger hooks or REPL command dispatch on hot execution paths
- release/install/update behavior that affects users broadly

This usually does not include:

- comment-only, docs-only, or formatting-only edits
- pure renames with no behavior change
- test-only changes
- command help text changes
- narrowly scoped refactors that do not affect behavior or hot paths

When in doubt, treat the change as significant and benchmark it or explain why benchmarking is not practical.

### Benchmark workflow for significant changes

For significant changes, agents must:

- run relevant benchmarks before making the change and save the results as a baseline
- implement the change
- run the same benchmarks again after the change
- compare before/after results, preferably including `ns/op`, `B/op`, and `allocs/op`
- report the benchmark command used and summarize the performance delta

If no relevant benchmark exists for the changed hot path, add one when practical.

If benchmark tooling or environment is unavailable, state that explicitly and do not claim benchmark validation was completed.

## Test placement rules

- Cobra command behavior should have tests in `cmd`.
- Runtime option and adapter behavior should have tests in `pkg/runtime`.
- Run/execute input behavior should have tests in `pkg/run`.
- Source resolution behavior should have tests in `pkg/source`.
- Browser option/lifecycle behavior should have tests in `pkg/browser` where possible.
- Debugger command parsing, REPL lifecycle, and rendering should have tests in `pkg/debugger`.
- Build artifact/path behavior should have tests in `pkg/build`.
- Config behavior should have tests in `pkg/config` if changed.
- Self-update behavior should be tested with network boundaries mocked or isolated.

Prefer testing the owning package directly, then add command-level coverage when user-visible CLI behavior changes.

## Validation and evidence

When finishing a non-trivial change, agents should report:

- owning subsystem
- files changed
- tests added or updated
- benchmarks added or updated, if applicable
- validation commands run
- benchmark commands run, if applicable
- notable invariants preserved or intentionally changed

For significant changes:

- tests alone are not sufficient unless benchmarking is genuinely not practical
- both correctness tests and benchmarks are expected when the environment allows them
- benchmark results should be compared against a baseline

### Change discipline

- Prefer adapting an existing local pattern over introducing a new architectural pattern.
- Do not add new helper layers, wrappers, interfaces, or abstractions only for aesthetic reasons.
- Do not move code across packages unless the ownership boundary is genuinely wrong.
- Keep diffs focused on the requested task.
- If a cleanup is necessary to make the requested change safe, keep it tightly scoped and explain why it was needed.

### Comment and documentation discipline

- Add comments where semantics, invariants, side effects, ownership, lifecycle, or recovery behavior are non-obvious.
- Do not add comment wallpaper.
- Prefer comments that explain why, contract, or invariants rather than implementation narration.
- Public and user-facing behavior should be documented more carefully than local obvious helpers.

### Decision bias when uncertain

When uncertain:

- preserve existing behavior
- prefer the smaller local change
- add a focused test
- treat the change as significant if performance might be affected
- verify ownership before introducing a new abstraction or package-level dependency

## Tooling prerequisites

- Go must be installed at the version required by `go.mod`.
- `make` is optional but is the preferred entrypoint for repo-defined workflows.
- `staticcheck`, `goimports`, and `revive` are needed for lint/format flows; install them with `make install-tools`.
- Release work may require GoReleaser and any signing/notarization tools configured outside this repository.

## Command matrix

- Broad validation: `go test ./...`
- Repo test target: `make test`
- Lint: `make lint`
- Format: `make fmt`
- Build the CLI binary: `make compile`
- Full local build flow: `make build`
- Release: `make release TAG=<tag>`

Prefer narrow validation first, then broaden:

- Package-local changes: run `go test ./pkg/<name>` or `go test ./cmd`.
- Cross-cutting command/runtime changes: run the affected package tests, then `go test ./...` or `make test`.
- Release-sensitive changes: run `make build` when the toolchain is available.

## Editing rules

- Treat `Makefile`, `.github/workflows/*`, and `.goreleaser.yml` as the source of truth for validation and release flows.
- Do not add parser-generation rules to this repository; parser generation belongs to Ferret core.
- Do not vendor or duplicate Ferret core internals into the CLI.
- Keep version injection compatible with the existing `make compile` flags:
    - `main.version`
    - `github.com/MontFerret/cli/v2/pkg/runtime.version`
- If you change package paths used by `goimports` in `make fmt`, update the Makefile format target as part of the same change.
- If you change release scripts or GoReleaser config, verify the expected tag/version flow in `scripts/versions.sh` and `scripts/release.sh`.

### Validation expectations

- After code changes, run the narrowest tests that prove the behavior you touched.
- Before finishing broader changes, run the relevant repo-level command from the matrix above.
- If you changed formatting-sensitive files, run `make fmt` when available.
- If you changed lint-sensitive code paths or public behavior, run `make lint` when available.
- If the local environment cannot download the required Go toolchain or dependencies, state that explicitly and report which validation commands could not be run.

### Expectations for non-trivial changes

When proposing or implementing non-trivial changes:

- identify the owning subsystem first
- preserve invariants unless the task explicitly changes them
- prefer local, comprehensible changes before introducing new abstractions
- distinguish correctness work from performance work
- do not perform opportunistic refactors unrelated to the requested task unless they are necessary for correctness

## Secondary references

- `README.md` for product context and user-facing command examples.
- `CHANGELOG.md` for release history.
- `.github/workflows/*` for CI behavior.
- `.goreleaser.yml` for release packaging.
- The Ferret core repository for language/compiler/VM/runtime semantics.
