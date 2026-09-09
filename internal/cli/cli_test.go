package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/octopyid/rune/internal/ast"
)

func TestSuggest(t *testing.T) {
	candidates := []string{"db:fresh", "db:migrate", "db:seed", "build", "test", "release"}

	if s := Suggest("db:frehs", candidates); s != "db:fresh" {
		t.Errorf("expected 'db:fresh', got '%s'", s)
	}

	if s := Suggest("buidl", candidates); s != "build" {
		t.Errorf("expected 'build', got '%s'", s)
	}

	if s := Suggest("completely_different", candidates); s != "" {
		t.Errorf("expected empty suggestion, got '%s'", s)
	}
}

func TestBindTaskArgs(t *testing.T) {
	// build target="dev" --race?:
	task := &ast.Task{
		Name: "build",
		Parameters: []ast.Parameter{
			{Name: "target", DefaultValue: "dev", HasDefault: true},
		},
		Flags: []ast.Flag{
			{Name: "race"},
		},
	}

	// 1. Defaults when no args given
	bound, err := BindTaskArgs(task, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bound.Arguments["target"] != "dev" {
		t.Errorf("expected target=dev, got %s", bound.Arguments["target"])
	}
	if bound.Flags["race"] {
		t.Error("expected race=false")
	}

	// 2. Provided value + flag
	bound, err = BindTaskArgs(task, []string{"production", "--race"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bound.Arguments["target"] != "production" {
		t.Errorf("expected target=production, got %s", bound.Arguments["target"])
	}
	if !bound.Flags["race"] {
		t.Error("expected race=true")
	}

	// 3. Flag before positional
	bound, err = BindTaskArgs(task, []string{"--race", "staging"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bound.Arguments["target"] != "staging" {
		t.Errorf("expected target=staging, got %s", bound.Arguments["target"])
	}
	if !bound.Flags["race"] {
		t.Error("expected race=true")
	}

	// 4. Unknown option typo
	_, err = BindTaskArgs(task, []string{"--rce"})
	if err == nil {
		t.Fatal("expected unknown option error, got nil")
	}
	if !strings.Contains(err.Error(), "Did you mean:\n  --race") {
		t.Errorf("expected suggestion '--race', got: %v", err)
	}
}

func TestBindPassthroughArgs(t *testing.T) {
	// artisan *args:
	task := &ast.Task{
		Name:        "artisan",
		Passthrough: &ast.Passthrough{Name: "args"},
	}

	raw := []string{"make:model", "User Profile", "--force"}
	bound, err := BindTaskArgs(task, raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(bound.PassthroughArgs) != 3 {
		t.Fatalf("expected 3 passthrough args, got %d", len(bound.PassthroughArgs))
	}
	if bound.PassthroughArgs[1] != "User Profile" {
		t.Errorf("expected 'User Profile' as single argument, got '%s'", bound.PassthroughArgs[1])
	}
}

func TestRouteVersion(t *testing.T) {
	ctx := RunContext{Args: []string{"--version"}}
	action := Route(ctx)
	if action.Kind != ActionVersion {
		t.Errorf("expected ActionVersion, got %v", action.Kind)
	}
	if !strings.Contains(action.Output, Version) {
		t.Errorf("expected version %s in output, got %s", Version, action.Output)
	}
}

func TestRouteNamespaceHelp(t *testing.T) {
	tmpDir := t.TempDir()
	rf := `
db:fresh:
    echo fresh
db:migrate:
    echo migrate
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Runefile"), []byte(rf), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := RunContext{
		Args:       []string{"db"},
		WorkingDir: tmpDir,
	}
	action := Route(ctx)
	if action.Kind != ActionHelp {
		t.Fatalf("expected ActionHelp, got %v", action.Kind)
	}
	if !strings.Contains(action.Output, "db:<command>") {
		t.Errorf("expected namespace help, got: %s", action.Output)
	}
	if !strings.Contains(action.Output, "db:fresh") || !strings.Contains(action.Output, "db:migrate") {
		t.Errorf("expected tasks in namespace help, got: %s", action.Output)
	}
}

func TestRouteUnknownTaskSuggestion(t *testing.T) {
	tmpDir := t.TempDir()
	rf := `
db:fresh:
    echo fresh
`
	if err := os.WriteFile(filepath.Join(tmpDir, "Runefile"), []byte(rf), 0644); err != nil {
		t.Fatal(err)
	}

	ctx := RunContext{
		Args:       []string{"db:frehs"},
		WorkingDir: tmpDir,
	}
	action := Route(ctx)
	if action.Err == nil {
		t.Fatal("expected error for unknown task, got nil")
	}
	if !strings.Contains(action.Err.Error(), "Did you mean:\n  db:fresh") {
		t.Errorf("expected suggestion db:fresh, got: %v", action.Err)
	}
}

func TestCompletion(t *testing.T) {
	taskBuild := &ast.Task{Name: "build", Flags: []ast.Flag{{Name: "race"}}}
	taskDbFresh := &ast.Task{Name: "db:fresh", Namespace: "db", ShortName: "fresh"}
	file := ast.NewFile("Runefile", []*ast.Task{taskBuild, taskDbFresh})

	// 1. Initial words: should list tasks, namespaces, and flags
	cands := Complete(file, []string{"rune", ""})
	hasBuild := false
	hasDb := false
	for _, c := range cands {
		if c == "build" {
			hasBuild = true
		}
		if c == "db" {
			hasDb = true
		}
	}
	if !hasBuild || !hasDb {
		t.Errorf("expected build and db in candidates, got %+v", cands)
	}

	// 2. Task specified: should list task flags
	cands = Complete(file, []string{"rune", "build", "--r"})
	if len(cands) != 1 || cands[0] != "--race" {
		t.Errorf("expected [--race], got %+v", cands)
	}

	// 3. Shell scripts generation
	for _, sh := range []string{"bash", "zsh", "fish"} {
		script, err := GenerateCompletion(sh)
		if err != nil || script == "" {
			t.Errorf("failed generating completion for %s: %v", sh, err)
		}
	}
}

func TestJoinLines(t *testing.T) {
	out := joinLines([]string{"a", "b", "c"})
	if out != "a\nb\nc" {
		t.Errorf("expected 'a\\nb\\nc', got %q", out)
	}
}

func TestRouteBuiltinUIComponents(t *testing.T) {
	// 1. rune info "hello world"
	ctx := RunContext{Args: []string{"info", "hello", "world"}}
	action := Route(ctx)
	if action.ExitCode != 0 {
		t.Errorf("expected exit code 0 for info, got %d", action.ExitCode)
	}
	if !strings.Contains(action.Output, "INFO") || !strings.Contains(action.Output, "hello world") {
		t.Errorf("expected INFO and message in output, got %q", action.Output)
	}

	// 2. rune error "failed task"
	ctx = RunContext{Args: []string{"error", "failed", "task"}}
	action = Route(ctx)
	if action.ExitCode != 1 {
		t.Errorf("expected exit code 1 for error, got %d", action.ExitCode)
	}
	if !strings.Contains(action.Output, "ERROR") || !strings.Contains(action.Output, "failed task") {
		t.Errorf("expected ERROR and message in output, got %q", action.Output)
	}

	// 3. rune help info
	ctx = RunContext{Args: []string{"help", "info"}}
	action = Route(ctx)
	if action.Kind != ActionHelp {
		t.Errorf("expected ActionHelp for help info, got %v", action.Kind)
	}
	if !strings.Contains(action.Output, "Display an informational message badge") {
		t.Errorf("expected info task description, got %s", action.Output)
	}

	// 4. rune info --help
	ctx = RunContext{Args: []string{"info", "--help"}}
	action = Route(ctx)
	if action.Kind != ActionHelp {
		t.Errorf("expected ActionHelp for info --help, got %v", action.Kind)
	}

	// 5. rune help nonexistent with suggestion
	ctx = RunContext{Args: []string{"help", "inf"}}
	action = Route(ctx)
	if action.Err == nil {
		t.Fatal("expected error for help inf, got nil")
	}
	if !strings.Contains(action.Err.Error(), "Did you mean:\n  info") {
		t.Errorf("expected suggestion 'info', got: %v", action.Err)
	}
}
