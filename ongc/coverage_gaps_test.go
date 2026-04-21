// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

package ongc

import (
	"math"
	"testing"
)

// H1: direct test for wireObjectPointers side-effects.
//
// The rest of the catalog exercises wireObjectPointers indirectly via
// PrintDetails / filter tests, but nothing asserts that specific pointer
// fields match specific pool values with specific addresses. This test
// makes that wiring explicit so a future refactor can't accidentally
// silence the check by just keeping PrintDetails working.
func TestWireObjectPointersWiresFromPools(t *testing.T) {
	obj, ok := byName["NGC0001"]
	if !ok {
		t.Fatal("NGC0001 not found in byName index")
	}

	// Spot-check: NGC0001 has finite values for MajAx, BMag, VMag, RadVel.
	// Every non-nil field must point into the corresponding pool slot.
	cases := []struct {
		name string
		got  *float64
		pool *float64 // address of the pool entry for NGC0001 (row 0+... lookup)
	}{
		{"MajAx", obj.MajAx, &poolMajAx[obj.ID-1]},
		{"BMag", obj.BMag, &poolBMag[obj.ID-1]},
		{"VMag", obj.VMag, &poolVMag[obj.ID-1]},
		{"RadVel", obj.RadVel, &poolRadVel[obj.ID-1]},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.got == nil {
				t.Fatalf("%s pointer nil — fixture precondition broken", tc.name)
			}
			if tc.got != tc.pool {
				t.Errorf("%s points to %p, want pool entry at %p",
					tc.name, tc.got, tc.pool)
			}
		})
	}

	// Fields that are NaN in the pool must be nil on the Object.
	// CstarUMag is 99.9% NaN; NGC0001 is not a PN.
	if obj.CstarUMag != nil {
		t.Errorf("NGC0001.CstarUMag = %v, want nil (non-PN row, pool is NaN)", *obj.CstarUMag)
	}
}

// H2: explicit FormatDec pole tests. The existing TestFormatDec covers
// NGC0001 and an out-of-range clamp case, but not ±π/2 exactly.
func TestFormatDecPoles(t *testing.T) {
	cases := []struct {
		name string
		rad  float64
		want string
	}{
		{"exact_north_pole", math.Pi / 2, "+90:00:00.0"},
		{"exact_south_pole", -math.Pi / 2, "-90:00:00.0"},
		{"just_under_north_pole", math.Pi/2 - 1e-9, "+90:00:00.0"},
		{"just_under_south_pole", -math.Pi/2 + 1e-9, "-90:00:00.0"},
		{"clamp_above_north", math.Pi, "+90:00:00.0"},  // 180° clamps
		{"clamp_below_south", -math.Pi, "-90:00:00.0"}, // -180° clamps
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := FormatDec(tc.rad); got != tc.want {
				t.Errorf("FormatDec(%.9f) = %q, want %q", tc.rad, got, tc.want)
			}
		})
	}
}

// H3: InRegion with both an antimeridian RA wrap AND a restricted Dec
// band. The existing TestInRegion covers wrap with the full Dec range.
func TestInRegionAntimeridianWithDecBand(t *testing.T) {
	// Query: RA in [350°, 10°] (wrap) AND Dec in [30°, 60°].
	// Must return at least one object with RA ≥ 350 AND one with RA ≤ 10,
	// and every result must fall inside both bounds.
	results := InRegion(350, 10, 30, 60)
	if len(results) == 0 {
		t.Fatal("expected objects in wrap × Dec-band region, got none")
	}

	var hasHigh, hasLow bool
	for _, obj := range results {
		raDeg := obj.RADeg()
		decDeg := obj.DecDeg()
		if decDeg < 30 || decDeg > 60 {
			t.Errorf("object %s Dec=%.4f outside [30,60]", obj.Name, decDeg)
		}
		// RA must be in [350, 360) or [0, 10].
		if !((raDeg >= 350 && raDeg < 360) || (raDeg >= 0 && raDeg <= 10)) {
			t.Errorf("object %s RA=%.4f outside wrap range [350,10]", obj.Name, raDeg)
		}
		if raDeg >= 350 {
			hasHigh = true
		}
		if raDeg <= 10 {
			hasLow = true
		}
	}
	if !hasHigh {
		t.Error("expected at least one object with RA ≥ 350 in the wrap region")
	}
	if !hasLow {
		t.Error("expected at least one object with RA ≤ 10 in the wrap region")
	}
}

// H4: antipodal angularSep — two points exactly 180° apart.
// Vincenty's formula is numerically stable everywhere including the
// antipodal case, where simpler formulas (law of cosines) break.
func TestAngularSepAntipodal(t *testing.T) {
	cases := []struct {
		name                 string
		ra1, dec1, ra2, dec2 float64
	}{
		{"equator_to_equator", 0, 0, math.Pi, 0},
		{"north_to_south_pole", 0, math.Pi / 2, 0, -math.Pi / 2},
		{"arbitrary", math.Pi / 4, math.Pi / 6, math.Pi + math.Pi/4, -math.Pi / 6},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := angularSep(tc.ra1, tc.dec1, tc.ra2, tc.dec2)
			if math.Abs(got-180.0) > 1e-9 {
				t.Errorf("angularSep antipodal: got %.12f°, want 180° (diff %.2e)",
					got, math.Abs(got-180.0))
			}
		})
	}
}
