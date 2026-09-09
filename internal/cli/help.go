package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/fatih/color"
	"github.com/octopyid/rune/internal/ast"
)

var (
	colorHeader = color.New(color.FgYellow).SprintFunc()
	colorCmd    = color.New(color.FgGreen).SprintFunc()
	colorNs     = color.New(color.FgYellow).SprintFunc()
	colorOption = color.New(color.FgGreen).SprintFunc()
	colorBanner = color.New(color.FgGreen, color.Bold).SprintFunc()
	colorApp    = color.New(color.Bold).SprintFunc()
	colorVer    = color.New(color.FgYellow).SprintFunc()
)

const defaultCmdWidth = 40

type optionItem struct {
	short string
	long  string
	desc  string
}

var globalOptions = []optionItem{
	{short: "-h", long: "--help", desc: "Display help for the given command. When no command is given display help for the list command"},
	{short: "-v", long: "--version", desc: "Display this application version"},
	{short: "-f", long: "--file[=FILE]", desc: "Path to task file (default: Runefile)"},
	{short: "", long: "--dry-run", desc: "Simulate execution without running commands"},
	{short: "-y", long: "--yes", desc: "Do not ask any interactive question (bypass confirmation)"},
}

type commandEntry struct {
	Name string
	Desc string
}

func getBuiltinCommands() []commandEntry {
	return []commandEntry{
		{Name: "completion", Desc: "Dump the shell completion script"},
		{Name: "help", Desc: "Display help for a command"},
		{Name: "list", Desc: "List commands"},
	}
}

// GetBuiltinTask returns the synthetic AST Task for a built-in command or helper.
func GetBuiltinTask(name string) *ast.Task {
	switch name {
	case "info":
		return &ast.Task{
			Name:        "info",
			Description: "Display an informational message badge",
			Passthrough: &ast.Passthrough{Name: "message"},
		}
	case "warn":
		return &ast.Task{
			Name:        "warn",
			Description: "Display a warning message badge",
			Passthrough: &ast.Passthrough{Name: "message"},
		}
	case "error":
		return &ast.Task{
			Name:        "error",
			Description: "Display an error message badge and exit with code 1",
			Passthrough: &ast.Passthrough{Name: "message"},
		}
	case "fail":
		return &ast.Task{
			Name:        "fail",
			Description: "Display an error message badge and exit with code 1",
			Passthrough: &ast.Passthrough{Name: "message"},
		}
	case "done":
		return &ast.Task{
			Name:        "done",
			Description: "Display a success message badge",
			Passthrough: &ast.Passthrough{Name: "message"},
		}
	case "success":
		return &ast.Task{
			Name:        "success",
			Description: "Display a success message badge",
			Passthrough: &ast.Passthrough{Name: "message"},
		}
	case "completion":
		return &ast.Task{
			Name:        "completion",
			Description: "Dump the shell completion script",
			Parameters:  []ast.Parameter{{Name: "shell"}},
		}
	case "list":
		return &ast.Task{
			Name:        "list",
			Description: "List commands",
		}
	case "help":
		return &ast.Task{
			Name:        "help",
			Description: "Display help for a command",
			Parameters:  []ast.Parameter{{Name: "command", HasDefault: true, DefaultValue: ""}},
		}
	default:
		return nil
	}
}

// FormatRootHelp formats the root help text matching Laravel Artisan's exact layout and styling.
func FormatRootHelp(file *ast.File) string {
	var sb strings.Builder

	// Top banner: Minimalist Block (Gaya Bun / Deno)
	sb.WriteString(colorBanner("  █▀█ █░█ █▄░█ █▀▀") + "\n")
	sb.WriteString(colorBanner("  █▀▄ █▄█ █░▀█ ██▄") + "\n\n")
	fmt.Fprintf(&sb, "%s %s\n\n", colorApp("Rune"), colorVer(Version))

	sb.WriteString(colorHeader("Usage:") + "\n")
	sb.WriteString("  command [options] [arguments]\n\n")

	sb.WriteString(colorHeader("Options:") + "\n")
	formatOptionsTable(&sb, globalOptions, 24)
	sb.WriteString("\n")

	sb.WriteString(colorHeader("Available commands:") + "\n")

	// Collect all root commands
	cmdMap := make(map[string]string)
	for _, b := range getBuiltinCommands() {
		cmdMap[b.Name] = b.Desc
	}
	if file != nil {
		for _, t := range file.RootTasks() {
			cmdMap[t.Name] = t.Description
		}
	}

	var rootCommands []commandEntry
	for name, desc := range cmdMap {
		rootCommands = append(rootCommands, commandEntry{Name: name, Desc: desc})
	}
	sort.Slice(rootCommands, func(i, j int) bool {
		return rootCommands[i].Name < rootCommands[j].Name
	})

	// Calculate uniform command column width (minimum 40 to match Artisan)
	cmdWidth := defaultCmdWidth
	for _, c := range rootCommands {
		if len(c.Name)+4 > cmdWidth {
			cmdWidth = len(c.Name) + 4
		}
	}
	if file != nil {
		for _, t := range file.Tasks {
			if len(t.Name)+4 > cmdWidth {
				cmdWidth = len(t.Name) + 4
			}
		}
	}

	// 1. Root tasks first
	for _, c := range rootCommands {
		padding := strings.Repeat(" ", max(2, cmdWidth-len(c.Name)-2))
		fmt.Fprintf(&sb, "  %s%s%s\n", colorCmd(c.Name), padding, c.Desc)
	}

	// 2. Namespaced tasks grouped under single-space indented namespace headers
	if file != nil {
		namespaces := append([]string(nil), file.Namespaces()...)
		sort.Strings(namespaces)
		for _, ns := range namespaces {
			fmt.Fprintf(&sb, " %s\n", colorNs(ns))
			tasks := file.TasksInNamespace(ns)
			sort.Slice(tasks, func(i, j int) bool {
				return tasks[i].Name < tasks[j].Name
			})
			for _, t := range tasks {
				desc := t.Description
				padding := strings.Repeat(" ", max(2, cmdWidth-len(t.Name)-2))
				fmt.Fprintf(&sb, "  %s%s%s\n", colorCmd(t.Name), padding, desc)
			}
		}
	}

	return strings.TrimRight(sb.String(), "\n")
}

// FormatNamespaceHelp formats the help for a specific namespace.
func FormatNamespaceHelp(file *ast.File, ns string) string {
	var sb strings.Builder

	tasks := file.TasksInNamespace(ns)
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].Name < tasks[j].Name
	})

	sb.WriteString(colorBanner("  █▀█ █░█ █▄░█ █▀▀") + "\n")
	sb.WriteString(colorBanner("  █▀▄ █▄█ █░▀█ ██▄") + "\n\n")
	fmt.Fprintf(&sb, "%s %s\n\n", colorApp("Rune"), colorVer(Version))

	sb.WriteString(colorHeader("Usage:") + "\n")
	fmt.Fprintf(&sb, "  %s:<command> [options] [arguments]\n\n", ns)

	sb.WriteString(colorHeader("Options:") + "\n")
	formatOptionsTable(&sb, globalOptions, 24)
	sb.WriteString("\n")

	sb.WriteString(colorHeader("Available commands:") + "\n")
	cmdWidth := defaultCmdWidth
	for _, t := range tasks {
		if len(t.Name)+4 > cmdWidth {
			cmdWidth = len(t.Name) + 4
		}
	}

	fmt.Fprintf(&sb, " %s\n", colorNs(ns))
	for _, t := range tasks {
		desc := t.Description
		padding := strings.Repeat(" ", max(2, cmdWidth-len(t.Name)-2))
		fmt.Fprintf(&sb, "  %s%s%s\n", colorCmd(t.Name), padding, desc)
	}

	return strings.TrimRight(sb.String(), "\n")
}

// FormatTaskHelp formats detailed help for a specific task matching Laravel Artisan's command help.
func FormatTaskHelp(task *ast.Task) string {
	var sb strings.Builder

	if task.Description != "" {
		sb.WriteString(colorHeader("Description:") + "\n")
		fmt.Fprintf(&sb, "  %s\n\n", task.Description)
	}

	sb.WriteString(colorHeader("Usage:") + "\n")
	fmt.Fprintf(&sb, "  %s\n\n", formatTaskUsageArtisan(task))

	if len(task.Parameters) > 0 || task.Passthrough != nil {
		sb.WriteString(colorHeader("Arguments:") + "\n")
		argWidth := 24
		for _, p := range task.Parameters {
			if len(p.Name)+4 > argWidth {
				argWidth = len(p.Name) + 4
			}
		}
		if task.Passthrough != nil && len(task.Passthrough.Name)+4 > argWidth {
			argWidth = len(task.Passthrough.Name) + 4
		}
		for _, p := range task.Parameters {
			padding := strings.Repeat(" ", max(2, argWidth-len(p.Name)-2))
			desc := p.Description
			if desc == "" {
				desc = "Required argument"
				if p.HasDefault {
					desc = fmt.Sprintf("Optional argument [default: %q]", p.DefaultValue)
				}
			} else if p.HasDefault {
				desc = fmt.Sprintf("%s [default: %q]", p.Description, p.DefaultValue)
			}
			fmt.Fprintf(&sb, "  %s%s%s\n", colorCmd(p.Name), padding, desc)
		}
		if task.Passthrough != nil {
			padding := strings.Repeat(" ", max(2, argWidth-len(task.Passthrough.Name)-2))
			desc := "Passes arbitrary arguments through to the underlying command"
			if task.Passthrough.Description != "" {
				desc = task.Passthrough.Description
			}
			fmt.Fprintf(&sb, "  %s%s%s\n", colorCmd(task.Passthrough.Name), padding, desc)
		}
		sb.WriteString("\n")
	}

	sb.WriteString(colorHeader("Options:") + "\n")

	// Calculate maximum column width across task-specific flags AND global options
	optWidth := 24
	for _, f := range task.Flags {
		prefix := fmt.Sprintf("      --%s", f.Name)
		if len(prefix)+2 > optWidth {
			optWidth = len(prefix) + 2
		}
	}
	for _, opt := range globalOptions {
		prefix := formatOptionPrefix(opt.short, opt.long)
		if len(prefix)+2 > optWidth {
			optWidth = len(prefix) + 2
		}
	}

	// 1. Task-specific flags first
	for _, f := range task.Flags {
		prefix := fmt.Sprintf("      --%s", f.Name)
		padding := strings.Repeat(" ", max(2, optWidth-len(prefix)))
		desc := f.Description
		if desc == "" {
			desc = "Optional boolean flag"
		}
		fmt.Fprintf(&sb, "      %s%s%s\n", colorOption("--"+f.Name), padding, desc)
	}

	// 2. Global options next (perfectly aligned with task flags)
	formatOptionsTable(&sb, globalOptions, optWidth)

	if len(task.Dependencies) > 0 {
		sb.WriteString("\n" + colorHeader("Dependencies:") + "\n")
		fmt.Fprintf(&sb, "  %s\n", strings.Join(task.Dependencies, ", "))
	}

	if task.Confirmation != "" {
		warn := color.New(color.FgYellow, color.Bold).Sprint("⚠")
		sb.WriteString("\n" + colorHeader("Safety:") + "\n")
		fmt.Fprintf(&sb, "  %s Requires confirmation: %s\n", warn, task.Confirmation)
	}

	return strings.TrimRight(sb.String(), "\n")
}

func formatOptionPrefix(short, long string) string {
	if short != "" {
		return fmt.Sprintf("  %s, %s", short, long)
	}
	return fmt.Sprintf("      %s", long)
}

func formatOptionsTable(sb *strings.Builder, items []optionItem, colWidth int) {
	for _, opt := range items {
		prefix := formatOptionPrefix(opt.short, opt.long)
		padding := strings.Repeat(" ", max(2, colWidth-len(prefix)))
		if opt.short != "" {
			fmt.Fprintf(sb, "  %s%s%s\n", colorOption(opt.short+", "+opt.long), padding, opt.desc)
		} else {
			fmt.Fprintf(sb, "      %s%s%s\n", colorOption(opt.long), padding, opt.desc)
		}
	}
}

func formatTaskUsageArtisan(task *ast.Task) string {
	var parts []string
	parts = append(parts, task.Name, "[options]")

	if len(task.Parameters) > 0 || task.Passthrough != nil {
		parts = append(parts, "[--]")
		for _, param := range task.Parameters {
			if param.HasDefault {
				parts = append(parts, fmt.Sprintf("[<%s>]", param.Name))
			} else {
				parts = append(parts, fmt.Sprintf("<%s>", param.Name))
			}
		}
		if task.Passthrough != nil {
			parts = append(parts, fmt.Sprintf("[<%s>...]", task.Passthrough.Name))
		}
	}

	return strings.Join(parts, " ")
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
