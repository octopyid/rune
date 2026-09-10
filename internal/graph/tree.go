package graph

import (
	"fmt"
	"strings"

	"github.com/octopyid/rune/internal/ast"
)

// FormatTaskTree formats the dependency tree for a single task using ASCII box-drawing characters.
func FormatTaskTree(file *ast.File, rootTask *ast.Task) (string, error) {
	if _, err := ResolvePlan(file, rootTask); err != nil {
		return "", err
	}

	var sb strings.Builder
	sb.WriteString(nodeLabel(rootTask) + "\n")

	if len(rootTask.Dependencies) == 0 {
		sb.WriteString("└── (no dependencies)\n")
		return strings.TrimRight(sb.String(), "\n"), nil
	}

	visited := make(map[string]bool)
	for i, depName := range rootTask.Dependencies {
		depTask, exists := file.GetTask(depName)
		if !exists {
			continue
		}
		isLast := (i == len(rootTask.Dependencies)-1)
		renderTree(&sb, file, depTask, "", isLast, visited)
	}

	return strings.TrimRight(sb.String(), "\n"), nil
}

// FormatFileTree formats dependency trees for all public tasks in the Runefile.
func FormatFileTree(file *ast.File) (string, error) {
	publicTasks := file.PublicTasks()
	if len(publicTasks) == 0 {
		return "No public tasks found.", nil
	}

	// Validate plan upfront for every task to ensure no cycles exist
	for _, t := range publicTasks {
		if _, err := ResolvePlan(file, t); err != nil {
			return "", err
		}
	}

	var sb strings.Builder
	for i, t := range publicTasks {
		tree, err := FormatTaskTree(file, t)
		if err != nil {
			return "", err
		}
		if i > 0 {
			sb.WriteString("\n\n")
		}
		sb.WriteString(tree)
	}

	return sb.String(), nil
}

func renderTree(sb *strings.Builder, file *ast.File, task *ast.Task, prefix string, isLast bool, visited map[string]bool) {
	connector := "├── "
	childPrefix := prefix + "│   "
	if isLast {
		connector = "└── "
		childPrefix = prefix + "    "
	}

	label := nodeLabel(task)

	if visited[task.Name] {
		sb.WriteString(prefix + connector + label + " (deduped)\n")
		return
	}

	visited[task.Name] = true
	sb.WriteString(prefix + connector + label + "\n")

	for i, depName := range task.Dependencies {
		depTask, exists := file.GetTask(depName)
		if !exists {
			continue
		}
		depIsLast := (i == len(task.Dependencies)-1)
		renderTree(sb, file, depTask, childPrefix, depIsLast, visited)
	}
}

func nodeLabel(task *ast.Task) string {
	var sb strings.Builder
	sb.WriteString(task.Name)
	if task.Private {
		sb.WriteString(" [private]")
	}
	if task.Dir != "" {
		fmt.Fprintf(&sb, " (dir: %s)", task.Dir)
	}
	return sb.String()
}
