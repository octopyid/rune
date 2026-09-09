package executor

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/octopyid/rune/internal/ast"
	"github.com/octopyid/rune/internal/cli"
	"github.com/octopyid/rune/internal/env"
	"github.com/octopyid/rune/internal/ui"
)

// ExecutionContext holds parameters required for executing a task plan.
type ExecutionContext struct {
	Plan       []*ast.Task
	TargetTask *ast.Task
	BoundArgs  *cli.BoundArgs
	WorkingDir string
	DotEnv     map[string]string
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
}

// ExecutePlan runs each task in the plan sequentially.
// Dependencies are guaranteed to run before dependents.
// If any command fails (non-zero exit code), execution stops immediately (fail-fast).
func ExecutePlan(ctx ExecutionContext) int {
	stdin := ctx.Stdin
	if stdin == nil {
		stdin = os.Stdin
	}
	stdout := ctx.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := ctx.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}

	for _, task := range ctx.Plan {
		// Determine bound args for this task
		var taskBound *cli.BoundArgs
		if task.Name == ctx.TargetTask.Name {
			taskBound = ctx.BoundArgs
		} else {
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

		// Build task environment variables
		taskEnv := make(map[string]string)
		taskEnv["RUNE_TASK"] = task.Name
		for k, v := range taskBound.Arguments {
			taskEnv[k] = v
			taskEnv["RUNE_ARG_"+strings.ToUpper(k)] = v
		}
		for k, val := range taskBound.Flags {
			if val {
				taskEnv[k] = "true"
				taskEnv["RUNE_FLAG_"+strings.ToUpper(k)] = "1"
			}
		}

		mergedEnv := env.BuildEnvironment(os.Environ(), ctx.DotEnv, taskEnv)

		// Execute task commands sequentially
		for _, cmdLine := range task.Commands {
			argv, err := ExpandCommand(cmdLine, taskBound)
			if err != nil {
				ui.Fail(stderr, fmt.Sprintf("Failed to expand command: %v", err))
				return 1
			}

			if len(argv) == 0 {
				continue
			}

			exitCode := runCommand(argv, ctx.WorkingDir, mergedEnv, stdin, stdout, stderr)
			if exitCode != 0 {
				return exitCode
			}
		}
	}

	return 0
}

func runCommand(argv []string, dir string, env []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	setProcessGroup(cmd)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer func() {
		signal.Stop(sigChan)
		close(sigChan)
	}()

	if err := cmd.Start(); err != nil {
		ui.Fail(stderr, fmt.Sprintf("Command failed to start '%s': %v", argv[0], err))
		return 1
	}

	// Forward signals to child process
	go func() {
		for sig := range sigChan {
			if cmd.Process != nil {
				forwardSignal(cmd, sig)
			}
		}
	}()

	err := cmd.Wait()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitErr.ExitCode()
		}
		return 1
	}

	return 0
}
