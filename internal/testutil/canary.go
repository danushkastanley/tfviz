// Package testutil holds helpers shared by tests.
package testutil

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// RepoPath resolves a path relative to the repository root.
func RepoPath(rel string) string {
	_, file, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(file), "..", "..", rel)
}

// Canaries returns the planted secret literals from testdata/canaries.txt.
func Canaries(t testing.TB) []string {
	t.Helper()
	data, err := os.ReadFile(RepoPath("testdata/canaries.txt"))
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			out = append(out, line)
		}
	}
	if len(out) == 0 {
		t.Fatal("no canaries defined")
	}
	return out
}

// AssertNoCanaries fails if any planted secret appears in data.
func AssertNoCanaries(t testing.TB, what string, data []byte) {
	t.Helper()
	for _, canary := range Canaries(t) {
		if strings.Contains(string(data), canary) {
			t.Errorf("%s contains canary %s", what, canary)
		}
	}
}
