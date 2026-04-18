// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

package ongc

import (
	"math"
	"os"
	"strings"
	"testing"
	"unicode/utf8"
)

func TestString(t *testing.T) {
	obj := byName["NGC0001"]
	if obj == nil {
		t.Fatal("NGC0001 not found")
	}
	got := obj.String()
	want := "NGC0001, Galaxy in Peg"
	if got != want {
		t.Errorf("String() = %q, want %q", got, want)
	}
}

func TestPrintDetailsBox(t *testing.T) {
	obj := byName["NGC0001"]
	if obj == nil {
		t.Fatal("NGC0001 not found")
	}
	out := obj.PrintDetails()
	lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
	for i, line := range lines {
		if utf8.RuneCountInString(line) != 79 {
			t.Errorf("line %d has visual width %d, want 79: %q", i+1, utf8.RuneCountInString(line), line)
		}
	}
}

func TestPrintDetailsGolden(t *testing.T) {
	if _, err := os.Stat("../testdata/golden"); os.IsNotExist(err) {
		t.Skip("testdata/golden/ not found; run: go test ./ongc/... -run TestGolden -update")
	}
	// Golden comparison is done by TestGolden in golden_test.go via CLI binary.
	// This test just verifies the golden dir is present.
}

func TestPrintDetailsAllLines79(t *testing.T) {
	// Sample a few objects of different types to verify 79-char invariant.
	names := []string{"NGC0001", "NGC7293", "NGC0869"}
	for _, name := range names {
		obj := byName[name]
		if obj == nil {
			t.Logf("skipping %s: not found", name)
			continue
		}
		t.Run(name, func(t *testing.T) {
			out := obj.PrintDetails()
			lines := strings.Split(strings.TrimRight(out, "\n"), "\n")
			for i, line := range lines {
				if utf8.RuneCountInString(line) != 79 {
					t.Errorf("line %d has visual width %d, want 79: %q", i+1, utf8.RuneCountInString(line), line)
				}
			}
		})
	}
}

// TestPyFloatNaNInfLowercase locks in the B13 invariant: pyFloat must
// emit Python-style "nan"/"inf"/"-inf" for non-finite values so byte-exact
// PyOngc parity holds if any field ever contains NaN or Inf. Go's
// FormatFloat emits "NaN"/"+Inf"/"-Inf", which would diverge.
func TestPyFloatNaNInfLowercase(t *testing.T) {
	cases := []struct {
		v    float64
		want string
	}{
		{math.NaN(), "nan"},
		{math.Inf(1), "inf"},
		{math.Inf(-1), "-inf"},
		{0, "0.0"},
		{1.57, "1.57"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			if got := pyFloat(tc.v); got != tc.want {
				t.Errorf("pyFloat(%v) = %q, want %q", tc.v, got, tc.want)
			}
		})
	}
}

// TestJustifyTextRuneWidth locks in the B9 invariant: justifyText counts
// characters by rune, not byte, so multibyte text (e.g. Greek letters in
// common names) doesn't break the 79-column box alignment. This is a
// latent bug today — the catalog is ASCII-only — but the test prevents
// regressions if upstream OpenNGC ever introduces non-ASCII names.
func TestJustifyTextRuneWidth(t *testing.T) {
	// "αβγδε" — 5 runes, 10 bytes in UTF-8.
	out := justifyText("αβγδε γ Persei double cluster")
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		if utf8.RuneCountInString(line) != 79 {
			t.Errorf("line has visual width %d (byte len %d), want 79: %q",
				utf8.RuneCountInString(line), len(line), line)
		}
	}
}
