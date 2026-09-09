package cli

import (
	"fmt"
	"strings"

	"github.com/octopyid/rune/internal/ast"
)

// Complete generates completion suggestions given the current command-line words.
func Complete(file *ast.File, words []string) []string {
	globalFlags := []string{"--help", "-h", "--version", "-v", "--file", "-f", "--dry-run", "--yes", "-y"}

	// If no task file found, offer global flags
	if file == nil {
		return filterPrefix(globalFlags, lastWord(words))
	}

	// Determine if a task name has already been provided
	var foundTask *ast.Task
	for _, w := range words {
		if strings.HasPrefix(w, "-") {
			continue
		}
		if t, ok := file.GetTask(w); ok {
			foundTask = t
			break
		}
	}

	last := lastWord(words)

	// If a task is already chosen, complete task flags and global flags
	if foundTask != nil {
		var candidates []string
		for _, f := range foundTask.Flags {
			candidates = append(candidates, "--"+f.Name)
		}
		candidates = append(candidates, globalFlags...)
		return filterPrefix(candidates, last)
	}

	// Otherwise, complete task names, namespaces, and global flags
	var candidates []string
	candidates = append(candidates, file.AllTaskNames()...)
	candidates = append(candidates, file.Namespaces()...)
	candidates = append(candidates, "info", "warn", "fail", "error", "done", "help", "list", "completion")
	candidates = append(candidates, globalFlags...)

	return filterPrefix(candidates, last)
}

func lastWord(words []string) string {
	if len(words) == 0 {
		return ""
	}
	return words[len(words)-1]
}

func filterPrefix(candidates []string, prefix string) []string {
	var result []string
	seen := make(map[string]bool)
	for _, c := range candidates {
		if strings.HasPrefix(c, prefix) && !seen[c] {
			seen[c] = true
			result = append(result, c)
		}
	}
	return result
}

// BashCompletionScript generates bash completion code.
func BashCompletionScript() string {
	return `_rune_completions() {
    local cur
    cur="${COMP_WORDS[COMP_CWORD]}"
    local suggestions
    suggestions=$(rune __complete "${COMP_WORDS[@]:1}" 2>/dev/null)
    COMPREPLY=($(compgen -W "${suggestions}" -- "${cur}"))
}
complete -F _rune_completions rune
`
}

// ZshCompletionScript generates zsh completion code.
func ZshCompletionScript() string {
	return `#compdef rune

_rune() {
    local -a suggestions
    suggestions=(${(f)"$(rune __complete "${words[@]:1}" 2>/dev/null)"})
    _describe 'commands' suggestions
}

_rune "$@"
`
}

// FishCompletionScript generates fish completion code.
func FishCompletionScript() string {
	return `function __fish_rune_complete
    set -l cmd (commandline -cop)
    rune __complete $cmd[2..-1] 2>/dev/null
end

complete -c rune -f -a "(__fish_rune_complete)"
`
}

// GenerateCompletion prints the completion script for the requested shell.
func GenerateCompletion(shell string) (string, error) {
	switch strings.ToLower(shell) {
	case "bash":
		return BashCompletionScript(), nil
	case "zsh":
		return ZshCompletionScript(), nil
	case "fish":
		return FishCompletionScript(), nil
	default:
		return "", fmt.Errorf("unsupported shell '%s' (supported: bash, zsh, fish)", shell)
	}
}
