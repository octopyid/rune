package cli

import (
	"fmt"
	"strings"

	"github.com/octopyid/rune/internal/ast"
)

// CompletionItem represents a completion candidate with an optional description.
type CompletionItem struct {
	Value       string
	Description string
}

var globalFlagsWithDesc = []CompletionItem{
	{Value: "--help", Description: "Display help for command"},
	{Value: "-h", Description: "Display help for command"},
	{Value: "--version", Description: "Display application version"},
	{Value: "-v", Description: "Display application version"},
	{Value: "--file", Description: "Path to task file (default: Runefile)"},
	{Value: "-f", Description: "Path to task file (default: Runefile)"},
	{Value: "--dry-run", Description: "Simulate execution without running commands"},
	{Value: "--yes", Description: "Bypass interactive confirmation"},
	{Value: "-y", Description: "Bypass interactive confirmation"},
	{Value: "--verbose", Description: "Display commands before executing them"},
	{Value: "--time", Description: "Display task execution duration"},
	{Value: "--tree", Description: "Display the task dependency tree"},
}

var builtinCommandsWithDesc = []CompletionItem{
	{Value: "completion", Description: "Dump the shell completion script"},
	{Value: "help", Description: "Display help for a command"},
	{Value: "list", Description: "List available commands"},
	{Value: "info", Description: "Display an informational message badge"},
	{Value: "warn", Description: "Display a warning message badge"},
	{Value: "fail", Description: "Display a failure alert with exit code 1"},
	{Value: "error", Description: "Display an error alert with exit code 1"},
	{Value: "done", Description: "Display a success message badge"},
}

// Complete generates completion suggestions given the current command-line words.
func Complete(file *ast.File, words []string) []string {
	last := lastWord(words)

	// If no task file found, offer global flags
	if file == nil {
		return filterPrefix(globalFlagsWithDesc, last)
	}

	// Determine if a task name has already been provided prior to the current word
	var foundTask *ast.Task
	if len(words) > 1 {
		for _, w := range words[:len(words)-1] {
			if strings.HasPrefix(w, "-") {
				continue
			}
			if t, ok := file.GetTask(w); ok {
				foundTask = t
				break
			}
		}
	}

	// If a task is already chosen, complete task flags and global flags
	if foundTask != nil {
		var candidates []CompletionItem
		for _, f := range foundTask.Flags {
			desc := f.Description
			if desc == "" {
				desc = "Flag for " + foundTask.Name
			}
			candidates = append(candidates, CompletionItem{
				Value:       "--" + f.Name,
				Description: desc,
			})
		}
		candidates = append(candidates, globalFlagsWithDesc...)
		return filterPrefix(candidates, last)
	}

	// If user is typing a flag (starts with '-'), complete global flags only
	if strings.HasPrefix(last, "-") {
		return filterPrefix(globalFlagsWithDesc, last)
	}

	// Otherwise, complete public task names, public namespaces, and built-in commands
	var candidates []CompletionItem
	for _, t := range file.PublicTasks() {
		desc := t.Description
		if desc == "" {
			desc = "Run task"
		}
		candidates = append(candidates, CompletionItem{
			Value:       t.Name,
			Description: desc,
		})
	}

	for _, ns := range file.PublicNamespaces() {
		candidates = append(candidates, CompletionItem{
			Value:       ns,
			Description: fmt.Sprintf("Tasks in '%s' namespace", ns),
		})
	}

	candidates = append(candidates, builtinCommandsWithDesc...)

	return filterPrefix(candidates, last)
}

func lastWord(words []string) string {
	if len(words) == 0 {
		return ""
	}
	return words[len(words)-1]
}

func filterPrefix(candidates []CompletionItem, prefix string) []string {
	var result []string
	seen := make(map[string]bool)
	for _, c := range candidates {
		if strings.HasPrefix(c.Value, prefix) && !seen[c.Value] {
			seen[c.Value] = true
			if c.Description != "" {
				result = append(result, c.Value+"\t"+c.Description)
			} else {
				result = append(result, c.Value)
			}
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
    suggestions=$(rune __complete "${COMP_WORDS[@]:1}" 2>/dev/null | cut -f1)
    COMPREPLY=($(compgen -W "${suggestions}" -- "${cur}"))
}
complete -F _rune_completions rune
`
}

// ZshCompletionScript generates zsh completion code.
func ZshCompletionScript() string {
	return `#compdef rune
compdef _rune rune 2>/dev/null || true

_rune() {
    local -a raw_completions completions
    local comp tab
    tab="$(printf '\t')"

    raw_completions=(${(f)"$(rune __complete "${words[@]:1}" 2>/dev/null)"})

    for comp in "${raw_completions[@]}"; do
        if [[ -n "$comp" ]]; then
            comp=${comp//:/\\:}
            comp=${comp//$tab/:}
            completions+=("${comp}")
        fi
    done

    if [[ ${#completions[@]} -gt 0 ]]; then
        _describe 'commands' completions
    fi
}

if [ "$funcstack[1]" = "_rune" ]; then
    _rune "$@"
fi
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
