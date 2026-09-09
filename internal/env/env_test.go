package env

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseDotEnv(t *testing.T) {
	input := `
# Comment
DB_HOST=localhost
DB_PORT=5432
export APP_ENV="production"
SINGLE_QUOTED='hello world'
UNQUOTED=some_value
EMPTY=
# Another comment
`
	env, err := ParseDotEnv(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := map[string]string{
		"DB_HOST":       "localhost",
		"DB_PORT":       "5432",
		"APP_ENV":       "production",
		"SINGLE_QUOTED": "hello world",
		"UNQUOTED":      "some_value",
		"EMPTY":         "",
	}

	for k, v := range expected {
		if actual, ok := env[k]; !ok || actual != v {
			t.Errorf("key %s: expected '%s', got '%s'", k, v, actual)
		}
	}
}

func TestLoadDotEnvMissing(t *testing.T) {
	tmpDir := t.TempDir()
	env, err := LoadDotEnv(tmpDir)
	if err != nil {
		t.Fatalf("expected no error for missing .env, got %v", err)
	}
	if len(env) != 0 {
		t.Errorf("expected empty map, got %+v", env)
	}
}

func TestLoadDotEnvPresent(t *testing.T) {
	tmpDir := t.TempDir()
	envFile := filepath.Join(tmpDir, ".env")
	if err := os.WriteFile(envFile, []byte("GREETING=hello\n"), 0644); err != nil {
		t.Fatal(err)
	}

	env, err := LoadDotEnv(tmpDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if env["GREETING"] != "hello" {
		t.Errorf("expected GREETING=hello, got %s", env["GREETING"])
	}
}

func TestBuildEnvironmentPrecedence(t *testing.T) {
	baseEnv := []string{"FOO=base", "BAR=base"}
	dotEnv := map[string]string{"BAR": "dotenv", "BAZ": "dotenv"}
	taskEnv := map[string]string{"FOO": "task", "QUX": "task"}

	merged := BuildEnvironment(baseEnv, dotEnv, taskEnv)
	mergedMap := make(map[string]string)
	for _, entry := range merged {
		parts := strings.SplitN(entry, "=", 2)
		mergedMap[parts[0]] = parts[1]
	}

	// Task env overrides base
	if mergedMap["FOO"] != "task" {
		t.Errorf("expected FOO=task, got %s", mergedMap["FOO"])
	}
	// Base env takes precedence over .env
	if mergedMap["BAR"] != "base" {
		t.Errorf("expected BAR=base, got %s", mergedMap["BAR"])
	}
	// .env fills in missing
	if mergedMap["BAZ"] != "dotenv" {
		t.Errorf("expected BAZ=dotenv, got %s", mergedMap["BAZ"])
	}
	// Task sets new
	if mergedMap["QUX"] != "task" {
		t.Errorf("expected QUX=task, got %s", mergedMap["QUX"])
	}
}
