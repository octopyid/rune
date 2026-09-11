package executor

import (
	"bytes"
	"os"
	"path/filepath"
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

	// 5. Valued option expansion
	taskWithValued := &ast.Task{
		Name: "build",
		Flags: []ast.Flag{
			{Short: "o", Name: "output", IsValued: true},
		},
	}
	boundValued := &cli.BoundArgs{
		Task: taskWithValued,
		Arguments: map[string]string{
			"output": "bin/app",
		},
	}
	argv, err = ExpandCommand("go build -o {{output}} ./...", boundValued)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected = []string{"go", "build", "-o", "bin/app", "./..."}
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

func TestExecutePlanDir(t *testing.T) {
	tmpDir := t.TempDir()
	frontDir := filepath.Join(tmpDir, "frontend")
	nestedDir := filepath.Join(tmpDir, "nested", "sub")
	if err := os.MkdirAll(frontDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatal(err)
	}

	// 1. Task without dir executes in ctx.WorkingDir
	taskRoot := &ast.Task{
		Name:     "root_task",
		Commands: []string{"pwd"},
	}
	var stdout bytes.Buffer
	ctxRoot := ExecutionContext{
		Plan:       []*ast.Task{taskRoot},
		TargetTask: taskRoot,
		BoundArgs:  &cli.BoundArgs{Task: taskRoot},
		WorkingDir: tmpDir,
		Stdout:     &stdout,
	}
	if code := ExecutePlan(ctxRoot); code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	// On macOS /var can be a symlink to /private/var, so resolve paths
	realTmp, _ := filepath.EvalSymlinks(tmpDir)
	realOut, _ := filepath.EvalSymlinks(strings.TrimSpace(stdout.String()))
	if realOut != realTmp {
		t.Errorf("expected root task to run in %q, got %q", realTmp, realOut)
	}

	// 2. Relative dir
	taskFront := &ast.Task{
		Name:     "front_task",
		Dir:      "frontend",
		Commands: []string{"pwd"},
	}
	stdout.Reset()
	ctxFront := ExecutionContext{
		Plan:       []*ast.Task{taskFront},
		TargetTask: taskFront,
		BoundArgs:  &cli.BoundArgs{Task: taskFront},
		WorkingDir: tmpDir,
		Stdout:     &stdout,
	}
	if code := ExecutePlan(ctxFront); code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	realFront, _ := filepath.EvalSymlinks(frontDir)
	realOut, _ = filepath.EvalSymlinks(strings.TrimSpace(stdout.String()))
	if realOut != realFront {
		t.Errorf("expected front task to run in %q, got %q", realFront, realOut)
	}

	// 3. Nested dir
	taskNested := &ast.Task{
		Name:     "nested_task",
		Dir:      "nested/sub",
		Commands: []string{"pwd"},
	}
	stdout.Reset()
	ctxNested := ExecutionContext{
		Plan:       []*ast.Task{taskNested},
		TargetTask: taskNested,
		BoundArgs:  &cli.BoundArgs{Task: taskNested},
		WorkingDir: tmpDir,
		Stdout:     &stdout,
	}
	if code := ExecutePlan(ctxNested); code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	realNested, _ := filepath.EvalSymlinks(nestedDir)
	realOut, _ = filepath.EvalSymlinks(strings.TrimSpace(stdout.String()))
	if realOut != realNested {
		t.Errorf("expected nested task to run in %q, got %q", realNested, realOut)
	}

	// 4. Missing directory fails fast with error
	taskMissing := &ast.Task{
		Name:     "missing_task",
		Dir:      "nonexistent_dir",
		Commands: []string{"pwd"},
	}
	var stderr bytes.Buffer
	ctxMissing := ExecutionContext{
		Plan:       []*ast.Task{taskMissing},
		TargetTask: taskMissing,
		BoundArgs:  &cli.BoundArgs{Task: taskMissing},
		WorkingDir: tmpDir,
		Stderr:     &stderr,
	}
	if code := ExecutePlan(ctxMissing); code != 1 {
		t.Errorf("expected code 1 for missing dir, got %d", code)
	}
	if !strings.Contains(stderr.String(), "Directory 'nonexistent_dir' does not exist") {
		t.Errorf("expected missing dir error message, got: %s", stderr.String())
	}

	// 5. Dependency tasks with different directories
	taskDep := &ast.Task{
		Name:     "dep_task",
		Dir:      "frontend",
		Commands: []string{"pwd"},
	}
	taskMain := &ast.Task{
		Name:     "main_task",
		Dir:      "nested/sub",
		Commands: []string{"pwd"},
	}
	stdout.Reset()
	ctxMulti := ExecutionContext{
		Plan:       []*ast.Task{taskDep, taskMain},
		TargetTask: taskMain,
		BoundArgs:  &cli.BoundArgs{Task: taskMain},
		WorkingDir: tmpDir,
		Stdout:     &stdout,
	}
	if code := ExecutePlan(ctxMulti); code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("expected 2 output lines, got %d: %v", len(lines), lines)
	}
	outDep, _ := filepath.EvalSymlinks(lines[0])
	outMain, _ := filepath.EvalSymlinks(lines[1])
	if outDep != realFront {
		t.Errorf("dep directory mismatch: expected %q, got %q", realFront, outDep)
	}
	if outMain != realNested {
		t.Errorf("main directory mismatch: expected %q, got %q", realNested, outMain)
	}
}

func TestExecutePlanEnv(t *testing.T) {
	// Task A has custom env; Task B does not
	taskA := &ast.Task{
		Name:     "task_a",
		Env:      map[string]string{"CUSTOM_VAR": "value_a", "OVERRIDE": "task_override"},
		Commands: []string{"sh -c 'echo A_VAR=$CUSTOM_VAR OVERRIDE=$OVERRIDE'"},
	}
	taskB := &ast.Task{
		Name:     "task_b",
		Commands: []string{"sh -c 'echo B_VAR=$CUSTOM_VAR OVERRIDE=$OVERRIDE'"},
	}

	var stdout bytes.Buffer
	ctx := ExecutionContext{
		Plan:       []*ast.Task{taskA, taskB},
		TargetTask: taskB,
		BoundArgs:  &cli.BoundArgs{Task: taskB},
		DotEnv:     map[string]string{"OVERRIDE": "dotenv_val"},
		Stdout:     &stdout,
	}

	if code := ExecutePlan(ctx); code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}

	out := stdout.String()
	// Task A should have custom var and overridden value
	if !strings.Contains(out, "A_VAR=value_a OVERRIDE=task_override") {
		t.Errorf("expected Task A to have custom env and override dotenv, got:\n%s", out)
	}
	// Task B should NOT have CUSTOM_VAR, and OVERRIDE should fall back to dotenv
	if !strings.Contains(out, "B_VAR= OVERRIDE=dotenv_val") {
		t.Errorf("expected Task B to have isolated env without Task A variables, got:\n%s", out)
	}
}

func TestExecutePlanVerbose(t *testing.T) {
	task := &ast.Task{
		Name:     "hello",
		Commands: []string{`echo "hello world"`},
	}

	// 1. Verbose disabled by default
	var stdout1 bytes.Buffer
	ctx1 := ExecutionContext{
		Plan:       []*ast.Task{task},
		TargetTask: task,
		BoundArgs:  &cli.BoundArgs{Task: task},
		Verbose:    false,
		Stdout:     &stdout1,
	}
	if code := ExecutePlan(ctx1); code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if strings.Contains(stdout1.String(), "$") {
		t.Errorf("expected no command echo when verbose is disabled, got: %s", stdout1.String())
	}
	if !strings.Contains(stdout1.String(), "hello world") {
		t.Errorf("expected command output 'hello world', got: %s", stdout1.String())
	}

	// 2. Verbose enabled
	var stdout2 bytes.Buffer
	ctx2 := ExecutionContext{
		Plan:       []*ast.Task{task},
		TargetTask: task,
		BoundArgs:  &cli.BoundArgs{Task: task},
		Verbose:    true,
		Stdout:     &stdout2,
	}
	if code := ExecutePlan(ctx2); code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	if !strings.Contains(stdout2.String(), `$ echo "hello world"`) {
		t.Errorf("expected verbose command echo '$ echo \"hello world\"', got:\n%s", stdout2.String())
	}
	if !strings.Contains(stdout2.String(), "hello world") {
		t.Errorf("expected command output, got:\n%s", stdout2.String())
	}
}

func TestExecutePlanTime(t *testing.T) {
	task1 := &ast.Task{
		Name:     "build",
		Commands: []string{"echo building"},
	}
	task2 := &ast.Task{
		Name:     "test",
		Commands: []string{"echo testing"},
	}

	// 1. Single task timing
	var stdout1 bytes.Buffer
	ctx1 := ExecutionContext{
		Plan:       []*ast.Task{task1},
		TargetTask: task1,
		BoundArgs:  &cli.BoundArgs{Task: task1},
		Time:       true,
		Stdout:     &stdout1,
	}
	if code := ExecutePlan(ctx1); code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	out1 := stdout1.String()
	if !strings.Contains(out1, "✔ build (") || !strings.Contains(out1, "Total:") {
		t.Errorf("expected single task timing with checkmark and Total, got:\n%s", out1)
	}

	// 2. Multi-task dependencies timing
	var stdout2 bytes.Buffer
	ctx2 := ExecutionContext{
		Plan:       []*ast.Task{task1, task2},
		TargetTask: task2,
		BoundArgs:  &cli.BoundArgs{Task: task2},
		Time:       true,
		Stdout:     &stdout2,
	}
	if code := ExecutePlan(ctx2); code != 0 {
		t.Fatalf("expected code 0, got %d", code)
	}
	out2 := stdout2.String()
	if !strings.Contains(out2, "[1/2] build") || !strings.Contains(out2, "[2/2] test") || !strings.Contains(out2, "✔ Total:") {
		t.Errorf("expected multi-task timing report with [1/2] and [2/2], got:\n%s", out2)
	}

	// 3. Failed task timing
	taskFail := &ast.Task{
		Name:     "broken",
		Commands: []string{"sh -c 'exit 5'"},
	}
	var stdout3 bytes.Buffer
	ctx3 := ExecutionContext{
		Plan:       []*ast.Task{taskFail},
		TargetTask: taskFail,
		BoundArgs:  &cli.BoundArgs{Task: taskFail},
		Time:       true,
		Stdout:     &stdout3,
	}
	code3 := ExecutePlan(ctx3)
	if code3 != 5 {
		t.Errorf("expected exit code 5, got %d", code3)
	}
	out3 := stdout3.String()
	if !strings.Contains(out3, "✖ broken (") || !strings.Contains(out3, "Total:") {
		t.Errorf("expected failed task timing report with ✖ and Total, got:\n%s", out3)
	}
}
