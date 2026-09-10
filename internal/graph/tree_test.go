package graph

import (
	"strings"
	"testing"

	"github.com/octopyid/rune/internal/ast"
)

func TestFormatTaskTreeNoDependencies(t *testing.T) {
	task := &ast.Task{Name: "clean"}
	file := ast.NewFile("Runefile", []*ast.Task{task})

	out, err := FormatTaskTree(file, task)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := "clean\n└── (no dependencies)"
	if out != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, out)
	}
}

func TestFormatTaskTreeWithDependenciesAndDedup(t *testing.T) {
	// release -> build, test
	// test -> build, lint
	// lint -> (none)
	// build -> (none, with dir "backend")
	// secret -> (none, private)
	// deploy -> release, secret
	build := &ast.Task{Name: "build", Dir: "backend"}
	lint := &ast.Task{Name: "lint"}
	test := &ast.Task{Name: "test", Dependencies: []string{"build", "lint"}}
	release := &ast.Task{Name: "release", Dependencies: []string{"build", "test"}}
	secret := &ast.Task{Name: "secret", Private: true}
	deploy := &ast.Task{Name: "deploy", Dependencies: []string{"release", "secret"}}

	file := ast.NewFile("Runefile", []*ast.Task{build, lint, test, release, secret, deploy})

	out, err := FormatTaskTree(file, deploy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `deploy
├── release
│   ├── build (dir: backend)
│   └── test
│       ├── build (dir: backend) (deduped)
│       └── lint
└── secret [private]`

	if out != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, out)
	}
}

func TestFormatTaskTreeCycleError(t *testing.T) {
	a := &ast.Task{Name: "a", Dependencies: []string{"b"}}
	b := &ast.Task{Name: "b", Dependencies: []string{"a"}}
	file := ast.NewFile("Runefile", []*ast.Task{a, b})

	_, err := FormatTaskTree(file, a)
	if err == nil {
		t.Fatal("expected cycle error, got nil")
	}
	if !strings.Contains(err.Error(), "dependency cycle detected") {
		t.Errorf("expected cycle error message, got: %v", err)
	}
}

func TestFormatFileTree(t *testing.T) {
	build := &ast.Task{Name: "build"}
	test := &ast.Task{Name: "test", Dependencies: []string{"build"}}
	hidden := &ast.Task{Name: "hidden", Private: true}

	file := ast.NewFile("Runefile", []*ast.Task{build, test, hidden})

	out, err := FormatFileTree(file)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := `build
└── (no dependencies)

test
└── build`

	if out != expected {
		t.Errorf("expected:\n%s\ngot:\n%s", expected, out)
	}

	// Ensure hidden private task is not a root in file tree
	if strings.Contains(out, "hidden") {
		t.Errorf("expected private task 'hidden' to be excluded from root file tree, got:\n%s", out)
	}
}
