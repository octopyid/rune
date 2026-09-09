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
