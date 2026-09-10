# Changelog

All notable changes to Rune will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.4.1] - 2026-09-11

### Fixed

- **Zsh Shell Completion Execution**:
  - Fix `can only be called from completion function` error when sourcing completion scripts directly (e.g. `source <(rune completion zsh)`) or reloading Oh My Zsh (`omz reload`).
  - Guard immediate execution of `_rune "$@"` with `[ "$funcstack[1]" = "_rune" ]` so the completion function is only evaluated during completion calls, not during shell initialization.
  - Automatically register completion via `compdef _rune rune` when the script is sourced into an active shell.

## [1.4.0] - 2026-09-11

### Added

- **Task Working Directory (`#[dir: <path>]`)**:
  - Support task-scoped working directory via `#[dir: <path>]`, resolving paths relative to `Runefile` location (or absolute paths).
  - Validates directory existence prior to task execution with fail-fast error reporting.
  - Maintains isolated working directories across dependency execution.
- **Task Environment Variables (`#[env: ...]`)**:
  - Support declaring task-scoped environment variables with single, multiple, or semicolon-separated (`KEY=VAL; KEY2=V2`) assignments.
  - Variables override `.env` and system environment values during task execution without leaking across tasks or to the parent shell.
  - Multiple `#[env]` attributes on a task accumulate predictably.
- **Verbose Command Echo (`--verbose`)**:
  - Global option (`--verbose`) displaying each command formatted as `$ <command>` prior to execution.
  - Properly formats arguments with shell escaping when containing whitespace or quotes.
- **Task Execution Timing (`--time`)**:
  - Global option (`--time`) displaying elapsed runtime for tasks.
  - Formats single task elapsed duration (`✔ build (0.11s)`) and multi-task dependency breakdowns (`[1/3] ... [2/3] ... ✔ Total: 5.48s`).
  - Correctly tracks and reports duration even when a task fails.
- **Standalone Shell Installer (`install.sh`)**:
  - POSIX-compliant one-line installer supporting automatic OS (`darwin`/`linux`) and architecture (`amd64`/`arm64`) detection.

### Changed

- **Go Version Compatibility**:
  - Lower minimum Go requirement to **Go 1.22+** in `go.mod`, `README.md`, and `CONTRIBUTING.md` by pinning compatible module dependencies (`golang.org/x/term`, `golang.org/x/sys`, `github.com/fatih/color`).
- **Documentation Refinement**:
  - Remove redundant manual Table of Contents in `README.md` in favor of native GitHub Markdown navigation.
  - Fix command syntax examples in Runefile recipes to avoid invalid shell chaining assumptions.

## [1.3.2] - 2026-09-10

### Fixed

- **Interactive TTY Input & Signal Handling**:
  - Fix interactive child process terminal access by ensuring child processes share the foreground process group with Rune instead of detaching via `Setpgid`.
  - Fix single-key keyboard shortcuts (e.g. Flutter `s` for screenshot, `r` for hot reload, `h` for help, `q` for quit) previously blocked because background processes were suspended by `SIGTTIN`.
  - Fix `Ctrl+C` (`SIGINT`) handling during child process execution, adding force-kill on repeated interrupt signals.
  - Fix signal exit code propagation to return POSIX-standard `128 + signal` (e.g. 130 for `SIGINT`) and abort remaining tasks in the plan on interruption.

## [1.3.1] - 2026-09-10

### Fixed

- **Module Entrypoint & Distribution**:
  - Restructure entrypoint to `cmd/rune` for standard Go CLI installations.
  - Fix Go module proxy compatibility: supersedes earlier unversioned proxy cache anomalies (`v1.2.0` and `v1.3.0` skipped) to restore seamless `go install github.com/octopyid/rune/cmd/rune@latest`.

## [1.1.0] - 2026-09-10

### Added

- **Doc-Comments for Task Arguments & Flags**:
  - Support natural doc-comments directly preceding task headers to document CLI arguments and flags.
  - Parse `# --<flag>: <desc>` for boolean flags, `# <arg>: <desc>` for positional arguments, and `# *<args>: <desc>` for passthrough arguments.
  - Automatically render custom descriptions into `Arguments:` and `Options:` help tables (`rune <task> --help`).
  - Strict matching against declared task parameters; unassociated comments (like `# NOTE: ...` or `# TODO: ...`), comments separated by blank lines, and comments inside the task body are ignored.

## [1.0.0] - 2026-09-10

Initial release of Rune — a small, predictable, and opinionated task runner for Go and modern development workflows, inspired by Make, Just, and Laravel Artisan.

### Added

- **CLI-First Mental Model**:
  - Every task signature defines a real CLI command.
  - Positional parameters with optional default values (`task target="dev":`).
  - Boolean flag parameters with automatic flag syntax (`task --race?:`).
  - Namespace command grouping using colon syntax (e.g. `rune db:fresh`, `rune test:coverage`).
- **Passthrough & Interactive I/O**:
  - Explicit passthrough argument support (`*args`) preserving raw arguments, arbitrary flags, and quotation boundaries without shell mangling.
  - Direct TTY attachment for interactive programs, REPLs (`tinker`, `python`, `vim`), and curses applications.
  - Signal forwarding (`SIGINT`/`Ctrl+C`, `SIGTERM`) directly to child processes.
  - Standard Unix stream piping (`cat dump.sql | rune db:import`).
  - Exact process exit code propagation.
- **Topological Dependency Graph**:
  - Deterministic dependency resolution executing prerequisites before dependents.
  - Automatic dependency deduplication across shared branches in DAG.
  - Cycle detection (`A -> B -> A`) before executing any commands.
  - Fail-fast execution aborting immediately when any task fails.
- **Interactive Safety & Confirmation**:
  - `#[confirm: message]` directive prompting users before running dangerous actions.
  - Upfront graph-level verification checking confirmations before any dependency executes.
  - Interactive arrow-key navigation (`❯ Yes   No`) with raw terminal input mode and instant terminal restoration.
  - Keyboard shortcuts (`←`/`→`, `Tab`, `h`/`l`, `y`, `n`, `Esc`, `Ctrl+C`).
  - Automated non-interactive fallback for CI runners, background processes, and pipes without terminal freezing.
  - Non-interactive bypass via `--yes` / `-y`.
- **Console UI Components**:
  - Built-in console badges styled after Laravel Console Components / Termwind:
    - `rune info <msg>`: Blue badge with informational status.
    - `rune warn <msg>`: Yellow badge with black bold text for warnings.
    - `rune done <msg>` / `rune success <msg>`: Green badge for successful completion.
    - `rune fail <msg>`: Red badge for task, test, or validation failures, exiting with code 1.
    - `rune error <msg>`: Red badge for system error alerts and fatal exceptions, exiting with code 1.
  - Reusable directly in terminal or inside `Runefile` task recipes.
  - Built-in help routing (`rune help info`, `rune info --help`) and typo suggestions.
- **Simulation & Introspection**:
  - `--dry-run` flag to simulate execution and preview topologically sorted execution plans without running commands.
- **Help & Self-Documentation**:
  - Minimalist block ASCII banner inspired by Deno/Bun.
  - Laravel Artisan inspired help menus for root commands, namespaces, and tasks.
  - Levenshtein-based typo suggestions ("Did you mean: ...") for unknown tasks, namespaces, and flags.
- **Environment Handling**:
  - Automatic `.env` loading from project directory.
  - Injection of task context variables (`RUNE_TASK`, `RUNE_ARG_<NAME>`, `RUNE_FLAG_<NAME>`).
  - System environment inheritance.
- **Shell Completion**:
  - Native tab-completion generators for Zsh (`source <(rune completion zsh)`), Bash, and Fish.
  - Autocompletion for tasks, namespaces, flags, global options, and built-in UI helpers.
- **Project Governance & Repository Standards**:
  - MIT License (`LICENSE`).
  - Security policy (`SECURITY.md`) with private disclosure instructions.
  - Contributor guidelines (`CONTRIBUTING.md`) with `Runefile` self-hosting instructions.
  - Code of conduct (`CODE_OF_CONDUCT.md`).
  - Sponsorship configuration (`.github/FUNDING.yml`) supporting GitHub Sponsors and Ko-fi.
  - Pull request template (`.github/PULL_REQUEST_TEMPLATE.md`).
  - GitHub Issue Forms (`.github/ISSUE_TEMPLATE/bug_report.yml`, `feature_request.yml`, `config.yml`).

[Unreleased]: https://github.com/octopyid/rune/compare/v1.4.1...HEAD
[1.4.1]: https://github.com/octopyid/rune/compare/v1.4.0...v1.4.1
[1.4.0]: https://github.com/octopyid/rune/compare/v1.3.2...v1.4.0
[1.3.2]: https://github.com/octopyid/rune/compare/v1.3.1...v1.3.2
[1.3.1]: https://github.com/octopyid/rune/compare/v1.1.0...v1.3.1
[1.1.0]: https://github.com/octopyid/rune/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/octopyid/rune/releases/tag/v1.0.0
