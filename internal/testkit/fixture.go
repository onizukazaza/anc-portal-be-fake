package testkit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// Fixture returns the absolute path to a file inside the testdata/ directory
// relative to the calling test file.
//
//	path := testkit.Fixture(t, "golden", "expected.json")
func Fixture(t testing.TB, parts ...string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(1)
	if !ok {
		t.Fatal("testkit.Fixture: cannot determine caller")
	}
	dir := filepath.Dir(file)
	elems := append([]string{dir, "testdata"}, parts...)
	return filepath.Join(elems...)
}

// LoadJSON reads a JSON file and unmarshals it into dest.
//
//	var users []User
//	testkit.LoadJSON(t, "testdata/users.json", &users)
func LoadJSON(t testing.TB, path string, dest any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("testkit.LoadJSON: read %s: %v", path, err)
	}
	if err := json.Unmarshal(data, dest); err != nil {
		t.Fatalf("testkit.LoadJSON: unmarshal %s: %v", path, err)
	}
}

// Golden compares got against the contents of a golden file.
// If the TESTKIT_UPDATE env var is set, it writes got to the golden file instead.
// Useful for snapshot testing of large outputs (banner, SQL, etc.).
//
//	testkit.Golden(t, "testdata/banner.golden", actualOutput)
func Golden(t testing.TB, goldenPath string, got string) {
	t.Helper()

	if os.Getenv("TESTKIT_UPDATE") != "" {
		dir := filepath.Dir(goldenPath)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("testkit.Golden: mkdir %s: %v", dir, err)
		}
		if err := os.WriteFile(goldenPath, []byte(got), 0o600); err != nil {
			t.Fatalf("testkit.Golden: write %s: %v", goldenPath, err)
		}
		return
	}

	want, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("testkit.Golden: read %s: %v (run with TESTKIT_UPDATE=1 to create)", goldenPath, err)
	}

	if got != string(want) {
		t.Fatalf("testkit.Golden: output differs from %s\n--- want ---\n%s\n--- got ---\n%s",
			goldenPath, string(want), got)
	}
}
