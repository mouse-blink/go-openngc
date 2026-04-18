// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

package ongc

import (
	"os/exec"
	"strings"
	"testing"
)

// TestCLISearchRangeFlagsUseFSVisit locks in the B8 invariant: the CLI
// must detect whether a range flag was explicitly set (via flag.FlagSet.Visit),
// not by checking its numeric value against 0. Otherwise "--minra 0" is
// indistinguishable from "flag not set", which is a PyOngc-parity gap.
func TestCLISearchRangeFlagsUseFSVisit(t *testing.T) {
	bin := buildBinary(t)

	// --maxra 1 narrows to RA ≤ ~1 rad (~57°), which is a smaller slice than
	// the full catalog. If fs.Visit is not consulted, the flag would be
	// applied only when the value is non-zero in the OLD code — which is
	// the same here. The important test is --minra 0 --maxra 0 below.
	fullOut, err := exec.Command(bin, "search", "--type", "G").CombinedOutput()
	if err != nil {
		t.Fatalf("baseline CLI failed: %v\n%s", err, fullOut)
	}
	fullCount := strings.Count(string(fullOut), "\n")
	if fullCount == 0 {
		t.Fatal("expected baseline search to return rows")
	}

	// "--minra 0 --maxra 0" should filter to objects with RA == 0 exactly.
	// Very unlikely anything matches — but result MUST be smaller than the
	// unfiltered result; if the old sentinel logic comes back, both would
	// be identical.
	narrowOut, err := exec.Command(bin, "search", "--type", "G", "--minra", "0", "--maxra", "0").CombinedOutput()
	if err != nil {
		t.Fatalf("narrow CLI failed: %v\n%s", err, narrowOut)
	}
	narrowCount := strings.Count(string(narrowOut), "\n")
	if narrowCount >= fullCount {
		t.Errorf("--minra 0 --maxra 0 did not narrow the result set (got %d, baseline %d) — fs.Visit not wired?",
			narrowCount, fullCount)
	}
}
