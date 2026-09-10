package test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var runeBin string

func TestMain(m *testing.M) {
	// Build rune binary into a temporary location for tests
	tmpDir, err := os.MkdirTemp("", "rune-e2e-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		os.Exit(1)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	runeBin = filepath.Join(tmpDir, "rune")
	cmd := exec.Command("go", "build", "-o", runeBin, "../cmd/rune")
	if out, err := cmd.CombinedOutput(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to build rune binary: %v\n%s\n", err, out)
		os.Exit(1)
	}

	os.Exit(m.Run())
}

func runRune(dir string, stdin string, args ...string) (string, string, int) {
	cmd := exec.Command(runeBin, args...)
	cmd.Dir = dir
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			exitCode = 1
		}
	}

	return stdout.String(), stderr.String(), exitCode
}

const sampleRunefile = `
#[Build the application]
build target="dev" --race?:
    echo "BUILD: target={{target}} race={{race}}"

#[Run tests]
test: build
    echo "TEST: running tests"

#[Reset the database]
#[confirm: This will permanently delete all database data.]
db:fresh:
    echo "DB_FRESH: deleted and migrated"

#[Laravel command]
artisan *args:
    echo "ARTISAN: {{args}}"

greet name:
    echo "Hello, {{name}}!"

release: build test
    echo "RELEASE: done"

failing_dep:
    sh -c "exit 33"

broken_chain: failing_dep
    echo "SHOULD_NOT_EXECUTE"

destructive_dep:
    echo "DESTROYED"

#[confirm: Confirm dangerous task?]
dangerous: destructive_dep
    echo "DANGEROUS_DONE"

env_check:
    sh -c "echo ENV_TEST=$ENV_TEST RUNE_TASK=$RUNE_TASK"
`

func setupFixture(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()
	rfPath := filepath.Join(tmpDir, "Runefile")
	if err := os.WriteFile(rfPath, []byte(sampleRunefile), 0644); err != nil {
		t.Fatalf("failed to write Runefile: %v", err)
	}
	return tmpDir
}

func TestE2ERootHelp(t *testing.T) {
	dir := setupFixture(t)
	stdout, _, code := runRune(dir, "", "--help")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	for _, expected := range []string{
		"Rune",
		"Usage:",
		"Options:",
		"Available commands:",
		"db",
		"build",
		"test",
		"greet",
		"release",
	} {
		if !strings.Contains(stdout, expected) {
			t.Errorf("expected root help to contain %q, got:\n%s", expected, stdout)
		}
	}
}

func TestE2ENamespaceHelp(t *testing.T) {
	dir := setupFixture(t)
	// 1. rune db
	stdout, _, code := runRune(dir, "", "db")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(stdout, "db:<command>") || !strings.Contains(stdout, "db:fresh") {
		t.Errorf("unexpected namespace help:\n%s", stdout)
	}

	// 2. rune db --help
	stdout, _, code = runRune(dir, "", "db", "--help")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(stdout, "db:<command>") {
		t.Errorf("unexpected namespace help:\n%s", stdout)
	}
}

func TestE2ETaskHelp(t *testing.T) {
	dir := setupFixture(t)
	stdout, _, code := runRune(dir, "", "build", "--help")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	for _, exp := range []string{
		"Description:\n  Build the application",
		"Usage:\n  build [options]",
		"Arguments:\n  target",
		"Options:\n      --race",
	} {
		if !strings.Contains(stdout, exp) {
			t.Errorf("expected task help to contain %q, got:\n%s", exp, stdout)
		}
	}
}

func TestE2EArgumentsAndDefaults(t *testing.T) {
	dir := setupFixture(t)

	// 1. Required arg provided
	stdout, _, code := runRune(dir, "", "greet", "Supian")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(stdout, "Hello, Supian!") {
		t.Errorf("expected 'Hello, Supian!', got %q", stdout)
	}

	// 2. Required arg missing
	_, stderr, code := runRune(dir, "", "greet")
	if code == 0 {
		t.Fatalf("expected non-zero code for missing arg")
	}
	if !strings.Contains(stderr, "Missing argument: name") || !strings.Contains(stderr, "rune greet <name>") {
		t.Errorf("unexpected missing arg error:\n%s", stderr)
	}

	// 3. Default argument fallback
	stdout, _, code = runRune(dir, "", "build")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(stdout, "BUILD: target=dev race=") {
		t.Errorf("expected target=dev and empty race, got %q", stdout)
	}

	// 4. Default argument overridden
	stdout, _, code = runRune(dir, "", "build", "production")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(stdout, "BUILD: target=production race=") {
		t.Errorf("expected target=production, got %q", stdout)
	}

	// 5. Boolean flag enabled
	stdout, _, code = runRune(dir, "", "build", "production", "--race")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(stdout, "BUILD: target=production race=--race") {
		t.Errorf("expected race=--race, got %q", stdout)
	}

	// 6. Boolean flag before positional
	stdout, _, code = runRune(dir, "", "build", "--race", "staging")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(stdout, "BUILD: target=staging race=--race") {
		t.Errorf("expected target=staging race=--race, got %q", stdout)
	}
}

func TestE2EUnknownTaskAndOptionSuggestions(t *testing.T) {
	dir := setupFixture(t)

	// 1. Unknown task typo
	_, stderr, code := runRune(dir, "", "db:frehs")
	if code == 0 {
		t.Fatal("expected failure on unknown task")
	}
	if !strings.Contains(stderr, "Unknown task: db:frehs") || !strings.Contains(stderr, "Did you mean:\n  db:fresh") {
		t.Errorf("expected suggestion for db:fresh, got:\n%s", stderr)
	}

	// 2. Unknown option typo
	_, stderr, code = runRune(dir, "", "build", "--rce")
	if code == 0 {
		t.Fatal("expected failure on unknown option")
	}
	if !strings.Contains(stderr, "Unknown option: --rce") || !strings.Contains(stderr, "Did you mean:\n  --race") {
		t.Errorf("expected suggestion for --race, got:\n%s", stderr)
	}
}

func TestE2EPassthroughArguments(t *testing.T) {
	dir := setupFixture(t)

	// rune artisan make:model "User Profile" --force
	stdout, _, code := runRune(dir, "", "artisan", "make:model", "User Profile", "--force")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(stdout, "ARTISAN: make:model User Profile --force") {
		t.Errorf("expected passthrough args, got %q", stdout)
	}
}

func TestE2EDependenciesAndDeduplication(t *testing.T) {
	dir := setupFixture(t)

	// release: build test (where test: build)
	// build must run only ONCE, followed by test, followed by release
	stdout, _, code := runRune(dir, "", "release")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}

	buildIdx := strings.Index(stdout, "BUILD: target=dev")
	testIdx := strings.Index(stdout, "TEST: running tests")
	releaseIdx := strings.Index(stdout, "RELEASE: done")

	if buildIdx == -1 || testIdx == -1 || releaseIdx == -1 {
		t.Fatalf("missing task output in:\n%s", stdout)
	}

	if buildIdx >= testIdx || testIdx >= releaseIdx {
		t.Errorf("tasks did not execute in correct order: build=%d, test=%d, release=%d", buildIdx, testIdx, releaseIdx)
	}

	// Count occurrences of build output - must be exactly 1
	if strings.Count(stdout, "BUILD: target=dev") != 1 {
		t.Errorf("expected build to execute exactly once, but ran %d times", strings.Count(stdout, "BUILD: target=dev"))
	}
}

func TestE2EFailFast(t *testing.T) {
	dir := setupFixture(t)

	// broken_chain: failing_dep (which exits with 33)
	stdout, _, code := runRune(dir, "", "broken_chain")
	if code != 33 {
		t.Fatalf("expected exit code 33, got %d", code)
	}
	if strings.Contains(stdout, "SHOULD_NOT_EXECUTE") {
		t.Errorf("dependent task executed despite dependency failure!")
	}
}

func TestE2EConfirmation(t *testing.T) {
	dir := setupFixture(t)

	// 1. Bypass with --yes
	stdout, _, code := runRune(dir, "", "--yes", "db:fresh")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(stdout, "DB_FRESH: deleted and migrated") {
		t.Errorf("expected task to execute with --yes, got %q", stdout)
	}

	// 2. Reject confirmation with 'n'
	stdout, stderr, code := runRune(dir, "n\n", "db:fresh")
	if code == 0 {
		t.Fatal("expected rejection to exit with non-zero")
	}
	if strings.Contains(stdout, "DB_FRESH") {
		t.Errorf("command executed despite rejection!")
	}
	if !strings.Contains(stdout, "Operation cancelled.") && !strings.Contains(stderr, "Operation cancelled.") {
		t.Errorf("expected cancellation message")
	}

	// 3. Accept confirmation with 'y'
	stdout, _, code = runRune(dir, "y\n", "db:fresh")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(stdout, "DB_FRESH: deleted and migrated") {
		t.Errorf("expected execution on 'y', got %q", stdout)
	}

	// 4. Confirmation happens BEFORE dependencies run!
	// dangerous: destructive_dep. If dangerous is rejected, destructive_dep must NOT run!
	stdout, _, code = runRune(dir, "n\n", "dangerous")
	if code == 0 {
		t.Fatal("expected non-zero on rejection")
	}
	if strings.Contains(stdout, "DESTROYED") {
		t.Errorf("dependency executed BEFORE confirmation was accepted!")
	}
}

func TestE2EDryRun(t *testing.T) {
	dir := setupFixture(t)

	// 1. Dry run on dependency chain
	stdout, _, code := runRune(dir, "", "--dry-run", "release")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	for _, exp := range []string{
		"[dry-run] Execution plan for 'release':",
		"1. build",
		"$ echo \"BUILD: target=dev race=\"",
		"2. test",
		"$ echo \"TEST: running tests\"",
		"3. release",
		"$ echo \"RELEASE: done\"",
	} {
		if !strings.Contains(stdout, exp) {
			t.Errorf("expected dry run to contain %q, got:\n%s", exp, stdout)
		}
	}

	// 2. Dry run with confirmation: no prompt asked, commands not executed
	stdout, _, code = runRune(dir, "", "--dry-run", "db:fresh")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(stdout, "[dry-run] Execution plan for 'db:fresh':") {
		t.Errorf("expected dry-run plan, got:\n%s", stdout)
	}
	if strings.Contains(stdout, "Continue? [y/N]") {
		t.Errorf("dry-run should not prompt for confirmation!")
	}
}

func TestE2EDotEnv(t *testing.T) {
	dir := setupFixture(t)
	// Create .env file
	envFile := filepath.Join(dir, ".env")
	if err := os.WriteFile(envFile, []byte("ENV_TEST=loaded_from_dotenv\n"), 0644); err != nil {
		t.Fatal(err)
	}

	stdout, _, code := runRune(dir, "", "env_check")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(stdout, "ENV_TEST=loaded_from_dotenv") {
		t.Errorf("expected .env variable in child process, got %q", stdout)
	}
	if !strings.Contains(stdout, "RUNE_TASK=env_check") {
		t.Errorf("expected RUNE_TASK=env_check, got %q", stdout)
	}
}

func TestE2EStdinForwarding(t *testing.T) {
	dir := setupFixture(t)
	rfPath := filepath.Join(dir, "Runefile")
	extraTask := "\nread_task:\n    sh -c \"read line; echo GOT: $line\"\n"
	f, err := os.OpenFile(rfPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString(extraTask)
	_ = f.Close()

	stdout, _, code := runRune(dir, "flutter_screenshot_s\n", "read_task")
	if code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(stdout, "GOT: flutter_screenshot_s") {
		t.Errorf("expected child process to receive stdin, got: %s", stdout)
	}
}

func TestE2ESignalExitCode(t *testing.T) {
	dir := setupFixture(t)
	rfPath := filepath.Join(dir, "Runefile")
	extraTask := "\nsig_task:\n    sh -c \"kill -INT $$\"\n"
	f, err := os.OpenFile(rfPath, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = f.WriteString(extraTask)
	_ = f.Close()

	_, _, code := runRune(dir, "", "sig_task")
	if code != 130 {
		t.Errorf("expected exit code 130 for SIGINT, got %d", code)
	}
}
