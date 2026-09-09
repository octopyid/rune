package parser

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"

	"github.com/octopyid/rune/internal/ast"
)

// ParseError represents an error during Runefile parsing.
type ParseError struct {
	File    string
	Line    int
	Message string
}

func (e *ParseError) Error() string {
	if e.File != "" {
		return fmt.Sprintf("%s:%d: %s", e.File, e.Line, e.Message)
	}
	return fmt.Sprintf("line %d: %s", e.Line, e.Message)
}

// ParseFile parses a Runefile from the filesystem.
func ParseFile(filePath string) (*ast.File, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	return Parse(f, filePath)
}

// Parse parses a Runefile from an io.Reader.
func Parse(r io.Reader, filePath string) (*ast.File, error) {
	scanner := bufio.NewScanner(r)
	var (
		tasks       []*ast.Task
		currentTask *ast.Task
		lineNum     int

		pendingDesc    string
		pendingConfirm string

		taskLineMap = make(map[string]int)
	)

	for scanner.Scan() {
		lineNum++
		rawLine := scanner.Text()

		// 1. Check indentation
		isIndented := len(rawLine) > 0 && (rawLine[0] == ' ' || rawLine[0] == '\t')
		trimmed := strings.TrimSpace(rawLine)

		// Blank lines
		if trimmed == "" {
			continue
		}

		// 2. Indented lines belong to current task body
		if isIndented {
			if currentTask == nil {
				return nil, &ParseError{
					File:    filePath,
					Line:    lineNum,
					Message: "unexpected command line outside of a task definition",
				}
			}
			// Command line inside current task
			currentTask.Commands = append(currentTask.Commands, trimmed)
			continue
		}

		// Once we hit a non-indented line, current task is finished
		currentTask = nil

		// 3. Metadata attributes: #[...]
		if strings.HasPrefix(trimmed, "#[") && strings.HasSuffix(trimmed, "]") {
			content := strings.TrimSpace(trimmed[2 : len(trimmed)-1])
			if after, ok := strings.CutPrefix(content, "confirm:"); ok {
				msg := strings.TrimSpace(after)
				pendingConfirm = msg
			} else if after, ok := strings.CutPrefix(content, "description:"); ok {
				desc := strings.TrimSpace(after)
				pendingDesc = desc
			} else {
				// Bare metadata #[Build the application] is treated as description
				pendingDesc = content
			}
			continue
		}

		// 4. Regular comments: # ...
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		// 5. Task header line
		task, err := parseTaskHeader(trimmed, lineNum, filePath)
		if err != nil {
			return nil, err
		}

		// Check for duplicate task names
		if prevLine, exists := taskLineMap[task.Name]; exists {
			return nil, &ParseError{
				File:    filePath,
				Line:    lineNum,
				Message: fmt.Sprintf("duplicate task '%s' (previously defined at line %d)", task.Name, prevLine),
			}
		}
		taskLineMap[task.Name] = lineNum

		// Attach accumulated metadata
		task.Description = pendingDesc
		task.Confirmation = pendingConfirm
		pendingDesc = ""
		pendingConfirm = ""

		tasks = append(tasks, task)
		currentTask = task
	}

	if err := scanner.Err(); err != nil {
		return nil, &ParseError{
			File:    filePath,
			Line:    lineNum,
			Message: err.Error(),
		}
	}

	return ast.NewFile(filePath, tasks), nil
}

// parseTaskHeader parses the signature and dependencies of a task.
// Format: <task_name> [params...] : [dep1 dep2 ...]
func parseTaskHeader(line string, lineNum int, filePath string) (*ast.Task, error) {
	// Find the separator colon.
	// The separator colon is the colon after the task name and parameters.
	// Note: task name can contain namespaces like "db:fresh".
	colonIdx, err := findSignatureColon(line)
	if err != nil {
		return nil, &ParseError{
			File:    filePath,
			Line:    lineNum,
			Message: err.Error(),
		}
	}
	if colonIdx == -1 {
		return nil, &ParseError{
			File:    filePath,
			Line:    lineNum,
			Message: "missing colon ':' in task signature",
		}
	}

	sigPart := strings.TrimSpace(line[:colonIdx])
	depPart := strings.TrimSpace(line[colonIdx+1:])

	if sigPart == "" {
		return nil, &ParseError{
			File:    filePath,
			Line:    lineNum,
			Message: "missing task name before colon",
		}
	}

	// Tokenize sigPart
	tokens, err := tokenizeSignature(sigPart)
	if err != nil {
		return nil, &ParseError{
			File:    filePath,
			Line:    lineNum,
			Message: err.Error(),
		}
	}

	if len(tokens) == 0 {
		return nil, &ParseError{
			File:    filePath,
			Line:    lineNum,
			Message: "missing task name in signature",
		}
	}

	taskName := tokens[0]
	if err := validateTaskName(taskName); err != nil {
		return nil, &ParseError{
			File:    filePath,
			Line:    lineNum,
			Message: err.Error(),
		}
	}

	// Determine namespace and short name
	var namespace, shortName string
	if lastColon := strings.LastIndex(taskName, ":"); lastColon != -1 {
		namespace = taskName[:lastColon]
		shortName = taskName[lastColon+1:]
	} else {
		shortName = taskName
	}

	task := &ast.Task{
		Name:      taskName,
		Namespace: namespace,
		ShortName: shortName,
		Line:      lineNum,
	}

	// Parse parameters (tokens[1:])
	seenDefault := false
	for _, tok := range tokens[1:] {
		// Passthrough argument *args
		if after, ok := strings.CutPrefix(tok, "*"); ok {
			passthroughName := after
			if passthroughName == "" {
				return nil, &ParseError{
					File:    filePath,
					Line:    lineNum,
					Message: "passthrough argument name cannot be empty",
				}
			}
			if task.Passthrough != nil {
				return nil, &ParseError{
					File:    filePath,
					Line:    lineNum,
					Message: "only one passthrough argument (*args) is allowed per task",
				}
			}
			task.Passthrough = &ast.Passthrough{Name: passthroughName}
			continue
		}

		// Passthrough must be last
		if task.Passthrough != nil {
			return nil, &ParseError{
				File:    filePath,
				Line:    lineNum,
				Message: fmt.Sprintf("argument '%s' cannot follow passthrough argument '*%s'", tok, task.Passthrough.Name),
			}
		}

		// Boolean flag: --race? or --race
		if after, ok := strings.CutPrefix(tok, "--"); ok {
			flagRaw := after
			flagName := strings.TrimSuffix(flagRaw, "?")
			if flagName == "" {
				return nil, &ParseError{
					File:    filePath,
					Line:    lineNum,
					Message: "flag name cannot be empty",
				}
			}
			task.Flags = append(task.Flags, ast.Flag{Name: flagName})
			continue
		}

		// Default argument: target="dev" or target=dev
		if before, after, ok := strings.Cut(tok, "="); ok {
			paramName := before
			defaultVal := unquote(after)
			if paramName == "" {
				return nil, &ParseError{
					File:    filePath,
					Line:    lineNum,
					Message: "parameter name cannot be empty in default argument",
				}
			}
			seenDefault = true
			task.Parameters = append(task.Parameters, ast.Parameter{
				Name:         paramName,
				DefaultValue: defaultVal,
				HasDefault:   true,
			})
			continue
		}

		// Required argument
		if seenDefault {
			return nil, &ParseError{
				File:    filePath,
				Line:    lineNum,
				Message: fmt.Sprintf("required argument '%s' cannot follow default argument", tok),
			}
		}
		task.Parameters = append(task.Parameters, ast.Parameter{
			Name:       tok,
			HasDefault: false,
		})
	}

	// Parse dependencies
	if depPart != "" {
		for _, dep := range strings.Fields(depPart) {
			if err := validateTaskName(dep); err != nil {
				return nil, &ParseError{
					File:    filePath,
					Line:    lineNum,
					Message: fmt.Sprintf("invalid dependency name '%s': %s", dep, err),
				}
			}
			task.Dependencies = append(task.Dependencies, dep)
		}
	}

	return task, nil
}

// findSignatureColon locates the delimiter colon in a task signature line.
// Colon within namespace identifiers (e.g. db:fresh) does NOT have whitespace or follows an identifier.
// The signature delimiter colon is either followed by whitespace or is at the end of the signature.
func findSignatureColon(line string) (int, error) {
	var inQuotes rune
	for i := 0; i < len(line); i++ {
		ch := rune(line[i])
		if inQuotes != 0 {
			if ch == inQuotes {
				inQuotes = 0
			}
			continue
		}
		if ch == '"' || ch == '\'' {
			inQuotes = ch
			continue
		}

		if ch == ':' {
			// Check if this colon is part of a namespace identifier:
			// e.g. "db:fresh". In a namespace, preceding char is [a-zA-Z0-9_-] and succeeding char is [a-zA-Z0-9_-].
			if i > 0 && isIdentChar(rune(line[i-1])) && i+1 < len(line) && isIdentChar(rune(line[i+1])) {
				// Internal namespace colon, skip it
				continue
			}
			return i, nil
		}
	}
	if inQuotes != 0 {
		return -1, fmt.Errorf("unclosed quote in task signature")
	}
	return -1, nil
}

func isIdentChar(ch rune) bool {
	return unicode.IsLetter(ch) || unicode.IsDigit(ch) || ch == '_' || ch == '-'
}

func validateTaskName(name string) error {
	if name == "" {
		return fmt.Errorf("task name cannot be empty")
	}
	for _, part := range strings.Split(name, ":") {
		if part == "" {
			return fmt.Errorf("empty namespace segment in '%s'", name)
		}
		for _, ch := range part {
			if !isIdentChar(ch) {
				return fmt.Errorf("invalid character '%c' in task name '%s'", ch, name)
			}
		}
	}
	return nil
}

// tokenizeSignature splits a signature string into tokens, preserving quoted substrings.
func tokenizeSignature(sig string) ([]string, error) {
	var (
		tokens  []string
		current strings.Builder
		inQuote rune
	)

	for _, ch := range sig {
		if inQuote != 0 {
			if ch == inQuote {
				inQuote = 0
			}
			current.WriteRune(ch)
			continue
		}

		if ch == '"' || ch == '\'' {
			inQuote = ch
			current.WriteRune(ch)
			continue
		}

		if unicode.IsSpace(ch) {
			if current.Len() > 0 {
				tokens = append(tokens, current.String())
				current.Reset()
			}
			continue
		}

		current.WriteRune(ch)
	}

	if inQuote != 0 {
		return nil, fmt.Errorf("unclosed quote in signature: %s", sig)
	}

	if current.Len() > 0 {
		tokens = append(tokens, current.String())
	}

	return tokens, nil
}

func unquote(s string) string {
	if len(s) >= 2 {
		if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
			return s[1 : len(s)-1]
		}
	}
	return s
}
