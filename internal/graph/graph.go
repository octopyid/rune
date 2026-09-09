package graph

import (
	"fmt"
	"strings"

	"github.com/octopyid/rune/internal/ast"
)

// CycleError represents a circular dependency error.
type CycleError struct {
	Cycle []string
}

func (e *CycleError) Error() string {
	return fmt.Sprintf("dependency cycle detected: %s", strings.Join(e.Cycle, " -> "))
}

// DependencyError represents a missing dependency task error.
type DependencyError struct {
	Task       string
	Dependency string
}

func (e *DependencyError) Error() string {
	return fmt.Sprintf("unknown dependency '%s' required by '%s'", e.Dependency, e.Task)
}

// ResolvePlan builds a deterministic, deduplicated execution plan for the given root task.
// Dependencies are guaranteed to execute before their dependents.
func ResolvePlan(file *ast.File, rootTask *ast.Task) ([]*ast.Task, error) {
	const (
		stateUnvisited = 0
		stateVisiting  = 1
		stateVisited   = 2
	)

	state := make(map[string]int)
	var (
		order []*ast.Task
		stack []string
	)

	var visit func(task *ast.Task) error
	visit = func(task *ast.Task) error {
		st := state[task.Name]
		if st == stateVisiting {
			// Find cycle in stack
			cycleStart := 0
			for i, name := range stack {
				if name == task.Name {
					cycleStart = i
					break
				}
			}
			cyclePath := append(stack[cycleStart:], task.Name)
			return &CycleError{Cycle: cyclePath}
		}
		if st == stateVisited {
			return nil
		}

		state[task.Name] = stateVisiting
		stack = append(stack, task.Name)

		for _, depName := range task.Dependencies {
			depTask, exists := file.GetTask(depName)
			if !exists {
				return &DependencyError{
					Task:       task.Name,
					Dependency: depName,
				}
			}

			if err := visit(depTask); err != nil {
				return err
			}
		}

		stack = stack[:len(stack)-1]
		state[task.Name] = stateVisited
		order = append(order, task)
		return nil
	}

	if err := visit(rootTask); err != nil {
		return nil, err
	}

	return order, nil
}
