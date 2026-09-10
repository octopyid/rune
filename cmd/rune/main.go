package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/octopyid/rune/internal/cli"
	"github.com/octopyid/rune/internal/env"
	"github.com/octopyid/rune/internal/executor"
	"github.com/octopyid/rune/internal/graph"
	"github.com/octopyid/rune/internal/ui"
)

var version = ""

func init() {
	if version != "" {
		cli.Version = strings.TrimPrefix(version, "v")
	}
}

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	wd, err := os.Getwd()
	if err != nil {
		printErrorTo(os.Stderr, fmt.Sprintf("failed to determine working directory: %v", err))
		return 1
	}

	ctx := cli.RunContext{
		Args:       args,
		Stdout:     os.Stdout,
		Stderr:     os.Stderr,
		Stdin:      os.Stdin,
		WorkingDir: wd,
	}

	action := cli.Route(ctx)

	if action.Err != nil {
		printErrorTo(ctx.Stderr, action.Err.Error())
		return action.ExitCode
	}

	if action.Output != "" {
		_, _ = fmt.Fprint(ctx.Stdout, action.Output)
		return action.ExitCode
	}

	if action.Kind == cli.ActionExecute {
		// 1. Resolve dependency plan
		plan, err := graph.ResolvePlan(action.File, action.TargetTask)
		if err != nil {
			printErrorTo(ctx.Stderr, err.Error())
			return 1
		}

		// 2. Handle dry run
		if action.GlobalFlags.DryRun {
			output, err := executor.FormatDryRun(plan, action.TargetTask, action.BoundArgs)
			if err != nil {
				printErrorTo(ctx.Stderr, err.Error())
				return 1
			}
			_, _ = fmt.Fprint(ctx.Stdout, output)
			return 0
		}

		// 3. Confirmation check before executing any task in the plan
		if err := executor.ConfirmExecution(plan, action.GlobalFlags.Yes, ctx.Stdin, ctx.Stdout); err != nil {
			return 1
		}

		// 4. Load .env from Runefile's directory
		taskFileDir := filepath.Dir(action.File.Path)
		dotEnv, _ := env.LoadDotEnv(taskFileDir)

		// 5. Execute plan
		execCtx := executor.ExecutionContext{
			Plan:       plan,
			TargetTask: action.TargetTask,
			BoundArgs:  action.BoundArgs,
			WorkingDir: taskFileDir,
			DotEnv:     dotEnv,
			Verbose:    action.GlobalFlags.Verbose,
			Time:       action.GlobalFlags.Time,
			Stdin:      ctx.Stdin,
			Stdout:     ctx.Stdout,
			Stderr:     ctx.Stderr,
		}

		return executor.ExecutePlan(execCtx)
	}

	return 0
}

func printErrorTo(w io.Writer, msg string) {
	if after, ok := strings.CutPrefix(msg, "✗ "); ok {
		msg = after
	}

	// Capitalize the first letter
	if len(msg) > 0 {
		runes := []rune(msg)
		if len(runes) > 0 {
			runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
			msg = string(runes)
		}
	}

	ui.Fail(w, msg)
}
