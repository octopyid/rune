package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRunVersion(t *testing.T) {
	code := run([]string{"--version"})
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestRunHelp(t *testing.T) {
	code := run([]string{"--help"})
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestRunTaskExecution(t *testing.T) {
	tmpDir := t.TempDir()
	rf := filepath.Join(tmpDir, "Runefile")
	content := `
#[Test task]
hello:
    echo "world"
`
	if err := os.WriteFile(rf, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	// Run with explicit file
	code := run([]string{"-f", rf, "hello"})
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}

	// Dry run with explicit file
	code = run([]string{"-f", rf, "--dry-run", "hello"})
	if code != 0 {
		t.Errorf("expected exit code 0, got %d", code)
	}
}

func TestRunErrors(t *testing.T) {
	// Missing file
	code := run([]string{"-f", "/non/existent/Runefile", "hello"})
	if code != 1 {
		t.Errorf("expected exit code 1 for missing file, got %d", code)
	}

	// Unknown option
	code = run([]string{"--invalid-flag"})
	if code != 1 {
		t.Errorf("expected exit code 1 for invalid flag, got %d", code)
	}
}
