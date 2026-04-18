// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

package ongc

import (
	"fmt"
	"math"
	"testing"
)

// bruteCone is the reference implementation — the pre-G3 scan that
// iterated the full objects slice. Used solely by the parity test to
// prove the Dec-band index returns identical results.
func bruteCone(raRad, decRad, radiusArcmin float64, catalog string, excludeID int) []neighbor {
	radiusDeg := radiusArcmin / 60.0
	neighbors := make([]neighbor, 0, 32)
	for _, obj := range objects {
		if excludeID != 0 && obj.ID == excludeID {
			continue
		}
		if !hasCoords(obj) {
			continue
		}
		if !matchCatalog(obj, catalog) {
			continue
		}
		dist := angularSep(raRad, decRad, obj.RA, obj.Dec)
		if dist <= radiusDeg {
			neighbors = append(neighbors, neighbor{obj: obj, dist: dist})
		}
	}
	return neighbors
}

// TestDecBandIndexMatchesBruteForce locks in the G3 invariant: the
// Dec-band-pruned coneSearch returns the EXACT same set of objects as
// the brute-force scan for a stratified sample of query points.
//
// Covers the equator, polar caps, and the band boundaries at Dec = k*10°.
// If this test ever fails, the index lost parity and must not ship.
func TestDecBandIndexMatchesBruteForce(t *testing.T) {
	const toRad = math.Pi / 180
	cases := []struct {
		name     string
		raDeg    float64
		decDeg   float64
		radius   float64 // arcmin
		catalog  string
	}{
		// Equator, mid-latitude, poles, and all band boundaries.
		{"equator", 0, 0, 60, "all"},
		{"equator_IC", 150, 0, 30, "IC"},
		{"equator_NGC", 220, 0, 30, "NGC"},
		{"mid_north", 80, 45, 60, "all"},
		{"mid_south", 200, -45, 120, "all"},
		{"near_north_pole", 45, 85, 60, "all"},
		{"near_south_pole", 190, -85, 90, "all"},
		{"over_north_pole", 30, 89.5, 120, "all"}, // radius crosses pole
		{"over_south_pole", 200, -89.5, 90, "all"},
		{"band_boundary_pos", 120, 10, 45, "all"}, // exactly at Dec=+10°
		{"band_boundary_neg", 300, -10, 45, "all"},
		{"small_radius", 0, 27.7, 5, "all"}, // NGC0001 at ~27.7°, tight cone
		{"large_radius", 180, 0, 600, "all"}, // spans ±10° (two bands)
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raRad := tc.raDeg * toRad
			decRad := tc.decDeg * toRad
			idxed, _, err := coneSearch(raRad, decRad, tc.radius, tc.catalog, 0)
			if err != nil {
				t.Fatalf("coneSearch: %v", err)
			}
			brute := bruteCone(raRad, decRad, tc.radius, tc.catalog, 0)

			if len(idxed) != len(brute) {
				t.Fatalf("result length: indexed=%d, brute=%d", len(idxed), len(brute))
			}
			// coneSearch sorts by distance; bruteCone is unsorted. Compare
			// by ID set instead of position.
			idSet := map[int]bool{}
			for _, n := range brute {
				idSet[n.obj.ID] = true
			}
			for _, o := range idxed {
				if !idSet[o.ID] {
					t.Errorf("indexed result %q (ID=%d) not in brute-force set", o.Name, o.ID)
				}
				delete(idSet, o.ID)
			}
			if len(idSet) > 0 {
				missing := []string{}
				for id := range idSet {
					for _, n := range brute {
						if n.obj.ID == id {
							missing = append(missing, fmt.Sprintf("%s(ID=%d)", n.obj.Name, id))
						}
					}
				}
				t.Errorf("brute-force found %d objects not in indexed set: %v", len(idSet), missing)
			}
		})
	}
}

// TestDecBandIndexCoverage confirms every coord-bearing object ends up in
// exactly one band and nothing is lost. Preconditions for correctness.
func TestDecBandIndexCoverage(t *testing.T) {
	seen := map[int]int{} // obj.ID → band index it appears in
	for band, objs := range decBands {
		for _, obj := range objs {
			if prev, dup := seen[obj.ID]; dup {
				t.Errorf("object %q (ID=%d) appears in bands %d AND %d", obj.Name, obj.ID, prev, band)
			}
			seen[obj.ID] = band
		}
	}

	// Every object with coords should be in exactly one band; no-coord
	// objects should be in zero bands.
	for _, obj := range objects {
		_, inBand := seen[obj.ID]
		if hasCoords(obj) && !inBand {
			t.Errorf("coord-bearing object %q (ID=%d) missing from decBands", obj.Name, obj.ID)
		}
		if !hasCoords(obj) && inBand {
			t.Errorf("no-coord object %q (ID=%d) unexpectedly in decBands[%d]", obj.Name, obj.ID, seen[obj.ID])
		}
	}
}

// TestDecBandIndexFunction covers the index-computation edges that
// downstream pruning relies on.
func TestDecBandIndexFunction(t *testing.T) {
	cases := []struct {
		decDeg   float64
		wantBand int
	}{
		{-90, 0},    // south pole
		{-89.999, 0},
		{-80, 1},    // exactly at -80 starts band 1
		{-0.1, 8},   // just below zero
		{0, 9},      // equator
		{10, 10},    // band boundary
		{89.999, 17},
		{90, 17}, // north pole clamps to last band
		{95, 17}, // out of range clamps
		{-95, 0},
	}
	for _, tc := range cases {
		got := decBandIndex(tc.decDeg * math.Pi / 180)
		if got != tc.wantBand {
			t.Errorf("decBandIndex(%.3f°) = %d, want %d", tc.decDeg, got, tc.wantBand)
		}
	}
}
