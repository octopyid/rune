package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/octopyid/rune/internal/ast"
	"github.com/octopyid/rune/internal/parser"
	"github.com/octopyid/rune/internal/ui"
)

var Version = "1.0.0"

// RunContext encapsulates execution context for the CLI.
type RunContext struct {
	Args       []string
	Stdout     io.Writer
	Stderr     io.Writer
	Stdin      io.Reader
	WorkingDir string
}

// Action represents the outcome of CLI routing.
type ActionKind int

const (
	ActionNone ActionKind = iota
	ActionHelp
	ActionVersion
	ActionCompletion
	ActionExecute
)

type Action struct {
	Kind        ActionKind
	File        *ast.File
	TargetTask  *ast.Task
	BoundArgs   *BoundArgs
	GlobalFlags GlobalFlags
	Output      string
	Err         error
	ExitCode    int
}

// FindRunefile looks for Runefile or runefile starting in startDir and traversing parent directories.
func FindRunefile(startDir string, explicitFile string) (string, error) {
	if explicitFile != "" {
		if _, err := os.Stat(explicitFile); err != nil {
			return "", fmt.Errorf("task file not found: %s", explicitFile)
		}
		return filepath.Abs(explicitFile)
	}

	curr := startDir
	for {
		for _, name := range []string{"Runefile", "runefile"} {
			candidate := filepath.Join(curr, name)
			if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
				return candidate, nil
			}
		}

		parent := filepath.Dir(curr)
		if parent == curr {
			break
		}
		curr = parent
	}

	return "", os.ErrNotExist
}

// Route processes arguments and determines what action to take.
func Route(ctx RunContext) Action {
	args := ctx.Args

	// 1. Check for shell completion subcommands
	if len(args) > 0 {
		if args[0] == "completion" {
			if len(args) < 2 {
				return Action{
					Kind:     ActionNone,
					Err:      fmt.Errorf("usage: rune completion [bash|zsh|fish]"),
					ExitCode: 1,
				}
			}
			script, err := GenerateCompletion(args[1])
			if err != nil {
				return Action{
					Kind:     ActionNone,
					Err:      err,
					ExitCode: 1,
				}
			}
			return Action{
				Kind:     ActionCompletion,
				Output:   script,
				ExitCode: 0,
			}
		}

		if args[0] == "__complete" {
			rfPath, _ := FindRunefile(ctx.WorkingDir, "")
			var file *ast.File
			if rfPath != "" {
				file, _ = parser.ParseFile(rfPath)
			}
			suggestions := Complete(file, args[1:])
			return Action{
				Kind:     ActionCompletion,
				Output:   fmt.Sprintf("%s\n", joinLines(suggestions)),
				ExitCode: 0,
			}
		}
	}

	// 2. Parse global flags
	gf, taskName, taskArgs, err := ParseGlobalFlags(args)
	if err != nil {
		return Action{
			Kind:     ActionNone,
			Err:      err,
			ExitCode: 1,
		}
	}

	// 3. Check version
	if gf.Version {
		return Action{
			Kind:     ActionVersion,
			Output:   fmt.Sprintf("%s %s\n", colorApp("Rune"), colorVer(Version)),
			ExitCode: 0,
		}
	}

	// 4. Find Runefile
	rfPath, err := FindRunefile(ctx.WorkingDir, gf.File)
	var file *ast.File
	if err == nil {
		file, err = parser.ParseFile(rfPath)
		if err != nil {
			return Action{
				Kind:     ActionNone,
				Err:      err,
				ExitCode: 1,
			}
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return Action{
			Kind:     ActionNone,
			Err:      err,
			ExitCode: 1,
		}
	}

	// 5. If no task specified: show root help
	if taskName == "" {
		return Action{
			Kind:        ActionHelp,
			Output:      FormatRootHelp(file) + "\n",
			GlobalFlags: gf,
			ExitCode:    0,
		}
	}

	// Built-in 'list' command (unless user defined custom 'list' task in Runefile)
	if taskName == "list" && (file == nil || !file.HasTask("list")) {
		return Action{
			Kind:        ActionHelp,
			Output:      FormatRootHelp(file) + "\n",
			GlobalFlags: gf,
			ExitCode:    0,
		}
	}

	// Built-in 'help' command (unless user defined custom 'help' task in Runefile)
	if taskName == "help" && (file == nil || !file.HasTask("help")) {
		if len(taskArgs) == 0 {
			return Action{
				Kind:        ActionHelp,
				Output:      FormatRootHelp(file) + "\n",
				GlobalFlags: gf,
				ExitCode:    0,
			}
		}
		target := taskArgs[0]
		if file != nil {
			if t, ok := file.GetTask(target); ok {
				return Action{
					Kind:        ActionHelp,
					Output:      FormatTaskHelp(t) + "\n",
					TargetTask:  t,
					GlobalFlags: gf,
					ExitCode:    0,
				}
			}
			if file.IsNamespace(target) {
				return Action{
					Kind:        ActionHelp,
					Output:      FormatNamespaceHelp(file, target) + "\n",
					GlobalFlags: gf,
					ExitCode:    0,
				}
			}
		}
		if bTask := GetBuiltinTask(target); bTask != nil {
			return Action{
				Kind:        ActionHelp,
				Output:      FormatTaskHelp(bTask) + "\n",
				TargetTask:  bTask,
				GlobalFlags: gf,
				ExitCode:    0,
			}
		}

		var candidates []string
		if file != nil {
			candidates = append(candidates, file.AllTaskNames()...)
			candidates = append(candidates, file.Namespaces()...)
		}
		candidates = append(candidates, "completion", "list", "info", "warn", "error", "fail", "done", "help")
		sugg := Suggest(target, candidates)
		msg := fmt.Sprintf("unknown task: %s", target)
		if sugg != "" {
			msg = fmt.Sprintf("unknown task: %s\n\nDid you mean:\n  %s", target, sugg)
		}
		return Action{
			Kind:     ActionNone,
			Err:      fmt.Errorf("%s", msg),
			ExitCode: 1,
		}
	}

	// Built-in UI component commands: rune info, warn, fail/error, done/success
	switch taskName {
	case "info", "warn", "fail", "error", "done", "success":
		if file == nil || !file.HasTask(taskName) {
			if gf.Help {
				bTask := GetBuiltinTask(taskName)
				return Action{
					Kind:        ActionHelp,
					Output:      FormatTaskHelp(bTask) + "\n",
					TargetTask:  bTask,
					GlobalFlags: gf,
					ExitCode:    0,
				}
			}
			var buf strings.Builder
			msg := strings.Join(taskArgs, " ")
			exitCode := 0
			switch taskName {
			case "info":
				ui.Info(&buf, msg)
			case "warn":
				ui.Warn(&buf, msg)
			case "fail":
				ui.Fail(&buf, msg)
				exitCode = 1
			case "error":
				ui.Error(&buf, msg)
				exitCode = 1
			case "done", "success":
				ui.Done(&buf, msg)
			}
			return Action{
				Kind:     ActionNone,
				Output:   buf.String(),
				ExitCode: exitCode,
			}
		}
	}

	// If task is specified, but no Runefile exists
	if file == nil {
		return Action{
			Kind:     ActionNone,
			Err:      fmt.Errorf("no Runefile found in current directory or parents"),
			ExitCode: 1,
		}
	}

	// 6. Check if task exists
	task, taskExists := file.GetTask(taskName)

	// If task does not exist, check if taskName is a namespace (e.g. 'rune db' or 'rune db --help')
	if !taskExists && file.IsNamespace(taskName) {
		return Action{
			Kind:        ActionHelp,
			Output:      FormatNamespaceHelp(file, taskName) + "\n",
			GlobalFlags: gf,
			ExitCode:    0,
		}
	}

	// If neither task nor namespace exists: suggestion error
	if !taskExists {
		var candidates []string
		candidates = append(candidates, file.AllTaskNames()...)
		candidates = append(candidates, file.Namespaces()...)

		sugg := Suggest(taskName, candidates)
		msg := fmt.Sprintf("unknown task: %s", taskName)
		if sugg != "" {
			msg = fmt.Sprintf("unknown task: %s\n\nDid you mean:\n  %s", taskName, sugg)
		}
		return Action{
			Kind:     ActionNone,
			Err:      fmt.Errorf("%s", msg),
			ExitCode: 1,
		}
	}

	// 7. If help flag is set on task: show task help
	if gf.Help {
		return Action{
			Kind:        ActionHelp,
			Output:      FormatTaskHelp(task) + "\n",
			TargetTask:  task,
			GlobalFlags: gf,
			ExitCode:    0,
		}
	}

	// 9. Bind arguments to task
	bound, err := BindTaskArgs(task, taskArgs)
	if err != nil {
		return Action{
			Kind:     ActionNone,
			Err:      err,
			ExitCode: 1,
		}
	}

	return Action{
		Kind:        ActionExecute,
		File:        file,
		TargetTask:  task,
		BoundArgs:   bound,
		GlobalFlags: gf,
		ExitCode:    0,
	}
}

func joinLines(lines []string) string {
	var sb strings.Builder
	for i, l := range lines {
		if i > 0 {
			sb.WriteString("\n")
		}
		sb.WriteString(l)
	}
	return sb.String()
}
