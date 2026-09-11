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
	Tree    bool
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
		if arg == "--tree" {
			gf.Tree = true
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
				"--tree",
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

	// Initialize boolean flags to false
	for _, f := range task.Flags {
		if !f.IsValued {
			bound.Flags[f.Name] = false
		}
	}

	// Collect flag candidates for typo suggestions
	var flagCandidates []string
	for _, f := range task.Flags {
		flagCandidates = append(flagCandidates, "--"+f.Name)
		if f.Short != "" {
			flagCandidates = append(flagCandidates, "-"+f.Short)
		}
	}

	suppliedOptions := make(map[string]bool)
	var positionalInputs []string

	i := 0
	for i < len(rawArgs) {
		arg := rawArgs[i]

		// Explicit end of options delimiter '--'
		if arg == "--" {
			i++
			for i < len(rawArgs) {
				if len(positionalInputs) < len(task.Parameters) {
					positionalInputs = append(positionalInputs, rawArgs[i])
				} else if task.Passthrough != nil {
					bound.PassthroughArgs = append(bound.PassthroughArgs, rawArgs[i])
				} else {
					return nil, fmt.Errorf("unexpected argument: %s\n\nUsage:\n  %s", rawArgs[i], FormatUsage(task))
				}
				i++
			}
			break
		}

		// Long option: --foo or --foo=value
		if strings.HasPrefix(arg, "--") {
			optPart := strings.TrimPrefix(arg, "--")
			optName, val, hasEqual := strings.Cut(optPart, "=")

			var foundFlag *ast.Flag
			for idx := range task.Flags {
				if task.Flags[idx].Name == optName {
					foundFlag = &task.Flags[idx]
					break
				}
			}

			if foundFlag != nil {
				if foundFlag.IsValued {
					var optVal string
					if hasEqual {
						optVal = val
					} else {
						if i+1 < len(rawArgs) {
							optVal = rawArgs[i+1]
							i++
						} else {
							return nil, fmt.Errorf("option '--%s' requires a value", foundFlag.Name)
						}
					}
					if len(foundFlag.Choices) > 0 && !contains(foundFlag.Choices, optVal) {
						return nil, fmt.Errorf("invalid value %q for option %s\nAllowed choices: %s", optVal, foundFlag.OptionPrefix(), strings.Join(foundFlag.Choices, ", "))
					}
					bound.Arguments[foundFlag.Name] = optVal
					suppliedOptions[foundFlag.Name] = true
					i++
					continue
				}

				// Boolean flag
				if hasEqual {
					bound.Flags[foundFlag.Name] = (val == "true" || val == "1" || val == "")
				} else {
					bound.Flags[foundFlag.Name] = true
				}
				suppliedOptions[foundFlag.Name] = true
				i++
				continue
			}

			// Unknown option with passthrough
			if task.Passthrough != nil {
				bound.PassthroughArgs = append(bound.PassthroughArgs, arg)
				i++
				continue
			}

			err := fmt.Errorf("unknown option: %s", arg)
			if sugg := Suggest(arg, flagCandidates); sugg != "" {
				err = fmt.Errorf("unknown option: %s\n\nDid you mean:\n  %s", arg, sugg)
			}
			return nil, err
		}

		// Short option: -f, -f=value, or -f value
		if strings.HasPrefix(arg, "-") && arg != "-" {
			shortPart := strings.TrimPrefix(arg, "-")
			shortName, val, hasEqual := strings.Cut(shortPart, "=")

			var foundFlag *ast.Flag
			var attachedVal string
			var hasAttached bool

			if len(shortName) == 1 {
				for idx := range task.Flags {
					if task.Flags[idx].Short == shortName {
						foundFlag = &task.Flags[idx]
						break
					}
				}
			} else if !hasEqual && len(shortPart) > 1 {
				firstChar := string(shortPart[0])
				for idx := range task.Flags {
					if task.Flags[idx].Short == firstChar && task.Flags[idx].IsValued {
						foundFlag = &task.Flags[idx]
						attachedVal = shortPart[1:]
						hasAttached = true
						break
					}
				}
			}

			if foundFlag != nil {
				if foundFlag.IsValued {
					var optVal string
					if hasEqual {
						optVal = val
					} else if hasAttached {
						optVal = attachedVal
					} else {
						if i+1 < len(rawArgs) {
							optVal = rawArgs[i+1]
							i++
						} else {
							return nil, fmt.Errorf("option '-%s' requires a value", foundFlag.Short)
						}
					}
					if len(foundFlag.Choices) > 0 && !contains(foundFlag.Choices, optVal) {
						return nil, fmt.Errorf("invalid value %q for option %s\nAllowed choices: %s", optVal, foundFlag.OptionPrefix(), strings.Join(foundFlag.Choices, ", "))
					}
					bound.Arguments[foundFlag.Name] = optVal
					suppliedOptions[foundFlag.Name] = true
					i++
					continue
				}

				// Boolean flag
				if hasEqual {
					bound.Flags[foundFlag.Name] = (val == "true" || val == "1" || val == "")
				} else {
					bound.Flags[foundFlag.Name] = true
				}
				suppliedOptions[foundFlag.Name] = true
				i++
				continue
			}

			// Unknown option with passthrough
			if task.Passthrough != nil {
				bound.PassthroughArgs = append(bound.PassthroughArgs, arg)
				i++
				continue
			}

			err := fmt.Errorf("unknown option: %s", arg)
			if sugg := Suggest(arg, flagCandidates); sugg != "" {
				err = fmt.Errorf("unknown option: %s\n\nDid you mean:\n  %s", arg, sugg)
			}
			return nil, err
		}

		// Positional argument
		if len(positionalInputs) < len(task.Parameters) {
			positionalInputs = append(positionalInputs, arg)
			i++
			continue
		}

		// Passthrough argument
		if task.Passthrough != nil {
			bound.PassthroughArgs = append(bound.PassthroughArgs, arg)
			i++
			continue
		}

		// Extra positional argument
		return nil, fmt.Errorf("unexpected argument: %s\n\nUsage:\n  %s", arg, FormatUsage(task))
	}

	// 1. Verify required options and apply option defaults
	for _, f := range task.Flags {
		if f.IsValued {
			if suppliedOptions[f.Name] {
				continue
			}
			if f.HasDefault {
				bound.Arguments[f.Name] = f.DefaultValue
				continue
			}
			// Missing required option
			optDisplay := f.OptionPrefix()
			return nil, fmt.Errorf("missing required option: %s\n\nUsage:\n  %s", optDisplay, FormatUsage(task))
		}

		if f.Required && !bound.Flags[f.Name] {
			optDisplay := f.OptionPrefix()
			return nil, fmt.Errorf("missing required option: %s\n\nUsage:\n  %s", optDisplay, FormatUsage(task))
		}
	}

	// 2. Match positional inputs against task.Parameters
	for idx, param := range task.Parameters {
		if idx < len(positionalInputs) {
			val := positionalInputs[idx]
			if len(param.Choices) > 0 && !contains(param.Choices, val) {
				return nil, fmt.Errorf("invalid value %q for argument %q\nAllowed choices: %s", val, param.Name, strings.Join(param.Choices, ", "))
			}
			bound.Arguments[param.Name] = val
		} else if param.HasDefault {
			bound.Arguments[param.Name] = param.DefaultValue
		} else {
			return nil, fmt.Errorf("missing argument: %s\n\nUsage:\n  %s", param.Name, FormatUsage(task))
		}
	}

	return bound, nil
}

func contains(list []string, item string) bool {
	for _, v := range list {
		if v == item {
			return true
		}
	}
	return false
}

// FormatUsage formats the usage string for a task.
func FormatUsage(task *ast.Task) string {
	var parts []string
	parts = append(parts, "rune", task.Name)

	for _, f := range task.Flags {
		var optStr string
		if f.Short != "" {
			optStr = fmt.Sprintf("-%s|--%s", f.Short, f.Name)
		} else {
			optStr = "--" + f.Name
		}
		if f.IsValued {
			if len(f.Choices) > 0 {
				optStr += fmt.Sprintf("=[%s]", strings.Join(f.Choices, "|"))
			} else {
				optStr += "=VALUE"
			}
		}
		if f.Required {
			parts = append(parts, fmt.Sprintf("<%s>", optStr))
		} else {
			parts = append(parts, fmt.Sprintf("[%s]", optStr))
		}
	}

	for _, param := range task.Parameters {
		if param.HasDefault {
			parts = append(parts, fmt.Sprintf("[%s]", param.Name))
		} else {
			parts = append(parts, fmt.Sprintf("<%s>", param.Name))
		}
	}

	if task.Passthrough != nil {
		parts = append(parts, fmt.Sprintf("[%s...]", task.Passthrough.Name))
	}

	return strings.Join(parts, " ")
}
