package ui

import (
	"bytes"
	"strings"
	"testing"

	"github.com/fatih/color"
)

func TestComponents(t *testing.T) {
	color.NoColor = true
	defer func() { color.NoColor = false }()

	var buf bytes.Buffer

	Info(&buf, "application started")
	if !strings.HasPrefix(buf.String(), "\n") || !strings.Contains(buf.String(), "INFO  application started") {
		t.Errorf("expected Info to have margin-top and contain 'INFO  application started', got: %q", buf.String())
	}

	buf.Reset()
	Warn(&buf, "low disk space")
	if !strings.HasPrefix(buf.String(), "\n") || !strings.Contains(buf.String(), "WARN  low disk space") {
		t.Errorf("expected Warn to have margin-top and contain 'WARN  low disk space', got: %q", buf.String())
	}

	buf.Reset()
	Fail(&buf, "connection failed")
	if !strings.HasPrefix(buf.String(), "\n") || !strings.Contains(buf.String(), "FAIL  connection failed") {
		t.Errorf("expected Fail to have margin-top and contain 'FAIL  connection failed', got: %q", buf.String())
	}

	buf.Reset()
	Done(&buf, "deployment finished")
	if !strings.HasPrefix(buf.String(), "\n") || !strings.Contains(buf.String(), "DONE  deployment finished") {
		t.Errorf("expected Done to have margin-top and contain 'DONE  deployment finished', got: %q", buf.String())
	}

	buf.Reset()
	Error(&buf, "fatal error")
	if !strings.Contains(buf.String(), "ERROR  fatal error") {
		t.Errorf("expected Error to output ERROR, got: %q", buf.String())
	}

	buf.Reset()
	Success(&buf, "all good")
	if !strings.Contains(buf.String(), "DONE  all good") {
		t.Errorf("expected Success alias to output DONE, got: %q", buf.String())
	}
}

func TestComponentsWithColor(t *testing.T) {
	color.NoColor = false
	defer func() { color.NoColor = true }()

	var buf bytes.Buffer
	Info(&buf, "test bright white")

	// 44;37m is ANSI Blue background with White text matching Laravel
	if !strings.Contains(buf.String(), "44;37m") {
		t.Errorf("expected ANSI Blue background with white text (44;37m) in output, got: %q", buf.String())
	}
}
