# Changelog

All notable changes to Rune will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/octopyid/rune/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/octopyid/rune/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/octopyid/rune/releases/tag/v1.0.0
