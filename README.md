<p align="center">
  <img src="logo.svg" alt="Rune" />
</p>

<p align="center">
  <a href="https://github.com/octopyid/rune/releases">
    <img src="https://img.shields.io/github/v/release/octopyid/rune?style=for-the-badge&logo=github&label=Release&color=6e56cf" alt="Release" />
  </a>
  <a href="https://github.com/octopyid/rune/actions/workflows/ci.yml">
    <img src="https://img.shields.io/github/actions/workflow/status/octopyid/rune/ci.yml?style=for-the-badge&logo=github&label=CI" alt="CI" />
  </a>
  <a href="https://github.com/octopyid/rune/blob/main/LICENSE">
    <img src="https://img.shields.io/badge/license-MIT-blue?style=for-the-badge&color=3d9970" alt="License" />
  </a>
  <a href="https://go.dev">
    <img src="https://img.shields.io/badge/Go-1.21%2B-00ADD8?style=for-the-badge&logo=go&logoColor=white" alt="Go" />
  </a>
</p>

---

## About Rune

Rune is a project-local task runner written in Go. It turns your `Runefile` into a self-documenting CLI — every task becomes a real command with arguments, flags, namespaces, and a generated `--help` menu.

Inspired by the mental model of **Laravel Artisan**: every task is a first-class CLI command, not a build target. Make is powerful, Just is simple — but neither supports colon namespaces or boolean flags out of the box. Rune does.

```bash
rune build --race        # boolean flags, not ENV=var workarounds
rune db:migrate          # colon namespaces that actually work
rune --dry-run release   # inspect the full execution plan first
rune compose up --build  # full passthrough to any underlying tool
```

> [!NOTE]
> Rune searches for a `Runefile` starting from the current directory and traversing up to parent directories, so you can invoke `rune` from any subdirectory within your project.

---

## Key Features

- **Colon Namespaces** — Organize tasks with `rune db:migrate` instead of flat, hyphenated names.
- **Boolean Flags** — Pass `--race` or `--verbose` directly without `ENV=var` workarounds.
- **Auto-generated `--help`** — Every task has its own help menu generated from the `Runefile`.
- **Dependency Deduplication** — A task runs exactly once no matter how many dependants require it.
- **Interactive Confirmation** — Protect destructive tasks with a built-in confirmation prompt.
- **Dry-run Mode** — Simulate the full execution plan without any side effects.

### Comparison

|  | Make | Just | Rune |
| :--- | :---: | :---: | :---: |
| Positional arguments | `ENV=var` | ✓ | ✓ |
| Boolean flags (`--flag`) | ✗ | ✗ | ✓ |
| Auto-generated `--help` | ✗ | ✗ | ✓ |
| Colon namespaces (`db:fresh`) | ✗ | ✗ | ✓ |
| Interactive confirmation | ✗ | ✗ | ✓ |
| Shell completion | Partial | Partial | ✓ |
| Dependency deduplication | ✓ | ✗ | ✓ |

---

## Install

**Homebrew (macOS / Linux)**

```bash
brew install octopyid/tap/rune
```

**Go**

```bash
go install github.com/octopyid/rune/cmd/rune@latest
```

<details>
<summary>Build from source</summary>
<br>

```bash
git clone https://github.com/octopyid/rune.git
cd rune
go build -o /usr/local/bin/rune ./cmd/rune
```

</details>

---

## Quick Start

### Prerequisites

- Go **1.26** or newer

### Steps

**1. Create a `Runefile` at your project root:**

```text
#[Build the application binary]
build target="dev" --race?:
    go build {{race}} -o ./bin/app ./...

#[Run unit and integration tests]
test: build
    go test ./...

#[Reset the database schema]
#[confirm: This will permanently delete all database data.]
db:fresh:
    dropdb --if-exists app_dev && createdb app_dev

#[Forward arbitrary commands to Docker Compose]
compose *args:
    docker compose {{args}}
```

**2. Run your tasks:**

```bash
rune                           # list all available tasks
rune build                     # run with defaults
rune build production --race   # with arguments and flags
rune build --help              # auto-generated per-task help
rune --dry-run release         # simulate without executing
```

> [!TIP]
> Use `rune <namespace>` or `rune <namespace> --help` to discover tasks grouped under a namespace, for example `rune db` or `rune db --help`.

---

## Documentation


<details>
<summary><b>Task Syntax</b> — Signatures, arguments, flags, and passthrough</summary>
<br>

A `Runefile` consists of metadata attributes, task signatures, and indented command bodies.

```text
#[Task description]
#[confirm: Confirmation prompt]
task_name [arguments...] [--flags...] [*passthrough] : [dependencies...]
    command 1
    command 2
```

| Element | Description |
| :--- | :--- |
| `#[...]` | Metadata — attaches description or safety prompt to the next task |
| `# ` | Comment — ignored entirely |
| Signature | Non-indented line ending with `:` (or `: dep1 dep2`) |
| Commands | Lines indented with 4 spaces or a tab |
| `{{var}}` | Interpolation — expands to the resolved argument or flag value |

### Arguments

#### Required Positional

```text
greet name:
    echo "Hello, {{name}}!"
```

```bash
rune greet Supian
# Output: Hello, Supian!
```

If a required argument is omitted, Rune displays an actionable error:

```text
✗ Missing argument: name

Usage:
  rune greet <name>
```

#### Default Arguments

Assign a default value using `=` (supports quoted or unquoted values):

```text
build target="dev":
    echo "Building for target: {{target}}"
```

```bash
rune build             # Building for target: dev
rune build production  # Building for target: production
```

> **Note**: Required arguments must always precede default arguments in the signature.

### Flags

Declare optional boolean flags using `--<name>?`:

```text
build target="dev" --race?:
    go build {{race}} ./...
```

```bash
rune build
rune build --race
rune build production --race
```

When `--race` is passed, `{{race}}` expands to `--race`. When omitted, it expands to nothing.

If you mistype a flag, Rune calculates edit distance and suggests the intended option:

```bash
rune build --rce
# ✗ Unknown option: --rce
# Did you mean: --race
```

### Passthrough Arguments (`*args`)

When an underlying tool accepts arbitrary arguments, declare an explicit passthrough parameter using `*<name>`:

```text
compose *args:
    docker compose {{args}}
```

```bash
rune compose exec api sh -c "echo 'Health check'" --user=root
```

### Documenting Arguments & Flags (Doc-Comments)

Document task parameters and flags by placing a contiguous comment block directly before the task header:

- Use `# --<flag>: Description` for boolean flags.
- Use `# <arg>: Description` for positional arguments.
- Use `# *<args>: Description` or `# <args>: Description` for passthrough arguments.

```text
#[Build Android APK]
# target: Target build environment (dev, staging, prod)
# --split: Build split-per-ABI APKs alongside universal APK
# --minify: Enable ProGuard/R8 code shrinking and obfuscation
build:apk target="dev" --split? --minify?:
    ./gradlew assembleRelease
```

Running `rune build:apk --help` automatically renders these descriptions in the `Arguments:` and `Options:` tables:

```text
Arguments:
  target                Target build environment (dev, staging, prod) [default: "dev"]

Options:
      --split           Build split-per-ABI APKs alongside universal APK
      --minify          Enable ProGuard/R8 code shrinking and obfuscation
  -h, --help            Display help for the given command
  -v, --version         Display this application version
```

> **Note**: Comments that do not match declared parameter names (such as developer notes `# NOTE: ...` or `# TODO: ...`), comments separated by blank lines, and comments inside the task body are completely ignored.

### Interactive TTY & REPL

Rune attaches the child process directly to your terminal's `stdin`, `stdout`, and `stderr`:

- **Interactive Tools** — `bash`, `python`, `psql`, `ssh`, `vim`, `htop` run natively.
- **Arrow Keys & Shortcuts** — Interactive navigation and control keys work out-of-the-box.
- **Signal Forwarding** — `SIGINT` (`Ctrl+C`), `SIGTERM`, and `SIGHUP` are forwarded.
- **Exit Code Preservation** — The exact exit code of the underlying command is propagated.

Tasks also seamlessly participate in standard Unix pipes:

```bash
cat dump.sql | rune db:import
rune compose ps | grep running
```

</details>

<details>
<summary><b>Features</b> — Namespaces, dependencies, confirmation, dry run, env</summary>
<br>

### Namespaces

Group related tasks using colon notation (`namespace:task`):

```text
db:migrate:
    migrate -path ./migrations -database "$DATABASE_URL" up

db:seed:
    go run ./cmd/seed

db:fresh:
    dropdb --if-exists app_dev && createdb app_dev
```

Namespaces act like command groups:

```bash
rune db        # discover tasks inside the 'db' namespace
rune db --help
```

```text
Namespace:
  db

Usage:
  rune db:<task> [arguments] [flags]

Available Tasks:
  fresh    Reset the database schema
  migrate  Run database migrations
  seed     Seed the database
```

### Dependencies

Specify task dependencies on the signature line after the colon:

```text
build:
    go build ./...

test: build
    go test ./...

release: build test
    ./release.sh
```

| Guarantee | Description |
| :--- | :--- |
| **Deterministic Ordering** | Dependencies execute before their dependent task |
| **Deduplication** | `build` runs only once even if multiple tasks depend on it |
| **Cycle Detection** | Circular dependencies are detected before any command runs |
| **Fail-Fast** | Non-zero exit code halts execution immediately |

### Confirmation & Safety

Protect dangerous actions with `#[confirm: message]`:

```text
#[Reset the database schema]
#[confirm: This will permanently delete all database data.]
db:fresh:
    dropdb --if-exists app_dev && createdb app_dev
```

```text
  WARN  This will permanently delete all database data.

  Continue?
  ❯ Yes    No
```

| Key | Action |
| :--- | :--- |
| `←` / `→`, `Tab`, `h` / `l` | Toggle selection |
| `Enter` or `Space` | Execute selected choice |
| `y` | Confirm immediately |
| `n`, `Esc`, `Ctrl+C` | Cancel |

Confirmation occurs **before any task in the dependency graph executes**. In non-interactive environments (CI, pipes), Rune falls back to line-based `Continue? [y/N]` input.

To skip confirmation entirely:

```bash
rune --yes db:fresh   # or -y
```

### Dry Run

Simulate execution and inspect the resolved command sequence without running anything:

```bash
rune --dry-run release
```

```text
[dry-run] Execution plan for 'release':
  1. build
     $ go build ./...
  2. test
     $ go test ./...
  3. release
     $ ./release.sh
```

### Environment Handling

| Variable | Description |
| :--- | :--- |
| Auto `.env` loading | Loaded automatically if `.env` exists beside the `Runefile` |
| `RUNE_TASK` | Name of the active task (e.g. `build`) |
| `RUNE_ARG_<NAME>` & `<NAME>` | Value of resolved positional arguments |
| `RUNE_FLAG_<NAME>` | Set to `1` when the flag is enabled |

Child processes also inherit the full system environment (`os.Environ()`).

</details>

<details>
<summary><b>Console UI Components</b> — Built-in <code>info</code>, <code>warn</code>, <code>done</code>, <code>fail</code>, <code>error</code></summary>
<br>

Rune includes built-in console UI components for use directly from the shell or inside your `Runefile` task recipes.

| Command | Badge | Color | Exit Code | Purpose |
| :--- | :--- | :--- | :--- | :--- |
| `rune info <msg>` | `INFO` | Blue bg, white bold | `0` | Informational status updates |
| `rune warn <msg>` | `WARN` | Yellow bg, black bold | `0` | Warnings or cautionary alerts |
| `rune done <msg>` | `DONE` | Green bg, white bold | `0` | Successful completions (alias: `rune success`) |
| `rune fail <msg>` | `FAIL` | Red bg, white bold | `1` | Task or validation failure alerts |
| `rune error <msg>` | `ERROR` | Red bg, white bold | `1` | System error or fatal exception alerts |

Example usage in a `Runefile`:

```text
#[Deploy application to production]
#[confirm: Are you sure you want to deploy to production?]
deploy:
    rune info "Preparing deployment assets..."
    npm run build
    rune info "Syncing files to remote server..."
    rsync -avz ./dist/ user@example.com:/var/www/
    rune done "Deployment finished successfully!"

#[Verify system dependencies]
check:
    test -f .env || (rune error "Missing .env configuration file!" && exit 1)
    git diff --quiet || (rune fail "Working directory has unstaged changes!" && exit 1)
    rune done "All checks passed."
```

Output:

```text
  INFO  Preparing deployment assets...

  DONE  Deployment finished successfully!
```

> **Task Precedence**: If your `Runefile` defines a custom task named `info`, `warn`, `error`, `fail`, or `done`, your custom task takes precedence over the built-in component.

</details>

<details>
<summary><b>Shell Completion</b> — Bash, Zsh, and Fish setup</summary>
<br>

Rune provides tab completion for Bash, Zsh, and Fish.

### Zsh

Add to your `~/.zshrc`:

```bash
source <(rune completion zsh)
```

### Bash

Add to your `~/.bashrc`:

```bash
source <(rune completion bash)
```

### Fish

Add to your `~/.config/fish/config.fish`:

```fish
rune completion fish | source
```

### What Gets Completed

| Input | Completion |
| :--- | :--- |
| `rune bu<TAB>` | `build` |
| `rune d<TAB>` | `db` |
| `rune build --<TAB>` | `--race` |
| `rune in<TAB>` | `info`, `warn`, `done`, `error`, `help`, `list` |
| `rune --<TAB>` | `--dry-run`, `--help`, `--yes`, etc. |

</details>

---

## Boundaries

Rune organizes and runs tasks. It does not try to manage your entire project lifecycle.

Rune is **not**:

- A build system or CI/CD platform
- A deployment tool or process supervisor
- A package or environment manager
- A scheduler or remote execution system
- A scripting language or workflow engine

If a feature doesn't directly improve how tasks are **defined, discovered, invoked, or protected** — it doesn't belong in Rune.

---

## Security

Rune is a local CLI tool. It does not collect telemetry, phone home, or require network access to operate.

**Reporting a Vulnerability:** If you discover a security issue, please follow the responsible disclosure process described in [SECURITY.md](SECURITY.md). Do not open a public GitHub issue for security vulnerabilities.

---

## Contributing

Contributions are welcome — bug reports, feature discussions, and pull requests alike. If you are new to the codebase, look for issues labeled `good first issue` as a starting point.

Before submitting a pull request, please read [CONTRIBUTING.md](CONTRIBUTING.md). It covers the code style, commit conventions, and the PR review process.

---

## License

Rune is licensed under the **MIT License**. See [LICENSE](LICENSE) for the full license text.
