// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

package ongc

import (
	"errors"
	"math"
	"testing"
)

// TestMissingCoordsAreNaN locks in the B2 invariant: rows whose RA/Dec cannot
// be parsed (missing or malformed in the catalog) must land as NaN, not 0 —
// otherwise they become phantom objects at (0 h, 0°) in spatial queries.
func TestMissingCoordsAreNaN(t *testing.T) {
	// IC1064 is a NonEx IC entry shipped with empty RA/Dec in the catalog;
	// if that ever changes, pick any row with empty RA/Dec fields.
	obj, ok := byName["IC1064"]
	if !ok {
		t.Skip("IC1064 no longer in catalog; update fixture to any object with empty RA/Dec")
	}
	if !math.IsNaN(obj.RA) {
		t.Errorf("IC1064.RA: got %v, want NaN (empty coord must not become 0)", obj.RA)
	}
	if !math.IsNaN(obj.Dec) {
		t.Errorf("IC1064.Dec: got %v, want NaN", obj.Dec)
	}
}

// TestSpatialQueriesSkipNaNObjects locks in the B2 invariant on the spatial
// side: GetNearby / GetNeighbors / InRegion must not treat NaN rows as (0, 0).
func TestSpatialQueriesSkipNaNObjects(t *testing.T) {
	// Query a 1° cap around (0 h, 0°). Any NaN row would otherwise land here.
	objs, _, err := GetNearby("00:00:00.00", "+00:00:00.0", 60, "IC")
	if err != nil {
		t.Fatalf("GetNearby: %v", err)
	}
	for _, o := range objs {
		if math.IsNaN(o.RA) || math.IsNaN(o.Dec) {
			t.Errorf("GetNearby returned NaN-coord object: %q", o.Name)
		}
	}

	// InRegion over the entire sphere: must not include NaN rows either.
	region := InRegion(0, 360, -90, 90)
	for _, o := range region {
		if math.IsNaN(o.RA) || math.IsNaN(o.Dec) {
			t.Errorf("InRegion returned NaN-coord object: %q", o.Name)
		}
	}

	// GetSeparation on an object with NaN coords must return InvalidCoordinates,
	// not a silent 0° separation.
	if _, err := GetSeparation("IC1064", "NGC0001"); err == nil {
		t.Error("GetSeparation(IC1064, NGC0001): expected error, got nil")
	} else {
		var invalid *InvalidCoordinates
		if !errors.As(err, &invalid) {
			t.Errorf("GetSeparation: got %T (%v), want *InvalidCoordinates", err, err)
		}
	}
}

// TestFormatRADecNaN confirms that display formatters return na for NaN,
// matching PyOngc which exposes na for objects with NULL coords.
func TestFormatRADecNaN(t *testing.T) {
	if got := FormatRA(math.NaN()); got != na {
		t.Errorf("FormatRA(NaN) = %q, want %q", got, na)
	}
	if got := FormatDec(math.NaN()); got != na {
		t.Errorf("FormatDec(NaN) = %q, want %q", got, na)
	}
}
