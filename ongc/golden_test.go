// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

package ongc

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "regenerate golden files from PyOngc")

func TestMain(m *testing.M) {
	flag.Parse()
	if *update {
		cmd := exec.Command("python3", "scripts/generate_testdata.py")
		cmd.Dir = ".."
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			fmt.Fprintln(os.Stderr, "golden generation failed:", err)
			os.Exit(1)
		}
	}
	os.Exit(m.Run())
}

// buildBinary compiles cmd/ongc to a temp dir and returns the binary path.
// t.Cleanup removes the binary when the test finishes.
func buildBinary(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	bin := filepath.Join(dir, "ongc")
	out, err := exec.Command("go", "build", "-o", bin, "../cmd/ongc").CombinedOutput()
	if err != nil {
		t.Fatalf("build failed: %v\n%s", err, out)
	}
	return bin
}

// skipHeader strips lines before the first type-count line in stats output.
// The type-count line pattern is: starts with a capital letter, ends with "-> N".
// Used for stats.txt only since the version header differs between implementations.
func skipHeader(s string) string {
	lines := strings.Split(s, "\n")
	re := regexp.MustCompile(`^\s*[A-Z].*-> [0-9]+$`)
	for i, line := range lines {
		if re.MatchString(line) {
			return strings.Join(lines[i:], "\n")
		}
	}
	return s
}

type commandEntry struct {
	Filename string   `json:"filename"`
	Args     []string `json:"args"`
}

func loadManifest(t *testing.T, path string) []commandEntry {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading commands.json: %v", err)
	}
	var entries []commandEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		t.Fatalf("parsing commands.json: %v", err)
	}
	return entries
}

func runBinary(t *testing.T, bin string, args []string) string {
	t.Helper()
	cmd := exec.Command(bin, args...)
	out, err := cmd.Output()
	if err != nil {
		// Non-zero exit is not fatal for golden comparison — capture stdout regardless
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			// Return stdout even on non-zero exit (some commands print to stdout then exit 1)
			_ = exitErr
		} else {
			t.Fatalf("running binary: %v", err)
		}
	}
	return string(out)
}

func readGolden(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading golden file %s: %v", path, err)
	}
	return string(data)
}

func TestGolden(t *testing.T) {
	manifestPath := "../testdata/golden/commands.json"
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		t.Skip("testdata/golden/commands.json not found; run: go test ./ongc/... -run TestGolden -update")
	}

	bin := buildBinary(t)
	entries := loadManifest(t, manifestPath)

	for _, entry := range entries {
		entry := entry
		t.Run(entry.Filename, func(t *testing.T) {
			got := runBinary(t, bin, entry.Args)
			want := readGolden(t, filepath.Join("../testdata/golden", entry.Filename))
			if entry.Filename == "stats.txt" {
				got = skipHeader(got)
				want = skipHeader(want)
			}
			if got != want {
				t.Errorf("output mismatch for %s\ngot:\n%s\nwant:\n%s\ndiff: use -update to regenerate", entry.Filename, got, want)
			}
		})
	}
}

// TestViewDetailsOrder ensures --details works whether it appears before or
// after the object name ("view NAME --details" was silently ignored previously).
func TestViewDetailsOrder(t *testing.T) {
	bin := buildBinary(t)

	before := runBinary(t, bin, []string{"view", "--details", "NGC7000"})
	after := runBinary(t, bin, []string{"view", "NGC7000", "--details"})

	if before != after {
		t.Errorf("--details output differs by flag position\nbefore:\n%s\nafter:\n%s", before, after)
	}
	if !strings.Contains(before, "+---") {
		t.Errorf("expected detail box in output, got:\n%s", before)
	}
}
