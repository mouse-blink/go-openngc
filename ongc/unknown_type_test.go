// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

package ongc

import "testing"

// TestUnknownTypePassThrough locks in the B3 invariant: an object whose Type
// is not in TypeNames (e.g. a future upstream code like "QSO") must render
// its raw type string in String() and PrintDetails, not an empty placeholder.
// PyOngc v1.2.0 stores the type field as a raw string and passes it through
// unchanged; skipping the row would be a parity divergence.
func TestUnknownTypePassThrough(t *testing.T) {
	obj := &Object{
		ID:    99999,
		Name:  "TEST9999",
		Type:  ObjectType("QSO"),
		Const: "Peg",
	}

	const want = "TEST9999, QSO in Peg"
	if got := obj.String(); got != want {
		t.Errorf("String() = %q, want %q (unknown type should pass through)", got, want)
	}

	if got := obj.TypeLongName(); got != "QSO" {
		t.Errorf("TypeLongName() = %q, want %q", got, "QSO")
	}

	// PrintDetails header must render the raw code, not an empty Type field.
	details := obj.PrintDetails()
	if !containsSubstring(details, "Type: QSO") {
		t.Errorf("PrintDetails output missing raw type header: %s", details)
	}
}

func containsSubstring(haystack, needle string) bool {
	for i := 0; i+len(needle) <= len(haystack); i++ {
		if haystack[i:i+len(needle)] == needle {
			return true
		}
	}
	return false
}
