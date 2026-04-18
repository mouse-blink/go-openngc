// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

package ongc

import (
	"errors"
	"math"
	"strings"
	"testing"
)

func TestGetSeparation(t *testing.T) {
	sep, err := GetSeparation("NGC0869", "NGC0884")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sep <= 0.0 {
		t.Errorf("expected positive separation, got %f", sep)
	}
	if sep >= 2.0 {
		t.Errorf("expected separation < 2.0 degrees, got %f", sep)
	}
}

func TestGetSeparationUnknownFirst(t *testing.T) {
	_, err := GetSeparation("NGC9999", "NGC0884")
	if err == nil {
		t.Fatal("expected error for unknown object, got nil")
	}
	var notFound *ObjectNotFound
	if !errors.As(err, &notFound) {
		t.Errorf("expected *ObjectNotFound, got %T: %v", err, err)
	}
}

func TestGetSeparationUnknownSecond(t *testing.T) {
	_, err := GetSeparation("NGC0869", "NGC9999")
	if err == nil {
		t.Fatal("expected error for unknown object, got nil")
	}
	var notFound *ObjectNotFound
	if !errors.As(err, &notFound) {
		t.Errorf("expected *ObjectNotFound, got %T: %v", err, err)
	}
}

func TestGetNeighbors(t *testing.T) {
	results, dists, err := GetNeighbors("NGC7000", 60, "all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != len(dists) {
		t.Fatalf("results and dists length mismatch: %d vs %d", len(results), len(dists))
	}

	ref, _ := Get("NGC7000")
	for _, obj := range results {
		if obj == ref {
			t.Error("reference object NGC7000 must not appear in results")
		}
	}

	const maxDeg = 1.0 // 60 arcmin = 1 degree
	for i, d := range dists {
		if d > maxDeg {
			t.Errorf("dist[%d] = %f exceeds radius of %f degrees", i, d, maxDeg)
		}
	}

	for i := 1; i < len(dists); i++ {
		if dists[i] < dists[i-1] {
			t.Errorf("distances not sorted at index %d: %.6f < %.6f", i, dists[i], dists[i-1])
		}
	}
}

func TestGetNeighborsUnknown(t *testing.T) {
	_, _, err := GetNeighbors("NGC9999", 60, "all")
	if err == nil {
		t.Fatal("expected error for unknown object, got nil")
	}
	var notFound *ObjectNotFound
	if !errors.As(err, &notFound) {
		t.Errorf("expected *ObjectNotFound, got %T: %v", err, err)
	}
}

func TestGetNeighborsCatalogFilter(t *testing.T) {
	results, _, err := GetNeighbors("NGC7000", 60, "IC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, obj := range results {
		if !strings.HasPrefix(obj.Name, "IC") {
			t.Errorf("catalog filter IC: unexpected object %s", obj.Name)
		}
	}
}

func TestGetNearby(t *testing.T) {
	// Coordinates of NGC7000 (North America Nebula): RA 20:59:17.10, Dec +44:31:43.6
	results, dists, err := GetNearby("20:59:17.10", "+44:31:43.6", 60, "all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one nearby object, got none")
	}
	if len(results) != len(dists) {
		t.Fatalf("results and dists length mismatch: %d vs %d", len(results), len(dists))
	}
	for i := 1; i < len(dists); i++ {
		if dists[i] < dists[i-1] {
			t.Errorf("distances not sorted at index %d: %.6f < %.6f", i, dists[i], dists[i-1])
		}
	}
}

func TestGetNearbyInvalidRA(t *testing.T) {
	_, _, err := GetNearby("99:99:99.99", "+00:00:00.0", 60, "all")
	if err == nil {
		t.Fatal("expected error for invalid RA, got nil")
	}
	var ic *InvalidCoordinates
	if !errors.As(err, &ic) {
		t.Errorf("expected *InvalidCoordinates, got %T: %v", err, err)
	}
}

func TestGetNearbyInvalidDec(t *testing.T) {
	_, _, err := GetNearby("00:00:00.00", "++00:00:00.0", 60, "all")
	if err == nil {
		t.Fatal("expected error for invalid Dec, got nil")
	}
	var ic *InvalidCoordinates
	if !errors.As(err, &ic) {
		t.Errorf("expected *InvalidCoordinates, got %T: %v", err, err)
	}
}

func TestInRegion(t *testing.T) {
	// RA wrap-around: 350° to 10° straddles 0h
	results := InRegion(350, 10, -90, 90)
	if len(results) == 0 {
		t.Fatal("expected objects in wrap-around RA region, got none")
	}

	var hasHigh, hasLow bool
	for _, obj := range results {
		raDeg := obj.RADeg()
		if raDeg >= 350 {
			hasHigh = true
		}
		if raDeg <= 10 {
			hasLow = true
		}
	}
	if !hasHigh {
		t.Error("expected at least one object with RADeg >= 350 in wrap-around region")
	}
	if !hasLow {
		t.Error("expected at least one object with RADeg <= 10 in wrap-around region")
	}
}

func TestInRegionNormal(t *testing.T) {
	// Non-wrapping range around the Virgo cluster
	results := InRegion(185, 190, 10, 15)
	if len(results) == 0 {
		t.Fatal("expected objects in Virgo cluster region, got none")
	}
	for _, obj := range results {
		raDeg := obj.RADeg()
		decDeg := obj.DecDeg()
		if raDeg < 185 || raDeg > 190 {
			t.Errorf("object %s RA=%.4f outside [185,190]", obj.Name, raDeg)
		}
		if decDeg < 10 || decDeg > 15 {
			t.Errorf("object %s Dec=%.4f outside [10,15]", obj.Name, decDeg)
		}
	}
}

func TestSortedNeighbors(t *testing.T) {
	_, dists, _ := GetNeighbors("NGC0001", 120, "all")
	for i := 1; i < len(dists); i++ {
		if dists[i] < dists[i-1] {
			t.Errorf("distances not sorted at index %d: %.4f < %.4f", i, dists[i], dists[i-1])
		}
	}
}

func TestAngularSepSamePoint(t *testing.T) {
	sep := angularSep(0, 0, 0, 0)
	if math.Abs(sep) > 1e-10 {
		t.Errorf("separation of same point should be 0, got %e", sep)
	}
}

func TestAngularSepPoles(t *testing.T) {
	northPole := math.Pi / 2
	southPole := -math.Pi / 2
	sep := angularSep(0, northPole, 0, southPole)
	if math.Abs(sep-180.0) > 1e-9 {
		t.Errorf("separation between poles should be 180°, got %f", sep)
	}
}

// TestGetNeighborsExcludesRefByID locks in the B6 invariant: GetNeighbors
// must exclude the reference object by ID, not by pointer identity. If the
// comparison ever regresses to `obj == ref`, callers who produced the ref
// from a source that doesn't return the package's canonical pointer would
// see the reference listed as a 0° self-match.
func TestGetNeighborsExcludesRefByID(t *testing.T) {
	objs, dists, err := GetNeighbors("NGC0001", 60, "all")
	if err != nil {
		t.Fatalf("GetNeighbors: %v", err)
	}
	if len(objs) == 0 {
		t.Skip("no neighbors within 60' of NGC0001 in current catalog")
	}
	// Find any reference object we know is in the catalog.
	ref, err := Get("NGC0001")
	if err != nil {
		t.Fatalf("Get(NGC0001): %v", err)
	}
	for i, o := range objs {
		if o.ID == ref.ID {
			t.Errorf("GetNeighbors returned the ref object itself at index %d (dist=%f)", i, dists[i])
		}
	}
}

// TestSpatialRejectsUnknownCatalog locks in the B4 invariant: GetNearby and
// GetNeighbors must validate the catalog selector up front and return a typed
// *UnknownIdentifier for unknown values instead of silently widening to "all".
// GetList already enforced this; the spatial path used to fail open.
func TestSpatialRejectsUnknownCatalog(t *testing.T) {
	var ui *UnknownIdentifier

	if _, _, err := GetNearby("00:00:00.00", "+00:00:00.0", 10, "bogus"); err == nil {
		t.Error("GetNearby(bogus catalog): expected error, got nil")
	} else if !errors.As(err, &ui) {
		t.Errorf("GetNearby: got %T (%v), want *UnknownIdentifier", err, err)
	}

	if _, _, err := GetNeighbors("NGC0001", 10, "bogus"); err == nil {
		t.Error("GetNeighbors(bogus catalog): expected error, got nil")
	} else if !errors.As(err, &ui) {
		t.Errorf("GetNeighbors: got %T (%v), want *UnknownIdentifier", err, err)
	}

	// "all"/"NGC"/"IC" must still be accepted.
	for _, ok := range []string{"all", "NGC", "IC"} {
		if _, _, err := GetNeighbors("NGC0001", 10, ok); err != nil {
			t.Errorf("GetNeighbors(%q): unexpected error %v", ok, err)
		}
	}
}
