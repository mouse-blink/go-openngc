// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

package ongc

import (
	"os/exec"
	"strings"
	"testing"
)

// TestCLIFlagsAfterPositionals locks in the B1 invariant end-to-end: the
// CLI must accept flags placed after positional arguments across every
// subcommand that takes flags, not just `view --details`.
func TestCLIFlagsAfterPositionals(t *testing.T) {
	bin := buildBinary(t)

	cases := []struct {
		name     string
		args     []string
		wantSub  string // substring expected in stdout
		minLines int    // also assert non-empty result
	}{
		{
			name:     "view details — flag after positional",
			args:     []string{"view", "NGC0001", "--details"},
			wantSub:  "Type: Galaxy",
			minLines: 5,
		},
		{
			name:     "nearby — radius flag after coords",
			args:     []string{"nearby", "20:59:17.14", "+44:31:43.6", "--radius", "30"},
			wantSub:  "search radius of 30 arcmin",
			minLines: 3,
		},
		{
			name:     "neighbors — radius flag after name",
			args:     []string{"neighbors", "NGC7000", "--radius", "30"},
			wantSub:  "search radius of 30 arcmin",
			minLines: 3,
		},
		{
			name:     "neighbors — catalog flag after name",
			args:     []string{"neighbors", "M31", "--catalog", "NGC", "--radius", "60"},
			wantSub:  "search radius of 60 arcmin",
			minLines: 3,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, err := exec.Command(bin, tc.args...).CombinedOutput()
			if err != nil {
				t.Fatalf("CLI failed: %v\n%s", err, out)
			}
			got := string(out)
			if !strings.Contains(got, tc.wantSub) {
				t.Errorf("output missing %q:\n%s", tc.wantSub, got)
			}
			if lines := strings.Count(got, "\n"); lines < tc.minLines {
				t.Errorf("expected at least %d lines of output, got %d:\n%s", tc.minLines, lines, got)
			}
		})
	}
}

// TestCLIRejectsUnknownFlag confirms that reorderFlags does not hoist an
// unknown -foo / --foo token to the front; the stdlib flag parser will
// reject it in its registered position, which is the desired behavior.
func TestCLIRejectsUnknownFlag(t *testing.T) {
	bin := buildBinary(t)

	out, err := exec.Command(bin, "view", "NGC0001", "--bogus-flag").CombinedOutput()
	if err == nil {
		t.Fatalf("expected CLI to fail on --bogus-flag, got success:\n%s", out)
	}
	if !strings.Contains(string(out), "bogus-flag") {
		t.Errorf("expected error to mention the unknown flag; got:\n%s", out)
	}
}
