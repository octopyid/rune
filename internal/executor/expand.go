package executor

import (
	"strings"

	"github.com/mattn/go-shellwords"
	"github.com/octopyid/rune/internal/cli"
)

// ExpandCommand parses a command line template and expands task arguments, flags, and passthrough.
// It preserves argument boundaries precisely using POSIX shell lexing rules.
func ExpandCommand(cmdLine string, bound *cli.BoundArgs) ([]string, error) {
	tokens, err := shellwords.Parse(cmdLine)
	if err != nil {
		return nil, err
	}

	var argv []string
	passthroughName := ""
	if bound != nil && bound.Task != nil && bound.Task.Passthrough != nil {
		passthroughName = bound.Task.Passthrough.Name
	}

	for _, tok := range tokens {
		// 1. Check if token is exactly {{passthrough}}
		if passthroughName != "" && tok == "{{"+passthroughName+"}}" {
			argv = append(argv, bound.PassthroughArgs...)
			continue
		}

		// 2. Check if token is a standalone boolean flag placeholder like {{race}} or {{--race}}
		if strings.HasPrefix(tok, "{{") && strings.HasSuffix(tok, "}}") {
			varName := tok[2 : len(tok)-2]
			flagName := strings.TrimPrefix(varName, "--")
			if bound != nil && bound.Task != nil {
				if flag, ok := bound.Task.FindFlag(flagName); ok && !flag.IsValued {
					if bound.Flags[flag.Name] {
						argv = append(argv, "--"+flag.Name)
					}
					// If false, omit token
					continue
				}
			}
		}

		// 3. General variable interpolation in token
		expanded := interpolateString(tok, bound)
		if expanded != "" || len(tok) == 0 {
			argv = append(argv, expanded)
		}
	}

	return argv, nil
}

// interpolateString replaces {{var}} placeholders in s.
func interpolateString(s string, bound *cli.BoundArgs) string {
	if bound == nil {
		return s
	}

	result := s
	// Replace arguments (positional parameters and valued options)
	for k, v := range bound.Arguments {
		placeholder := "{{" + k + "}}"
		result = strings.ReplaceAll(result, placeholder, v)
		phDash := "{{--" + k + "}}"
		result = strings.ReplaceAll(result, phDash, v)
	}

	// Replace boolean flags
	for k, val := range bound.Flags {
		ph1 := "{{" + k + "}}"
		ph2 := "{{--" + k + "}}"
		replacement := ""
		if val {
			replacement = "--" + k
		}
		result = strings.ReplaceAll(result, ph1, replacement)
		result = strings.ReplaceAll(result, ph2, replacement)
	}

	// Replace passthrough if embedded
	if bound.Task != nil && bound.Task.Passthrough != nil {
		ph := "{{" + bound.Task.Passthrough.Name + "}}"
		if strings.Contains(result, ph) {
			result = strings.ReplaceAll(result, ph, strings.Join(bound.PassthroughArgs, " "))
		}
	}

	return result
}
