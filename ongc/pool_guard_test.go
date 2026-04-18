// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

package ongc

import (
	"strings"
	"testing"
)

// TestExpectPoolLenMatches is the happy path: matching lengths never panic.
func TestExpectPoolLenMatches(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("expectPoolLen panicked on matching lengths: %v", r)
		}
	}()
	expectPoolLen("MajAx", 14033, 14033)
}

// TestExpectPoolLenMismatch locks in the F5 invariant: a generator-side
// bug that emits a pool with mismatched length must fail fast at init with
// a message naming the offending pool, instead of a cryptic runtime
// index-out-of-range inside wireObjectPointers.
func TestExpectPoolLenMismatch(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expectPoolLen did not panic on length mismatch")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("panic value is not a string: %T %v", r, r)
		}
		if !strings.Contains(msg, "MajAx") {
			t.Errorf("panic message missing pool name: %q", msg)
		}
		if !strings.Contains(msg, "generator bug") {
			t.Errorf("panic message missing diagnostic hint: %q", msg)
		}
	}()
	expectPoolLen("MajAx", 100, 14033)
}

// TestAssertPoolLengthsOnBuiltCatalog confirms the real catalog passes —
// which is the evidence that wireObjectPointers ran safely at init time.
func TestAssertPoolLengthsOnBuiltCatalog(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("assertPoolLengths on the real catalog panicked: %v", r)
		}
	}()
	assertPoolLengths()
}
