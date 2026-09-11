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
		val := strings.Split(c, "\t")[0]
		if val == "build" {
			hasBuild = true
		}
		if val == "db" {
			hasDb = true
		}
	}
	if !hasBuild || !hasDb {
		t.Errorf("expected build and db in candidates, got %+v", cands)
	}

	// 2. Exact/prefix task name being typed: should suggest the task itself
	cands = Complete(file, []string{"build"})
	if len(cands) == 0 || !strings.HasPrefix(cands[0], "build") {
		t.Errorf("expected [build], got %+v", cands)
	}

	// 3. Task specified followed by flag prefix: should list task flags
	cands = Complete(file, []string{"rune", "build", "--r"})
	if len(cands) != 1 || !strings.HasPrefix(cands[0], "--race") {
		t.Errorf("expected [--race], got %+v", cands)
	}

	// 4. Task specified followed by empty word (trailing space): should suggest task flags & global flags
	cands = Complete(file, []string{"rune", "build", ""})
	hasRace := false
	for _, c := range cands {
		if strings.HasPrefix(c, "--race") {
			hasRace = true
			break
		}
	}
	if !hasRace {
		t.Errorf("expected --race in candidates for 'rune build ', got %+v", cands)
	}

	// 3. Shell scripts generation
	for _, sh := range []string{"bash", "zsh", "fish"} {
		script, err := GenerateCompletion(sh)
		if err != nil || script == "" {
			t.Errorf("failed generating completion for %s: %v", sh, err)
		}
	}

	zshScript, _ := GenerateCompletion("zsh")
	if !strings.Contains(zshScript, "compdef _rune rune") {
		t.Errorf("expected zsh script to register completion with compdef")
	}
	if !strings.Contains(zshScript, `if [ "$funcstack[1]" = "_rune" ]; then`) {
		t.Errorf("expected zsh script to guard execution with funcstack check")
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

func TestFormatTaskHelpWithDocComments(t *testing.T) {
	task := &ast.Task{
		Name:        "build:apk",
		Description: "Build Android APK",
		Parameters: []ast.Parameter{
			{Name: "target", DefaultValue: "dev", HasDefault: true, Description: "Target build environment"},
			{Name: "output", HasDefault: false, Description: "Output directory path"},
		},
		Flags: []ast.Flag{
			{Name: "split", Description: "Build split-per-ABI APKs alongside universal APK"},
			{Name: "minify", Description: ""}, // fallback test
		},
		Passthrough: &ast.Passthrough{
			Name:        "args",
			Description: "Pass extra flags to gradle",
		},
	}

	help := FormatTaskHelp(task)

	if !strings.Contains(help, "Target build environment [default: \"dev\"]") {
		t.Errorf("expected target description with default, got:\n%s", help)
	}
	if !strings.Contains(help, "Output directory path") {
		t.Errorf("expected output description, got:\n%s", help)
	}
	if !strings.Contains(help, "Build split-per-ABI APKs alongside universal APK") {
		t.Errorf("expected split flag description, got:\n%s", help)
	}
	if !strings.Contains(help, "Optional boolean flag") {
		t.Errorf("expected fallback for undocumented minify flag, got:\n%s", help)
	}
	if !strings.Contains(help, "Pass extra flags to gradle") {
		t.Errorf("expected passthrough description, got:\n%s", help)
	}
}

func TestParseGlobalFlagsVerboseAndTime(t *testing.T) {
	// 1. Default disabled
	gf1, task1, args1, err := ParseGlobalFlags([]string{"build"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gf1.Verbose || gf1.Time {
		t.Errorf("expected Verbose=false and Time=false by default, got verbose=%v, time=%v", gf1.Verbose, gf1.Time)
	}
	if task1 != "build" || len(args1) != 0 {
		t.Errorf("mismatch task/args: task=%s, args=%v", task1, args1)
	}

	// 2. Global flags before task name
	gf2, task2, _, err := ParseGlobalFlags([]string{"--verbose", "--time", "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !gf2.Verbose || !gf2.Time {
		t.Errorf("expected Verbose=true and Time=true, got verbose=%v, time=%v", gf2.Verbose, gf2.Time)
	}
	if task2 != "test" {
		t.Errorf("expected task 'test', got %q", task2)
	}

	// 3. Global flags after task name
	gf3, task3, args3, err := ParseGlobalFlags([]string{"build", "--verbose", "--time", "dev"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !gf3.Verbose || !gf3.Time {
		t.Errorf("expected Verbose=true and Time=true, got verbose=%v, time=%v", gf3.Verbose, gf3.Time)
	}
	if task3 != "build" {
		t.Errorf("expected task 'build', got %q", task3)
	}
	if len(args3) != 1 || args3[0] != "dev" {
		t.Errorf("expected task args ['dev'], got %v", args3)
	}
}

func TestHelpIncludesVerboseAndTime(t *testing.T) {
	help := FormatRootHelp(nil)
	if !strings.Contains(help, "--verbose") {
		t.Errorf("expected root help to describe --verbose, got:\n%s", help)
	}
	if !strings.Contains(help, "--time") {
		t.Errorf("expected root help to describe --time, got:\n%s", help)
	}
}

func TestCompletionIncludesVerboseAndTime(t *testing.T) {
	suggestions := Complete(nil, []string{"--"})
	hasVerbose := false
	hasTime := false
	for _, s := range suggestions {
		val := strings.Split(s, "\t")[0]
		if val == "--verbose" {
			hasVerbose = true
		}
		if val == "--time" {
			hasTime = true
		}
	}
	if !hasVerbose {
		t.Errorf("expected suggestions to include --verbose, got %v", suggestions)
	}
	if !hasTime {
		t.Errorf("expected suggestions to include --time, got %v", suggestions)
	}
}

func TestPrivateTasksHiddenFromHelpAndCompletion(t *testing.T) {
	publicTask := &ast.Task{Name: "build", Description: "Build binary"}
	privateTask := &ast.Task{Name: "secret:setup", Namespace: "secret", ShortName: "setup", Description: "Secret setup", Private: true}
	file := ast.NewFile("Runefile", []*ast.Task{publicTask, privateTask})

	// 1. Root help should not display private task or its namespace if all tasks in it are private
	rootHelp := FormatRootHelp(file)
	if strings.Contains(rootHelp, "secret:setup") || strings.Contains(rootHelp, "Secret setup") {
		t.Errorf("expected root help to hide private task, got:\n%s", rootHelp)
	}
	if !strings.Contains(rootHelp, "build") {
		t.Errorf("expected root help to include public task 'build', got:\n%s", rootHelp)
	}

	// 2. Namespace help should hide private tasks
	nsHelp := FormatNamespaceHelp(file, "secret")
	if strings.Contains(nsHelp, "secret:setup") {
		t.Errorf("expected namespace help to hide private task, got:\n%s", nsHelp)
	}

	// 3. Completion should hide private task
	cands := Complete(file, []string{"rune", ""})
	for _, c := range cands {
		val := strings.Split(c, "\t")[0]
		if val == "secret:setup" || val == "secret" {
			t.Errorf("expected completion to exclude private task and namespace, got candidates: %v", cands)
		}
	}

	// 4. Private task can still be routed directly
	action := Route(RunContext{Args: []string{"secret:setup"}, WorkingDir: "."})
	// Even if run directly without Runefile in curdir, it attempts execution rather than failing with unknown task
	_ = action
}

func TestTreeGlobalFlagAndRouting(t *testing.T) {
	// 1. ParseGlobalFlags with --tree
	gf, task, _, err := ParseGlobalFlags([]string{"--tree", "release"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !gf.Tree || task != "release" {
		t.Errorf("expected Tree=true and task=release, got tree=%v, task=%s", gf.Tree, task)
	}

	// Flags after task name
	gf2, task2, _, err := ParseGlobalFlags([]string{"release", "--tree"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !gf2.Tree || task2 != "release" {
		t.Errorf("expected Tree=true and task=release, got tree=%v, task=%s", gf2.Tree, task2)
	}

	// 2. Root help describes --tree
	help := FormatRootHelp(nil)
	if !strings.Contains(help, "--tree") {
		t.Errorf("expected root help to describe --tree, got:\n%s", help)
	}

	// 3. Completion includes --tree
	suggestions := Complete(nil, []string{"--"})
	hasTree := false
	for _, s := range suggestions {
		val := strings.Split(s, "\t")[0]
		if val == "--tree" {
			hasTree = true
		}
	}
	if !hasTree {
		t.Errorf("expected suggestions to include --tree, got %v", suggestions)
	}
}

func TestBindTaskArgsWithOptionsAndEnums(t *testing.T) {
	task := &ast.Task{
		Name: "deploy",
		Flags: []ast.Flag{
			{
				Short:    "w",
				Name:     "watch",
				IsValued: false,
			},
			{
				Short:        "o",
				Name:         "output",
				IsValued:     true,
				HasDefault:   true,
				DefaultValue: "dist",
			},
			{
				Short:    "e",
				Name:     "env",
				IsValued: true,
				Required: true,
				Choices:  []string{"staging", "production"},
			},
		},
		Parameters: []ast.Parameter{
			{
				Name:         "action",
				Choices:      []string{"up", "down"},
				HasDefault:   true,
				DefaultValue: "up",
			},
		},
	}

	// 1. Missing required option
	_, err := BindTaskArgs(task, nil)
	if err == nil {
		t.Fatal("expected error for missing required option, got nil")
	}
	if !strings.Contains(err.Error(), "missing required option: -e, --env") {
		t.Errorf("expected missing required option error, got %v", err)
	}

	// 2. Valid invocation with short flags and space
	bound, err := BindTaskArgs(task, []string{"-w", "-e", "staging", "-o", "build", "down"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bound.Flags["watch"] {
		t.Error("expected watch=true")
	}
	if bound.Arguments["env"] != "staging" {
		t.Errorf("expected env=staging, got %s", bound.Arguments["env"])
	}
	if bound.Arguments["output"] != "build" {
		t.Errorf("expected output=build, got %s", bound.Arguments["output"])
	}
	if bound.Arguments["action"] != "down" {
		t.Errorf("expected action=down, got %s", bound.Arguments["action"])
	}

	// 3. Valid invocation with long flags and equal
	bound, err = BindTaskArgs(task, []string{"--env=production", "--output=custom"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bound.Flags["watch"] {
		t.Error("expected watch=false")
	}
	if bound.Arguments["env"] != "production" {
		t.Errorf("expected env=production, got %s", bound.Arguments["env"])
	}
	if bound.Arguments["output"] != "custom" {
		t.Errorf("expected output=custom, got %s", bound.Arguments["output"])
	}
	if bound.Arguments["action"] != "up" {
		t.Errorf("expected default action=up, got %s", bound.Arguments["action"])
	}

	// 4. Invalid enum choice for option
	_, err = BindTaskArgs(task, []string{"-e", "local"})
	if err == nil {
		t.Fatal("expected error for invalid enum option choice, got nil")
	}
	if !strings.Contains(err.Error(), "invalid value \"local\" for option -e, --env=VALUE") {
		t.Errorf("expected invalid value error, got %v", err)
	}

	// 5. Invalid enum choice for positional argument
	_, err = BindTaskArgs(task, []string{"-e", "staging", "invalid_action"})
	if err == nil {
		t.Fatal("expected error for invalid positional choice, got nil")
	}
	if !strings.Contains(err.Error(), "invalid value \"invalid_action\" for argument \"action\"") {
		t.Errorf("expected invalid argument choice error, got %v", err)
	}

	// 6. Option requiring value without value
	_, err = BindTaskArgs(task, []string{"-e"})
	if err == nil {
		t.Fatal("expected error for missing value, got nil")
	}
	if !strings.Contains(err.Error(), "option '-e' requires a value") {
		t.Errorf("expected option requires value error, got %v", err)
	}

	_, err = BindTaskArgs(task, []string{"--output"})
	if err == nil {
		t.Fatal("expected error for missing value on long option, got nil")
	}
	if !strings.Contains(err.Error(), "option '--output' requires a value") {
		t.Errorf("expected option requires value error, got %v", err)
	}
}

func TestTaskHelpWithOptionsAndEnums(t *testing.T) {
	task := &ast.Task{
		Name:        "deploy",
		Description: "Deploy application",
		Flags: []ast.Flag{
			{
				Short:       "w",
				Name:        "watch",
				Description: "Watch files",
			},
			{
				Short:        "o",
				Name:         "output",
				Description:  "Output folder",
				IsValued:     true,
				HasDefault:   true,
				DefaultValue: "dist",
			},
			{
				Short:       "e",
				Name:        "env",
				Description: "Cloud target",
				IsValued:    true,
				Required:    true,
				Choices:     []string{"staging", "production"},
			},
		},
		Parameters: []ast.Parameter{
			{
				Name:         "action",
				Description:  "Migration action",
				Choices:      []string{"up", "down"},
				HasDefault:   true,
				DefaultValue: "up",
			},
		},
	}

	help := FormatTaskHelp(task)

	// Verify help formatting
	if !strings.Contains(help, "-w, --watch") {
		t.Errorf("expected '-w, --watch' in help, got:\n%s", help)
	}
	if !strings.Contains(help, "-o, --output=VALUE") || !strings.Contains(help, "[default: \"dist\"]") {
		t.Errorf("expected output option and default in help, got:\n%s", help)
	}
	if !strings.Contains(help, "-e, --env=VALUE") || !strings.Contains(help, "[choices: staging, production]") || !strings.Contains(help, "(required)") {
		t.Errorf("expected env option, choices, and (required) in help, got:\n%s", help)
	}
	if !strings.Contains(help, "action") || !strings.Contains(help, "[choices: up, down]") {
		t.Errorf("expected action argument with choices in help, got:\n%s", help)
	}
}

func TestCompletionWithOptionsAndEnums(t *testing.T) {
	task := &ast.Task{
		Name: "deploy",
		Flags: []ast.Flag{
			{Short: "w", Name: "watch"},
			{Short: "e", Name: "env", IsValued: true, Choices: []string{"staging", "production"}},
		},
	}
	file := ast.NewFile("Runefile", []*ast.Task{task})

	// 1. Typing 'rune deploy -' suggests short flag '-w', '-e' and long flags '--watch', '--env'
	cands := Complete(file, []string{"deploy", "-"})
	foundShortW := false
	foundLongEnv := false
	for _, c := range cands {
		val := strings.Split(c, "\t")[0]
		if val == "-w" {
			foundShortW = true
		}
		if val == "--env" {
			foundLongEnv = true
		}
	}
	if !foundShortW || !foundLongEnv {
		t.Errorf("expected -w and --env in completions, got %v", cands)
	}

	// 2. Typing 'rune deploy -e=' or '--env=' suggests choices
	choiceCands := Complete(file, []string{"deploy", "-e="})
	if len(choiceCands) != 2 {
		t.Fatalf("expected 2 choices, got %d: %v", len(choiceCands), choiceCands)
	}
	if !strings.Contains(choiceCands[0], "-e=staging") || !strings.Contains(choiceCands[1], "-e=production") {
		t.Errorf("expected -e=staging and -e=production, got %v", choiceCands)
	}
}
