package executor

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/octopyid/rune/internal/ast"
	"github.com/octopyid/rune/internal/cli"
)

// FormatDryRun formats the dry run execution plan for the given task and dependency chain.
func FormatDryRun(plan []*ast.Task, targetTask *ast.Task, bound *cli.BoundArgs) (string, error) {
	var sb strings.Builder

	badge := color.New(color.FgYellow, color.Bold).Sprint("[dry-run]")
	fmt.Fprintf(&sb, "%s Execution plan for '%s':\n", badge, targetTask.Name)

	for i, task := range plan {
		fmt.Fprintf(&sb, "  %d. %s\n", i+1, task.Name)

		// If this is the target task, use bound arguments
		var taskBound *cli.BoundArgs
		if task.Name == targetTask.Name {
			taskBound = bound
		} else {
			// Dependencies use their default arguments and flags
			taskBound = &cli.BoundArgs{
				Task:      task,
				Arguments: make(map[string]string),
				Flags:     make(map[string]bool),
			}
			for _, p := range task.Parameters {
				if p.HasDefault {
					taskBound.Arguments[p.Name] = p.DefaultValue
				}
			}
			for _, f := range task.Flags {
				taskBound.Flags[f.Name] = false
			}
		}

		if len(task.Commands) == 0 {
			sb.WriteString("     (no commands)\n")
			continue
		}

		for _, cmdLine := range task.Commands {
			argv, err := ExpandCommand(cmdLine, taskBound)
			if err != nil {
				return "", err
			}
			fmt.Fprintf(&sb, "     $ %s\n", formatArgv(argv))
		}
	}

	return sb.String(), nil
}

func formatArgv(argv []string) string {
	var escaped []string
	for _, arg := range argv {
		if strings.ContainsAny(arg, " \t\n\"'") {
			escaped = append(escaped, fmt.Sprintf("%q", arg))
		} else {
			escaped = append(escaped, arg)
		}
	}
	return strings.Join(escaped, " ")
}
