package executor

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

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
	Verbose    bool
	Time       bool
	Stdin      io.Reader
	Stdout     io.Writer
	Stderr     io.Writer
}

type taskTiming struct {
	name     string
	duration time.Duration
	failed   bool
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

	var timings []taskTiming
	planStart := time.Now()
	totalPlan := len(ctx.Plan)
	var planExitCode int

	for _, task := range ctx.Plan {
		taskStart := time.Now()

		// 1. Resolve and validate task-scoped working directory
		taskDir := ctx.WorkingDir
		if task.Dir != "" {
			if filepath.IsAbs(task.Dir) {
				taskDir = filepath.Clean(task.Dir)
			} else {
				taskDir = filepath.Join(ctx.WorkingDir, task.Dir)
			}
			stat, err := os.Stat(taskDir)
			if err != nil || !stat.IsDir() {
				ui.Fail(stderr, fmt.Sprintf("Directory '%s' does not exist for task '%s'", task.Dir, task.Name))
				if ctx.Time {
					timings = append(timings, taskTiming{name: task.Name, duration: time.Since(taskStart), failed: true})
					printTiming(stdout, timings, totalPlan, time.Since(planStart), 1)
				}
				return 1
			}
		}

		// 2. Determine bound args for this task
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

		// 3. Build task environment variables
		taskEnv := make(map[string]string)
		// Apply task-scoped #[env] attributes first (they override OS and .env)
		for k, v := range task.Env {
			taskEnv[k] = v
		}
		// Apply internal task variables and arguments
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

		// 4. Execute task commands sequentially
		taskFailed := false
		for _, cmdLine := range task.Commands {
			argv, err := ExpandCommand(cmdLine, taskBound)
			if err != nil {
				ui.Fail(stderr, fmt.Sprintf("Failed to expand command: %v", err))
				taskFailed = true
				planExitCode = 1
				break
			}

			if len(argv) == 0 {
				continue
			}

			if ctx.Verbose {
				fmt.Fprintf(stdout, "$ %s\n", formatArgv(argv))
			}

			exitCode := runCommand(argv, taskDir, mergedEnv, stdin, stdout, stderr)
			if exitCode != 0 {
				taskFailed = true
				planExitCode = exitCode
				break
			}
		}

		taskElapsed := time.Since(taskStart)
		if taskFailed {
			if ctx.Time {
				timings = append(timings, taskTiming{name: task.Name, duration: taskElapsed, failed: true})
				printTiming(stdout, timings, totalPlan, time.Since(planStart), planExitCode)
			}
			return planExitCode
		}

		if ctx.Time {
			timings = append(timings, taskTiming{name: task.Name, duration: taskElapsed, failed: false})
		}
	}

	if ctx.Time {
		printTiming(stdout, timings, totalPlan, time.Since(planStart), 0)
	}

	return 0
}

func printTiming(w io.Writer, timings []taskTiming, totalPlan int, totalDuration time.Duration, exitCode int) {
	if len(timings) == 0 {
		return
	}
	if totalPlan == 1 {
		t := timings[0]
		if exitCode != 0 {
			fmt.Fprintf(w, "\n✖ %s (%.2fs)\n\nTotal: %.2fs\n", t.name, t.duration.Seconds(), totalDuration.Seconds())
		} else {
			fmt.Fprintf(w, "\n✔ %s (%.2fs)\n\nTotal: %.2fs\n", t.name, t.duration.Seconds(), totalDuration.Seconds())
		}
		return
	}

	fmt.Fprintln(w)
	for i, t := range timings {
		status := ""
		if t.failed {
			status = " (failed)"
		}
		fmt.Fprintf(w, "[%d/%d] %-16s %s%.2fs\n", i+1, totalPlan, t.name+status, "", t.duration.Seconds())
	}
	fmt.Fprintln(w)
	if exitCode != 0 {
		fmt.Fprintf(w, "✖ Total: %.2fs\n", totalDuration.Seconds())
	} else {
		fmt.Fprintf(w, "✔ Total: %.2fs\n", totalDuration.Seconds())
	}
}

func runCommand(argv []string, dir string, env []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	cmd := exec.Command(argv[0], argv[1:]...)
	cmd.Dir = dir
	cmd.Env = env
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	setProcessGroup(cmd)

	var interrupted atomic.Bool
	var sigCount atomic.Int32
	sigChan := make(chan os.Signal, 2)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	defer signal.Stop(sigChan)

	if err := cmd.Start(); err != nil {
		ui.Fail(stderr, fmt.Sprintf("Command failed to start '%s': %v", argv[0], err))
		return 1
	}

	// Forward signals to child process
	go func() {
		for sig := range sigChan {
			interrupted.Store(true)
			count := sigCount.Add(1)
			if cmd.Process != nil {
				if count > 1 && (sig == syscall.SIGINT || sig == syscall.SIGTERM) {
					// Force kill child process on repeated interrupt signal
					_ = cmd.Process.Kill()
				} else {
					forwardSignal(cmd, sig)
				}
			}
		}
	}()

	err := cmd.Wait()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return exitStatus(exitErr)
		}
		return 1
	}

	if interrupted.Load() {
		return 130
	}

	return 0
}
