package executor

import (
	"bytes"
	"strings"
	"testing"

	"github.com/octopyid/rune/internal/ast"
	"github.com/octopyid/rune/internal/cli"
)

func TestExpandCommand(t *testing.T) {
	task := &ast.Task{
		Name: "test",
		Parameters: []ast.Parameter{
			{Name: "name"},
		},
		Flags: []ast.Flag{
			{Name: "race"},
		},
		Passthrough: &ast.Passthrough{Name: "args"},
	}

	bound := &cli.BoundArgs{
		Task: task,
		Arguments: map[string]string{
			"name": "Supian",
		},
		Flags: map[string]bool{
			"race": true,
		},
		PassthroughArgs: []string{"make:model", "User Profile", "--force"},
	}

	// 1. Argument inside quotes
	argv, err := ExpandCommand(`echo "Hello, {{name}}!"`, bound)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(argv) != 2 || argv[0] != "echo" || argv[1] != "Hello, Supian!" {
		t.Errorf("expected [echo 'Hello, Supian!'], got %+v", argv)
	}

	// 2. Passthrough argument slice
	argv, err = ExpandCommand("php artisan {{args}}", bound)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{"php", "artisan", "make:model", "User Profile", "--force"}
	if len(argv) != len(expected) {
		t.Fatalf("expected %d args, got %d: %+v", len(expected), len(argv), argv)
	}
	for i, exp := range expected {
		if argv[i] != exp {
			t.Errorf("arg %d: expected %s, got %s", i, exp, argv[i])
		}
	}

	// 3. Standalone flag placeholder enabled
	argv, err = ExpandCommand("go build {{race}} ./...", bound)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected = []string{"go", "build", "--race", "./..."}
	if len(argv) != len(expected) {
		t.Fatalf("expected %d args, got %d: %+v", len(expected), len(argv), argv)
	}
	for i, exp := range expected {
		if argv[i] != exp {
			t.Errorf("arg %d: expected %s, got %s", i, exp, argv[i])
		}
	}

	// 4. Standalone flag placeholder disabled
	bound.Flags["race"] = false
	argv, err = ExpandCommand("go build {{race}} ./...", bound)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected = []string{"go", "build", "./..."}
	if len(argv) != len(expected) {
		t.Fatalf("expected %d args, got %d: %+v", len(expected), len(argv), argv)
	}
	for i, exp := range expected {
		if argv[i] != exp {
			t.Errorf("arg %d: expected %s, got %s", i, exp, argv[i])
		}
	}
}

func TestConfirmExecution(t *testing.T) {
	task1 := &ast.Task{Name: "task1"}
	task2 := &ast.Task{Name: "task2", Confirmation: "Delete all data?"}
	tasks := []*ast.Task{task1, task2}

	// 1. Bypass with yes
	var out bytes.Buffer
	err := ConfirmExecution(tasks, true, strings.NewReader(""), &out)
	if err != nil {
		t.Errorf("expected nil error on bypass, got %v", err)
	}

	// 2. User confirms with 'y'
	out.Reset()
	err = ConfirmExecution(tasks, false, strings.NewReader("y\n"), &out)
	if err != nil {
		t.Errorf("expected nil error on confirm 'y', got %v", err)
	}
	if !strings.Contains(out.String(), "Delete all data?") {
		t.Errorf("expected prompt in output, got %s", out.String())
	}

	// 3. User rejects with 'n'
	out.Reset()
	err = ConfirmExecution(tasks, false, strings.NewReader("n\n"), &out)
	if err != ErrCancelled {
		t.Errorf("expected ErrCancelled, got %v", err)
	}
	if !strings.Contains(out.String(), "Operation cancelled.") {
		t.Errorf("expected cancel message, got %s", out.String())
	}
}

func TestFormatDryRun(t *testing.T) {
	task1 := &ast.Task{Name: "build", Commands: []string{"go build ./..."}}
	task2 := &ast.Task{Name: "test", Commands: []string{"go test ./..."}}
	plan := []*ast.Task{task1, task2}

	bound := &cli.BoundArgs{Task: task2}
	output, err := FormatDryRun(plan, task2, bound)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(output, "[dry-run] Execution plan for 'test':") {
		t.Errorf("missing header in dry run output: %s", output)
	}
	if !strings.Contains(output, "1. build") || !strings.Contains(output, "$ go build ./...") {
		t.Errorf("missing step 1 in dry run output: %s", output)
	}
	if !strings.Contains(output, "2. test") || !strings.Contains(output, "$ go test ./...") {
		t.Errorf("missing step 2 in dry run output: %s", output)
	}
}

func TestExecutePlanSuccessAndFailFast(t *testing.T) {
	// 1. Success execution
	taskSuccess := &ast.Task{
		Name:     "ok",
		Commands: []string{"echo hello"},
	}
	bound := &cli.BoundArgs{Task: taskSuccess}

	var stdout, stderr bytes.Buffer
	ctx := ExecutionContext{
		Plan:       []*ast.Task{taskSuccess},
		TargetTask: taskSuccess,
		BoundArgs:  bound,
		Stdout:     &stdout,
		Stderr:     &stderr,
	}

	code := ExecutePlan(ctx)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d. stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "hello") {
		t.Errorf("expected stdout 'hello', got %s", stdout.String())
	}

	// 2. Fail-fast on dependency failure
	depFail := &ast.Task{
		Name:     "dep",
		Commands: []string{"sh -c 'exit 42'"},
	}
	mainTask := &ast.Task{
		Name:     "main",
		Commands: []string{"echo should_not_run"},
	}
	stdout.Reset()
	stderr.Reset()

	ctxFail := ExecutionContext{
		Plan:       []*ast.Task{depFail, mainTask},
		TargetTask: mainTask,
		BoundArgs:  &cli.BoundArgs{Task: mainTask},
		Stdout:     &stdout,
		Stderr:     &stderr,
	}

	codeFail := ExecutePlan(ctxFail)
	if codeFail != 42 {
		t.Errorf("expected exit code 42, got %d", codeFail)
	}
	if strings.Contains(stdout.String(), "should_not_run") {
		t.Errorf("main task should not have run after dependency failure! stdout: %s", stdout.String())
	}
}

func TestExecutePlanStdin(t *testing.T) {
	taskStdin := &ast.Task{
		Name:     "read_input",
		Commands: []string{"sh -c 'read line; echo \"RECEIVED: $line\"'"},
	}
	bound := &cli.BoundArgs{Task: taskStdin}

	var stdout, stderr bytes.Buffer
	ctx := ExecutionContext{
		Plan:       []*ast.Task{taskStdin},
		TargetTask: taskStdin,
		BoundArgs:  bound,
		Stdin:      strings.NewReader("flutter_screenshot_s\n"),
		Stdout:     &stdout,
		Stderr:     &stderr,
	}

	code := ExecutePlan(ctx)
	if code != 0 {
		t.Errorf("expected exit code 0, got %d. stderr: %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "RECEIVED: flutter_screenshot_s") {
		t.Errorf("expected stdin to be received by child, got: %s", stdout.String())
	}
}

func TestExitStatusSignal(t *testing.T) {
	taskSignal := &ast.Task{
		Name:     "sig",
		Commands: []string{"sh -c 'kill -INT $$'"},
	}
	bound := &cli.BoundArgs{Task: taskSignal}

	var stdout, stderr bytes.Buffer
	ctx := ExecutionContext{
		Plan:       []*ast.Task{taskSignal},
		TargetTask: taskSignal,
		BoundArgs:  bound,
		Stdout:     &stdout,
		Stderr:     &stderr,
	}

	code := ExecutePlan(ctx)
	if code != 130 {
		t.Errorf("expected exit code 130 for SIGINT, got %d", code)
	}
}
