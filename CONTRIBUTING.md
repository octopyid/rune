# Contributing to Rune

Thank you for your interest in contributing to **Rune**! We welcome bug reports, documentation improvements, and thoughtful feature proposals.

---

## 1. Guiding Principles

Before proposing or implementing changes, please review our core philosophy:

- **Zero Bloat**: Rune is intentionally simple, predictable, and opinionated.
- **Pure Go**: We maintain zero external runtime dependencies.
- **CLI-First**: Every task is a real CLI command with standard arguments, flags, and help output.
- **Explicit over Magic**: No complex configuration DSLs or speculative plugin systems.

If a proposed feature does not directly improve task definition, discovery, invocation, safety, or execution, it probably does not belong in Rune.

---

## 2. Development Setup

### Prerequisites

- **Go**: 1.26 or higher
- **golangci-lint**: 2.13.2 or higher
- **Git**

### Clone the Repository

```bash
git clone git@github.com:octopyid/rune.git
cd rune
```

### Self-Hosting with `Runefile`

Rune manages its own development lifecycle using its own `Runefile`:

| Command | Action |
| :--- | :--- |
| `rune build` | Build the binary at `./bin/rune` |
| `rune test` | Run the complete test suite with race detector |
| `rune test:coverage` | Run tests and generate `coverage.out` |
| `rune lint` | Run static analysis and linter (`golangci-lint`) |
| `rune lint:fix` | Automatically fix formatting and mechanical lint issues |
| `rune fmt` | Format code using `goimports` and `gofmt` |
| `rune clean` | Remove build artifacts and coverage files |

If you do not have `rune` installed yet, you can run standard Go commands:

```bash
# Build
go build -ldflags "-s -w" -o ./bin/rune ./cmd/rune

# Test
go test -race -v ./...

# Lint
golangci-lint run
```

---

## 3. Pull Request Guidelines

1. **Open an Issue First**: For non-trivial features or design changes, please open an issue to discuss your proposal before submitting a PR.
2. **Branch from `main`**:
   ```bash
   git checkout -b feature/your-feature-name
   ```
3. **Write Tests**:
   - Every new feature, bug fix, or parser change should have accompanying tests.
   - Unit tests belong alongside the package (`*_test.go`).
   - CLI execution and integration tests belong in `test/e2e_test.go`.
4. **Run Pre-Flight Checks**:
   Ensure all tests pass and there are zero linter warnings:
   ```bash
   go test -race ./...
   golangci-lint run
   test -z "$(gofmt -l .)"
   ```
5. **Conventional Commits**:
   Please follow the [Conventional Commits](https://www.conventionalcommits.org/) convention:
   - `feat(scope): message`
   - `fix(scope): message`
   - `docs(scope): message`
   - `test(scope): message`
   - `refactor(scope): message`
   - `ci: message`

---

## 4. Code Style

- Format all Go files with standard `gofmt` and `goimports`.
- Keep functions concise, focused, and well-documented.
- Error messages should be lowercase and without trailing punctuation (standard Go style).
- Console UI output must follow the established Laravel-identical badge styling (`internal/ui`).

Thank you for helping make Rune better for everyone!
