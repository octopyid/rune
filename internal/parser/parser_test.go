package parser

import (
	"strings"
	"testing"
)

func TestParseValidRunefile(t *testing.T) {
	input := `
# Global comment
#[Build the application]
build target="dev" --race?:
    go build ./...

#[Run tests]
test: build
    go test ./...

#[Reset the database]
#[confirm: This will permanently delete all database data.]
db:fresh:
    php artisan migrate:fresh

#[Laravel command]
artisan *args:
    php artisan {{args}}

greet name:
    echo "Hello, {{name}}!"

# Meta task with multiple dependencies
release: build test
`

	file, err := Parse(strings.NewReader(input), "Runefile")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	if len(file.Tasks) != 6 {
		t.Fatalf("expected 6 tasks, got %d", len(file.Tasks))
	}

	// 1. Check build
	build, ok := file.GetTask("build")
	if !ok {
		t.Fatal("task 'build' not found")
	}
	if build.Description != "Build the application" {
		t.Errorf("expected description 'Build the application', got '%s'", build.Description)
	}
	if len(build.Parameters) != 1 || build.Parameters[0].Name != "target" || build.Parameters[0].DefaultValue != "dev" {
		t.Errorf("build parameter mismatch: %+v", build.Parameters)
	}
	if len(build.Flags) != 1 || build.Flags[0].Name != "race" {
		t.Errorf("build flag mismatch: %+v", build.Flags)
	}
	if len(build.Commands) != 1 || build.Commands[0] != "go build ./..." {
		t.Errorf("build commands mismatch: %+v", build.Commands)
	}

	// 2. Check test
	testTask, ok := file.GetTask("test")
	if !ok {
		t.Fatal("task 'test' not found")
	}
	if len(testTask.Dependencies) != 1 || testTask.Dependencies[0] != "build" {
		t.Errorf("test dependencies mismatch: %+v", testTask.Dependencies)
	}

	// 3. Check db:fresh
	dbFresh, ok := file.GetTask("db:fresh")
	if !ok {
		t.Fatal("task 'db:fresh' not found")
	}
	if dbFresh.Namespace != "db" {
		t.Errorf("expected namespace 'db', got '%s'", dbFresh.Namespace)
	}
	if dbFresh.ShortName != "fresh" {
		t.Errorf("expected short name 'fresh', got '%s'", dbFresh.ShortName)
	}
	if dbFresh.Confirmation != "This will permanently delete all database data." {
		t.Errorf("confirmation mismatch: '%s'", dbFresh.Confirmation)
	}

	// 4. Check artisan *args
	artisan, ok := file.GetTask("artisan")
	if !ok {
		t.Fatal("task 'artisan' not found")
	}
	if artisan.Passthrough == nil || artisan.Passthrough.Name != "args" {
		t.Errorf("expected passthrough 'args', got %+v", artisan.Passthrough)
	}

	// 5. Check greet
	greet, ok := file.GetTask("greet")
	if !ok {
		t.Fatal("task 'greet' not found")
	}
	if len(greet.Parameters) != 1 || greet.Parameters[0].Name != "name" || greet.Parameters[0].HasDefault {
		t.Errorf("greet parameters mismatch: %+v", greet.Parameters)
	}

	// 6. Check release
	release, ok := file.GetTask("release")
	if !ok {
		t.Fatal("task 'release' not found")
	}
	if len(release.Dependencies) != 2 || release.Dependencies[0] != "build" || release.Dependencies[1] != "test" {
		t.Errorf("release dependencies mismatch: %+v", release.Dependencies)
	}

	// Check namespace querying
	if !file.IsNamespace("db") {
		t.Error("expected 'db' to be recognized as a namespace")
	}
	dbTasks := file.TasksInNamespace("db")
	if len(dbTasks) != 1 || dbTasks[0].Name != "db:fresh" {
		t.Errorf("expected db:fresh in db namespace, got %+v", dbTasks)
	}
}

func TestParseErrors(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		errContains string
	}{
		{
			name: "missing colon",
			input: `
build
    go build
`,
			errContains: "missing colon",
		},
		{
			name: "unclosed quote",
			input: `
build target="dev:
    go build
`,
			errContains: "unclosed quote",
		},
		{
			name: "duplicate task",
			input: `
build:
    go build
build:
    go build
`,
			errContains: "duplicate task 'build'",
		},
		{
			name: "required after default",
			input: `
build target="dev" name:
    go build
`,
			errContains: "required argument 'name' cannot follow default argument",
		},
		{
			name: "argument after passthrough",
			input: `
artisan *args extra:
    php artisan
`,
			errContains: "cannot follow passthrough argument",
		},
		{
			name: "multiple passthrough",
			input: `
artisan *args *rest:
    php artisan
`,
			errContains: "only one passthrough argument",
		},
		{
			name: "unexpected indented command",
			input: `
    echo "orphan command"
`,
			errContains: "unexpected command line outside of a task definition",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(tt.input), "Runefile")
			if err == nil {
				t.Fatalf("expected error containing '%s', got nil", tt.errContains)
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("expected error containing '%s', got '%s'", tt.errContains, err.Error())
			}
		})
	}
}

func TestParseDocComments(t *testing.T) {
	input := `
# Disconnected comment separated by blank line
# --split: Should be ignored

# NOTE: General note about build
#[Build Android APK]
# target: Target build environment
# --split: Build split-per-ABI APKs alongside universal APK
# --race?: Enable data race detector
build target="dev" --split? --race?:
    # Comment inside body:
    # --split: Should be ignored
    echo "Building {{target}}"

# *args: Pass arbitrary flags to artisan
#[Run artisan]
artisan *args:
    php artisan {{args}}

greet name:
    echo "Hello, {{name}}"
`

	file, err := Parse(strings.NewReader(input), "Runefile")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	// Check build task
	build, ok := file.GetTask("build")
	if !ok {
		t.Fatal("task 'build' not found")
	}
	if len(build.Parameters) != 1 || build.Parameters[0].Description != "Target build environment" {
		t.Errorf("expected target description 'Target build environment', got %q", build.Parameters[0].Description)
	}
	if len(build.Flags) != 2 {
		t.Fatalf("expected 2 flags, got %d", len(build.Flags))
	}
	if build.Flags[0].Name != "split" || build.Flags[0].Description != "Build split-per-ABI APKs alongside universal APK" {
		t.Errorf("split flag description mismatch: %+v", build.Flags[0])
	}
	if build.Flags[1].Name != "race" || build.Flags[1].Description != "Enable data race detector" {
		t.Errorf("race flag description mismatch: %+v", build.Flags[1])
	}

	// Check artisan task
	artisan, ok := file.GetTask("artisan")
	if !ok {
		t.Fatal("task 'artisan' not found")
	}
	if artisan.Passthrough == nil || artisan.Passthrough.Description != "Pass arbitrary flags to artisan" {
		t.Errorf("artisan passthrough description mismatch: %+v", artisan.Passthrough)
	}

	// Check greet task (undocumented)
	greet, ok := file.GetTask("greet")
	if !ok {
		t.Fatal("task 'greet' not found")
	}
	if len(greet.Parameters) != 1 || greet.Parameters[0].Description != "" {
		t.Errorf("expected empty description for greet parameter, got %q", greet.Parameters[0].Description)
	}
}

func TestParseDirAttribute(t *testing.T) {
	// 1. Task without dir
	input1 := `
build:
    go build ./...
`
	file1, err := Parse(strings.NewReader(input1), "Runefile")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	task1, _ := file1.GetTask("build")
	if task1.Dir != "" {
		t.Errorf("expected empty Dir for task without #[dir], got %q", task1.Dir)
	}

	// 2. Relative and absolute dir
	input2 := `
#[dir: frontend]
build:web:
    npm run build

#[dir: /tmp/custom]
build:custom:
    echo custom
`
	file2, err := Parse(strings.NewReader(input2), "Runefile")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	taskWeb, _ := file2.GetTask("build:web")
	if taskWeb.Dir != "frontend" {
		t.Errorf("expected Dir 'frontend', got %q", taskWeb.Dir)
	}
	taskCustom, _ := file2.GetTask("build:custom")
	if taskCustom.Dir != "/tmp/custom" {
		t.Errorf("expected Dir '/tmp/custom', got %q", taskCustom.Dir)
	}

	// 3. Empty dir should produce ParseError
	input3 := `
#[dir: ]
build:
    go build ./...
`
	_, err = Parse(strings.NewReader(input3), "Runefile")
	if err == nil {
		t.Fatal("expected error for empty #[dir] attribute, got nil")
	}
	if !strings.Contains(err.Error(), "empty directory path") {
		t.Errorf("expected 'empty directory path' error, got %v", err)
	}
}

func TestParseEnvAttribute(t *testing.T) {
	// 1. Single #[env]
	input1 := `
#[env: GOOS=linux]
build:
    go build ./...
`
	file1, err := Parse(strings.NewReader(input1), "Runefile")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	task1, _ := file1.GetTask("build")
	if len(task1.Env) != 1 || task1.Env["GOOS"] != "linux" {
		t.Errorf("expected GOOS=linux, got %+v", task1.Env)
	}

	// 2. Multiple #[env] attributes accumulating, mixed with semicolon syntax, whitespace, and quoted values
	input2 := `
#[env: GOOS=linux]
#[env: CGO_ENABLED=0; GOARCH=amd64]
#[env: URL="https://example.com?a=1&b=2"; DEBUG='true'; ]
build:mixed:
    go build ./...
`
	file2, err := Parse(strings.NewReader(input2), "Runefile")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	task2, _ := file2.GetTask("build:mixed")
	if task2.Env["GOOS"] != "linux" {
		t.Errorf("expected GOOS=linux, got %q", task2.Env["GOOS"])
	}
	if task2.Env["CGO_ENABLED"] != "0" {
		t.Errorf("expected CGO_ENABLED=0, got %q", task2.Env["CGO_ENABLED"])
	}
	if task2.Env["GOARCH"] != "amd64" {
		t.Errorf("expected GOARCH=amd64, got %q", task2.Env["GOARCH"])
	}
	if task2.Env["URL"] != "https://example.com?a=1&b=2" {
		t.Errorf("expected URL='https://example.com?a=1&b=2', got %q", task2.Env["URL"])
	}
	if task2.Env["DEBUG"] != "true" {
		t.Errorf("expected DEBUG=true, got %q", task2.Env["DEBUG"])
	}

	// 3. Values containing '=' (e.g. KEY=foo=bar)
	input3 := `
#[env: KEY=foo=bar=baz]
test:
    echo $KEY
`
	file3, err := Parse(strings.NewReader(input3), "Runefile")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	task3, _ := file3.GetTask("test")
	if task3.Env["KEY"] != "foo=bar=baz" {
		t.Errorf("expected 'foo=bar=baz', got %q", task3.Env["KEY"])
	}

	// 4. Invalid assignments
	invalidCases := []struct {
		input  string
		errSub string
	}{
		{"#[env: ]\ntask:\n    echo 1\n", "empty environment assignment"},
		{"#[env: ; ; ]\ntask:\n    echo 1\n", "empty environment assignment"},
		{"#[env: FOO]\ntask:\n    echo 1\n", "missing '='"},
		{"#[env: =bar]\ntask:\n    echo 1\n", "missing variable name"},
	}
	for _, tc := range invalidCases {
		_, err := Parse(strings.NewReader(tc.input), "Runefile")
		if err == nil {
			t.Errorf("expected error containing %q for input %q, got nil", tc.errSub, tc.input)
			continue
		}
		if !strings.Contains(err.Error(), tc.errSub) {
			t.Errorf("expected error containing %q, got %v", tc.errSub, err)
		}
	}
}

func TestParsePrivateAttribute(t *testing.T) {
	input := `
#[Build task]
build:
    go build ./...

#[private]
#[Clean certs]
setup:certs:
    ./certs.sh

#[deploy]
deploy: setup:certs
    ./deploy.sh
`
	file, err := Parse(strings.NewReader(input), "Runefile")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	build, ok := file.GetTask("build")
	if !ok || build.Private {
		t.Errorf("expected build to not be private, got ok=%v, private=%v", ok, build.Private)
	}

	certs, ok := file.GetTask("setup:certs")
	if !ok || !certs.Private {
		t.Errorf("expected setup:certs to be private, got ok=%v, private=%v", ok, certs.Private)
	}

	deploy, ok := file.GetTask("deploy")
	if !ok || deploy.Private {
		t.Errorf("expected deploy to not be private, got ok=%v, private=%v", ok, deploy.Private)
	}

	// Verify File helper methods
	publicNames := file.PublicTaskNames()
	if len(publicNames) != 2 {
		t.Fatalf("expected 2 public tasks, got %d: %v", len(publicNames), publicNames)
	}
	if publicNames[0] != "build" || publicNames[1] != "deploy" {
		t.Errorf("expected [build, deploy], got %v", publicNames)
	}

	publicRoot := file.PublicRootTasks()
	if len(publicRoot) != 2 {
		t.Fatalf("expected 2 public root tasks, got %d", len(publicRoot))
	}

	publicNs := file.PublicNamespaces()
	if len(publicNs) != 0 {
		t.Errorf("expected 0 public namespaces (setup only had private tasks), got %v", publicNs)
	}
}

func TestParseOptionsAndEnums(t *testing.T) {
	input := `
# -w|--watch: Watch files
# -o|--output: Destination folder
# target: Entrypoint
build -w|--watch? -o|--output="dist" target="./cmd/app":
    go build {{watch}} -o {{output}} {{target}}

# -e|--env: Cloud env
# -m|--mode: Deploy mode
# -v|--verbose: Verbose output
deploy -e|--env=[staging,production] -m|--mode=[rolling,canary]="rolling" -v|--verbose?:
    ./deploy.sh

# action: Migration direction
db:migrate action=[up,down,status]="up" *args:
    migrate {{action}} {{args}}

# -t|--token: Secret auth token
# -t2|--tag: Tag version
release -t|--token= --empty=? req_action=[start,stop]:
    ./release.sh
`
	file, err := Parse(strings.NewReader(input), "Runefile")
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}

	// 1. Check build task
	build, ok := file.GetTask("build")
	if !ok {
		t.Fatal("task 'build' not found")
	}
	if len(build.Flags) != 2 {
		t.Fatalf("expected 2 flags on build, got %d", len(build.Flags))
	}
	// -w|--watch?
	wFlag := build.Flags[0]
	if wFlag.Short != "w" || wFlag.Name != "watch" || wFlag.IsValued || wFlag.Description != "Watch files" {
		t.Errorf("watch flag mismatch: %+v", wFlag)
	}
	// -o|--output="dist"
	oFlag := build.Flags[1]
	if oFlag.Short != "o" || oFlag.Name != "output" || !oFlag.IsValued || !oFlag.HasDefault || oFlag.DefaultValue != "dist" || oFlag.Description != "Destination folder" {
		t.Errorf("output flag mismatch: %+v", oFlag)
	}
	// target="./cmd/app"
	if len(build.Parameters) != 1 || build.Parameters[0].Name != "target" || build.Parameters[0].DefaultValue != "./cmd/app" {
		t.Errorf("target param mismatch: %+v", build.Parameters)
	}

	// 2. Check deploy task
	deploy, ok := file.GetTask("deploy")
	if !ok {
		t.Fatal("task 'deploy' not found")
	}
	if len(deploy.Flags) != 3 {
		t.Fatalf("expected 3 flags on deploy, got %d", len(deploy.Flags))
	}
	// -e|--env=[staging,production]
	eFlag := deploy.Flags[0]
	if eFlag.Short != "e" || eFlag.Name != "env" || !eFlag.IsValued || !eFlag.Required || len(eFlag.Choices) != 2 {
		t.Errorf("env flag mismatch: %+v", eFlag)
	}
	if eFlag.Choices[0] != "staging" || eFlag.Choices[1] != "production" {
		t.Errorf("expected [staging, production], got %v", eFlag.Choices)
	}
	// -m|--mode=[rolling,canary]="rolling"
	mFlag := deploy.Flags[1]
	if mFlag.Short != "m" || mFlag.Name != "mode" || !mFlag.IsValued || mFlag.Required || !mFlag.HasDefault || mFlag.DefaultValue != "rolling" {
		t.Errorf("mode flag mismatch: %+v", mFlag)
	}
	// -v|--verbose?
	vFlag := deploy.Flags[2]
	if vFlag.Short != "v" || vFlag.Name != "verbose" || vFlag.IsValued {
		t.Errorf("verbose flag mismatch: %+v", vFlag)
	}

	// 3. Check db:migrate
	migrate, ok := file.GetTask("db:migrate")
	if !ok {
		t.Fatal("task 'db:migrate' not found")
	}
	if len(migrate.Parameters) != 1 {
		t.Fatalf("expected 1 param on db:migrate, got %d", len(migrate.Parameters))
	}
	actionParam := migrate.Parameters[0]
	if actionParam.Name != "action" || !actionParam.HasDefault || actionParam.DefaultValue != "up" || len(actionParam.Choices) != 3 {
		t.Errorf("action param mismatch: %+v", actionParam)
	}

	// 4. Check release task
	release, ok := file.GetTask("release")
	if !ok {
		t.Fatal("task 'release' not found")
	}
	tFlag := release.Flags[0]
	if tFlag.Short != "t" || tFlag.Name != "token" || !tFlag.IsValued || !tFlag.Required {
		t.Errorf("token flag mismatch: %+v", tFlag)
	}
	emptyFlag := release.Flags[1]
	if emptyFlag.Name != "empty" || !emptyFlag.IsValued || emptyFlag.Required || !emptyFlag.HasDefault || emptyFlag.DefaultValue != "" {
		t.Errorf("empty flag mismatch: %+v", emptyFlag)
	}
	reqAction := release.Parameters[0]
	if reqAction.Name != "req_action" || reqAction.HasDefault || len(reqAction.Choices) != 2 {
		t.Errorf("req_action param mismatch: %+v", reqAction)
	}

	// 5. Test errors
	errorInputs := []struct {
		name string
		rf   string
		err  string
	}{
		{
			name: "multi-char short alias",
			rf:   "build -wh|--watch:\n    echo 1\n",
			err:  "must be exactly one character",
		},
		{
			name: "duplicate option",
			rf:   "build --watch --watch:\n    echo 1\n",
			err:  "duplicate option '--watch'",
		},
		{
			name: "duplicate short alias",
			rf:   "build -w|--watch -w|--worker:\n    echo 1\n",
			err:  "duplicate short option alias '-w'",
		},
		{
			name: "empty choices",
			rf:   "build --env=[]:\n    echo 1\n",
			err:  "choices cannot be empty",
		},
	}

	for _, tt := range errorInputs {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Parse(strings.NewReader(tt.rf), "Runefile")
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.err)
			}
			if !strings.Contains(err.Error(), tt.err) {
				t.Errorf("expected error %q, got %v", tt.err, err)
			}
		})
	}
}
