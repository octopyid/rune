package executor

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/octopyid/rune/internal/ast"
	"github.com/octopyid/rune/internal/ui"
	"golang.org/x/term"
)

// ErrCancelled indicates that the user rejected the confirmation prompt.
var ErrCancelled = errors.New("operation cancelled")

// ConfirmExecution checks if any task in the plan requires confirmation and prompts the user.
// Returns ErrCancelled if the user aborts.
func ConfirmExecution(tasks []*ast.Task, bypassYes bool, r io.Reader, w io.Writer) error {
	if bypassYes {
		return nil
	}

	for _, t := range tasks {
		if t.Confirmation == "" {
			continue
		}

		ui.Warn(w, t.Confirmation)

		if inFile, ok := isInteractive(r, w); ok {
			if err := promptInteractiveArrow(inFile, w); err != nil {
				return err
			}
		} else {
			if err := promptNonInteractive(r, w); err != nil {
				return err
			}
		}
	}

	return nil
}

func isInteractive(r io.Reader, w io.Writer) (*os.File, bool) {
	inFile, okIn := r.(*os.File)
	outFile, okOut := w.(*os.File)
	if okIn && okOut && term.IsTerminal(int(inFile.Fd())) && term.IsTerminal(int(outFile.Fd())) {
		return inFile, true
	}
	return nil, false
}

func promptNonInteractive(r io.Reader, w io.Writer) error {
	prompt := color.New(color.Bold).Sprint("  Continue? [y/N] ")
	_, _ = fmt.Fprint(w, prompt)

	scanner := bufio.NewScanner(r)
	if !scanner.Scan() {
		_, _ = fmt.Fprintln(w, "\nOperation cancelled.")
		return ErrCancelled
	}

	response := strings.TrimSpace(scanner.Text())
	if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
		_, _ = fmt.Fprintln(w, "Operation cancelled.")
		return ErrCancelled
	}

	return nil
}

func promptInteractiveArrow(inFile *os.File, w io.Writer) error {
	oldState, err := term.MakeRaw(int(inFile.Fd()))
	if err != nil {
		return promptNonInteractive(inFile, w)
	}

	restored := false
	restore := func() {
		if !restored {
			restored = true
			_ = term.Restore(int(inFile.Fd()), oldState)
			_, _ = fmt.Fprint(w, "\x1b[?25h") // restore cursor visibility
		}
	}
	defer restore()

	// Hide cursor during interactive selection
	_, _ = fmt.Fprint(w, "\x1b[?25l")

	_, _ = fmt.Fprintf(w, "  %s\r\n", color.New(color.Bold).Sprint("Continue?"))

	selected := 0 // 0: Yes, 1: No

	renderSelection := func(sel int) {
		var yesStr, noStr string
		if sel == 0 {
			yesStr = color.New(color.FgCyan, color.Bold).Sprint("❯ ") + color.New(color.FgGreen, color.Bold).Sprint("Yes")
			noStr = "  No"
		} else {
			yesStr = "  Yes"
			noStr = color.New(color.FgCyan, color.Bold).Sprint("❯ ") + color.New(color.FgRed, color.Bold).Sprint("No")
		}
		_, _ = fmt.Fprintf(w, "\r\x1b[2K  %s    %s", yesStr, noStr)
	}

	renderSelection(selected)

	buf := make([]byte, 16)
	for {
		n, readErr := inFile.Read(buf)
		if readErr != nil {
			restore()
			_, _ = fmt.Fprintln(w)
			ui.Warn(w, "Operation cancelled.")
			return ErrCancelled
		}
		if n == 0 {
			continue
		}

		if n == 1 {
			switch buf[0] {
			case 3, 4: // Ctrl+C, Ctrl+D
				restore()
				_, _ = fmt.Fprintln(w)
				ui.Warn(w, "Operation cancelled.")
				return ErrCancelled
			case 13, 10: // Enter (\r or \n)
				restore()
				_, _ = fmt.Fprintln(w)
				if selected == 0 {
					return nil
				}
				ui.Warn(w, "Operation cancelled.")
				return ErrCancelled
			case ' ': // Space
				restore()
				_, _ = fmt.Fprintln(w)
				if selected == 0 {
					return nil
				}
				ui.Warn(w, "Operation cancelled.")
				return ErrCancelled
			case '\t': // Tab toggles
				selected = 1 - selected
				renderSelection(selected)
			case 'h', 'H': // Left
				if selected != 0 {
					selected = 0
					renderSelection(selected)
				}
			case 'l', 'L': // Right
				if selected != 1 {
					selected = 1
					renderSelection(selected)
				}
			case 'y', 'Y': // Direct confirm Yes
				restore()
				_, _ = fmt.Fprintln(w)
				return nil
			case 'n', 'N': // Direct confirm No
				restore()
				_, _ = fmt.Fprintln(w)
				ui.Warn(w, "Operation cancelled.")
				return ErrCancelled
			case 27: // Escape
				restore()
				_, _ = fmt.Fprintln(w)
				ui.Warn(w, "Operation cancelled.")
				return ErrCancelled
			}
		} else if n >= 3 && buf[0] == 27 && buf[1] == '[' {
			switch buf[2] {
			case 'D', 'A': // Left or Up
				if selected != 0 {
					selected = 0
					renderSelection(selected)
				}
			case 'C', 'B': // Right or Down
				if selected != 1 {
					selected = 1
					renderSelection(selected)
				}
			}
		}
	}
}
