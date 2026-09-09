package graph

import (
	"strings"
	"testing"

	"github.com/octopyid/rune/internal/ast"
)

func TestResolvePlanDeduplicationAndOrder(t *testing.T) {
	// build:
	// test: build
	// release: build test
	build := &ast.Task{Name: "build"}
	testTask := &ast.Task{Name: "test", Dependencies: []string{"build"}}
	release := &ast.Task{Name: "release", Dependencies: []string{"build", "test"}}

	file := ast.NewFile("Runefile", []*ast.Task{build, testTask, release})

	plan, err := ResolvePlan(file, release)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(plan) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(plan))
	}

	names := []string{plan[0].Name, plan[1].Name, plan[2].Name}
	expected := "build,test,release"
	actual := strings.Join(names, ",")
	if actual != expected {
		t.Errorf("expected plan %s, got %s", expected, actual)
	}
}

func TestResolvePlanDiamond(t *testing.T) {
	// d
	// b: d
	// c: d
	// a: b c
	d := &ast.Task{Name: "d"}
	b := &ast.Task{Name: "b", Dependencies: []string{"d"}}
	c := &ast.Task{Name: "c", Dependencies: []string{"d"}}
	a := &ast.Task{Name: "a", Dependencies: []string{"b", "c"}}

	file := ast.NewFile("Runefile", []*ast.Task{d, b, c, a})

	plan, err := ResolvePlan(file, a)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(plan) != 4 {
		t.Fatalf("expected 4 tasks, got %d", len(plan))
	}

	// d must execute before b and c; b and c must execute before a
	orderMap := make(map[string]int)
	for i, task := range plan {
		orderMap[task.Name] = i
	}

	if orderMap["d"] > orderMap["b"] || orderMap["d"] > orderMap["c"] {
		t.Errorf("d should execute before b and c: %+v", orderMap)
	}
	if orderMap["b"] > orderMap["a"] || orderMap["c"] > orderMap["a"] {
		t.Errorf("b and c should execute before a: %+v", orderMap)
	}
}

func TestResolvePlanDirectCycle(t *testing.T) {
	// a: b
	// b: a
	a := &ast.Task{Name: "a", Dependencies: []string{"b"}}
	b := &ast.Task{Name: "b", Dependencies: []string{"a"}}

	file := ast.NewFile("Runefile", []*ast.Task{a, b})

	_, err := ResolvePlan(file, a)
	if err == nil {
		t.Fatal("expected cycle error, got nil")
	}

	cycleErr, ok := err.(*CycleError)
	if !ok {
		t.Fatalf("expected *CycleError, got %T: %v", err, err)
	}

	expectedStr := "a -> b -> a"
	if !strings.Contains(cycleErr.Error(), expectedStr) {
		t.Errorf("expected cycle error to contain '%s', got '%s'", expectedStr, cycleErr.Error())
	}
}

func TestResolvePlanSelfCycle(t *testing.T) {
	// a: a
	a := &ast.Task{Name: "a", Dependencies: []string{"a"}}
	file := ast.NewFile("Runefile", []*ast.Task{a})

	_, err := ResolvePlan(file, a)
	if err == nil {
		t.Fatal("expected cycle error, got nil")
	}
	if !strings.Contains(err.Error(), "a -> a") {
		t.Errorf("expected self cycle in error, got '%s'", err.Error())
	}
}

func TestResolvePlanMissingDependency(t *testing.T) {
	a := &ast.Task{Name: "a", Dependencies: []string{"missing"}}
	file := ast.NewFile("Runefile", []*ast.Task{a})

	_, err := ResolvePlan(file, a)
	if err == nil {
		t.Fatal("expected missing dependency error, got nil")
	}

	depErr, ok := err.(*DependencyError)
	if !ok {
		t.Fatalf("expected *DependencyError, got %T", err)
	}
	if depErr.Dependency != "missing" || depErr.Task != "a" {
		t.Errorf("unexpected dependency error values: %+v", depErr)
	}
}
