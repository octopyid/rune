package env

import (
	"fmt"
	"io"
	"maps"
	"os"
	"path/filepath"
	"strings"

	"github.com/joho/godotenv"
)

// LoadDotEnv loads key-value pairs from a .env file if it exists in dir.
// If the file does not exist, an empty map is returned with no error.
func LoadDotEnv(dir string) (map[string]string, error) {
	envPath := filepath.Join(dir, ".env")
	env, err := godotenv.Read(envPath)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil
		}
		return nil, err
	}
	return env, nil
}

// ParseDotEnv parses .env formatted content from an io.Reader.
func ParseDotEnv(r io.Reader) (map[string]string, error) {
	return godotenv.Parse(r)
}

// BuildEnvironment merges OS environment, .env variables, and task variables.
func BuildEnvironment(baseEnv []string, dotEnv map[string]string, taskEnv map[string]string) []string {
	envMap := make(map[string]string)

	// 1. Base OS environment
	for _, entry := range baseEnv {
		before, after, ok := strings.Cut(entry, "=")
		if ok {
			envMap[before] = after
		}
	}

	// 2. .env file variables (applied if not already explicitly defined in OS, or sets defaults)
	for k, v := range dotEnv {
		if _, exists := envMap[k]; !exists {
			envMap[k] = v
		}
	}

	// 3. Task variables (always take highest precedence)
	maps.Copy(envMap, taskEnv)

	// Format back into KEY=VALUE slice
	res := make([]string, 0, len(envMap))
	for k, v := range envMap {
		res = append(res, fmt.Sprintf("%s=%s", k, v))
	}
	return res
}
