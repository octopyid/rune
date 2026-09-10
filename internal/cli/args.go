package cli

import (
	"fmt"
	"strings"

	"github.com/octopyid/rune/internal/ast"
)

// GlobalFlags holds global CLI options.
type GlobalFlags struct {
	Help    bool
	Version bool
	File    string
	DryRun  bool
	Yes     bool
	Verbose bool
	Time    bool
}

// BoundArgs contains the resolved arguments and options for a task invocation.
type BoundArgs struct {
	Task            *ast.Task
	Arguments       map[string]string // param name -> resolved value
	Flags           map[string]bool   // flag name -> boolean value
	PassthroughArgs []string          // arguments for *args
}

// ParseGlobalFlags separates global flags from task arguments.
// Returns global flags, task name (if any), and remaining task arguments.
func ParseGlobalFlags(args []string) (GlobalFlags, string, []string, error) {
	var (
		gf       GlobalFlags
		taskName string
		taskArgs []string
	)

	i := 0
	for i < len(args) {
		arg := args[i]

		if arg == "-h" || arg == "--help" {
			gf.Help = true
			i++
			continue
		}
		if arg == "-V" || arg == "-v" || arg == "--version" {
			gf.Version = true
			i++
			continue
		}
		if arg == "-f" || arg == "--file" {
			if i+1 >= len(args) {
				return gf, "", nil, fmt.Errorf("missing argument for %s", arg)
			}
			gf.File = args[i+1]
			i += 2
			continue
		}
		if after, ok := strings.CutPrefix(arg, "--file="); ok {
			gf.File = after
			i++
			continue
		}
		if arg == "--dry-run" {
			gf.DryRun = true
			i++
			continue
		}
		if arg == "-y" || arg == "--yes" || arg == "-n" || arg == "--no-interaction" {
			gf.Yes = true
			i++
			continue
		}
		if arg == "--verbose" {
			gf.Verbose = true
			i++
			continue
		}
		if arg == "--time" {
			gf.Time = true
			i++
			continue
		}

		// First non-flag argument is the task name (or namespace)
		if taskName == "" && !strings.HasPrefix(arg, "-") {
			taskName = arg
			i++
			continue
		}

		// If no task name has been encountered and it's a flag, it's an unrecognized global flag
		if taskName == "" && strings.HasPrefix(arg, "-") {
			globalCandidates := []string{
				"--help", "-h",
				"--version", "-v", "-V",
				"--file", "-f",
				"--dry-run",
				"--yes", "-y",
				"--verbose",
				"--time",
			}
			sugg := Suggest(arg, globalCandidates)
			if sugg != "" {
				return gf, "", nil, fmt.Errorf("unknown option: %s\n\nDid you mean:\n  %s", arg, sugg)
			}
			return gf, "", nil, fmt.Errorf("unknown option: %s", arg)
		}

		// Subsequent arguments are task arguments
		taskArgs = append(taskArgs, arg)
		i++
	}

	return gf, taskName, taskArgs, nil
}

// BindTaskArgs binds command-line arguments to a task's parameters, flags, and passthrough.
func BindTaskArgs(task *ast.Task, rawArgs []string) (*BoundArgs, error) {
	bound := &BoundArgs{
		Task:            task,
		Arguments:       make(map[string]string),
		Flags:           make(map[string]bool),
		PassthroughArgs: make([]string, 0),
	}

	// Initialize flags to false
	for _, f := range task.Flags {
		bound.Flags[f.Name] = false
	}

	// Collect defined flag names for validation and suggestion
	definedFlags := make(map[string]bool)
	var flagCandidates []string
	for _, f := range task.Flags {
		definedFlags[f.Name] = true
		flagCandidates = append(flagCandidates, "--"+f.Name)
	}

	var positionalInputs []string

	i := 0
	for i < len(rawArgs) {
		arg := rawArgs[i]

		// Check if it's a flag
		if after, ok := strings.CutPrefix(arg, "--"); ok {
			flagRaw := after
			before, after, ok := strings.Cut(flagRaw, "=")

			var flagName, flagVal string
			if ok {
				flagName = before
				flagVal = after
			} else {
				flagName = flagRaw
				flagVal = "true"
			}

			if definedFlags[flagName] {
				bound.Flags[flagName] = (flagVal == "true" || flagVal == "1" || flagVal == "")
				i++
				continue
			}

			// If task has passthrough, unknown options belong to passthrough
			if task.Passthrough != nil {
				bound.PassthroughArgs = append(bound.PassthroughArgs, arg)
				i++
				continue
			}

			// Unknown option error with suggestion
			err := fmt.Errorf("unknown option: %s", arg)
			if sugg := Suggest(arg, flagCandidates); sugg != "" {
				err = fmt.Errorf("unknown option: %s\n\nDid you mean:\n  %s", arg, sugg)
			}
			return nil, err
		}

		// Positional argument
		// If we haven't satisfied positional parameters, take it
		if len(positionalInputs) < len(task.Parameters) {
			positionalInputs = append(positionalInputs, arg)
			i++
			continue
		}

		// If all positional parameters are filled and task has passthrough, append to passthrough
		if task.Passthrough != nil {
			bound.PassthroughArgs = append(bound.PassthroughArgs, arg)
			i++
			continue
		}

		// Extra positional argument without passthrough
		return nil, fmt.Errorf("unexpected argument: %s\n\nUsage:\n  %s", arg, FormatUsage(task))
	}

	// Now match positionalInputs against task.Parameters
	for idx, param := range task.Parameters {
		if idx < len(positionalInputs) {
			bound.Arguments[param.Name] = positionalInputs[idx]
		} else if param.HasDefault {
			bound.Arguments[param.Name] = param.DefaultValue
		} else {
			return nil, fmt.Errorf("missing argument: %s\n\nUsage:\n  %s", param.Name, FormatUsage(task))
		}
	}

	return bound, nil
}

// FormatUsage formats the usage string for a task.
func FormatUsage(task *ast.Task) string {
	var parts []string
	parts = append(parts, "rune", task.Name)

	for _, param := range task.Parameters {
		if param.HasDefault {
			parts = append(parts, fmt.Sprintf("[%s]", param.Name))
		} else {
			parts = append(parts, fmt.Sprintf("<%s>", param.Name))
		}
	}

	for _, f := range task.Flags {
		parts = append(parts, fmt.Sprintf("[--%s]", f.Name))
	}

	if task.Passthrough != nil {
		parts = append(parts, fmt.Sprintf("[%s...]", task.Passthrough.Name))
	}

	return strings.Join(parts, " ")
}
