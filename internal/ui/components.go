package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
)

var (
	// Badges exactly matching Laravel Console Components (Termwind)
	badgeInfo  = color.New(color.BgBlue, color.FgWhite).SprintFunc()
	badgeWarn  = color.New(color.BgYellow, color.FgBlack).SprintFunc()
	badgeFail  = color.New(color.BgRed, color.FgWhite).SprintFunc()
	badgeError = color.New(color.BgRed, color.FgWhite).SprintFunc()
	badgeDone  = color.New(color.BgGreen, color.FgWhite).SprintFunc()
)

// Info prints a blue INFO badge with top and bottom margins (Laravel my-1 style).
func Info(w io.Writer, msg string) {
	_, _ = fmt.Fprintf(w, "\n  %s %s\n\n", badgeInfo(" INFO "), msg)
}

// Warn prints a yellow WARN badge with black text and top and bottom margins (Laravel my-1 style).
func Warn(w io.Writer, msg string) {
	_, _ = fmt.Fprintf(w, "\n  %s %s\n\n", badgeWarn(" WARN "), msg)
}

// Fail prints a red FAIL badge with top and bottom margins.
func Fail(w io.Writer, msg string) {
	cleanMsg := strings.TrimPrefix(msg, "✗ ")
	_, _ = fmt.Fprintf(w, "\n  %s %s\n\n", badgeFail(" FAIL "), cleanMsg)
}

// Error prints a red ERROR badge with top and bottom margins (Laravel my-1 style).
func Error(w io.Writer, msg string) {
	cleanMsg := strings.TrimPrefix(msg, "✗ ")
	_, _ = fmt.Fprintf(w, "\n  %s %s\n\n", badgeError(" ERROR "), cleanMsg)
}

// Done prints a green DONE badge with top and bottom margins.
func Done(w io.Writer, msg string) {
	_, _ = fmt.Fprintf(w, "\n  %s %s\n\n", badgeDone(" DONE "), msg)
}

// Success is an alias for Done.
func Success(w io.Writer, msg string) {
	Done(w, msg)
}
