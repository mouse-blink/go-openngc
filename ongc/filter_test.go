// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

package ongc

import (
	"math"
	"strings"
	"testing"
)

func TestGetListAllGalaxies(t *testing.T) {
	results, err := GetList(WithType("G"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one galaxy")
	}
	for _, obj := range results {
		if obj.Type != TypeGalaxy {
			t.Errorf("expected TypeGalaxy, got %q for %s", obj.Type, obj.Name)
		}
	}
}

func TestGetListByConstellation(t *testing.T) {
	results, err := GetList(WithConstellation("Vir"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one object in Vir")
	}
	for _, obj := range results {
		if obj.Const != "Vir" {
			t.Errorf("expected Const==Vir, got %q for %s", obj.Const, obj.Name)
		}
	}
}

func TestGetListMagnitude(t *testing.T) {
	results, err := GetList(WithUpToVMag(6.0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one object with VMag <= 6.0")
	}
	for _, obj := range results {
		if obj.VMag == nil {
			t.Errorf("nil VMag object included: %s", obj.Name)
			continue
		}
		if *obj.VMag > 6.0 {
			t.Errorf("VMag %f > 6.0 for %s", *obj.VMag, obj.Name)
		}
	}
}

func TestGetListUnknownType(t *testing.T) {
	_, err := GetList(WithType("BOGUS"))
	if err == nil {
		t.Fatal("expected error for unknown type")
	}
	if _, ok := err.(*UnknownIdentifier); !ok {
		t.Errorf("expected *UnknownIdentifier, got %T: %v", err, err)
	}
}

func TestGetListUnknownConstellation(t *testing.T) {
	_, err := GetList(WithConstellation("Xyz"))
	if err == nil {
		t.Fatal("expected error for unknown constellation")
	}
	if _, ok := err.(*UnknownIdentifier); !ok {
		t.Errorf("expected *UnknownIdentifier, got %T: %v", err, err)
	}
}

func TestGetListUnknownCatalog(t *testing.T) {
	_, err := GetList(WithCatalog("UNKNOWN"))
	if err == nil {
		t.Fatal("expected error for unknown catalog")
	}
	if _, ok := err.(*UnknownIdentifier); !ok {
		t.Errorf("expected *UnknownIdentifier, got %T: %v", err, err)
	}
}

func TestGetListCatalogFilter(t *testing.T) {
	ngcResults, err := GetList(WithCatalog("NGC"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, obj := range ngcResults {
		if !strings.HasPrefix(obj.Name, "NGC") {
			t.Errorf("expected NGC prefix, got %q", obj.Name)
		}
	}

	icResults, err := GetList(WithCatalog("IC"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, obj := range icResults {
		if !strings.HasPrefix(obj.Name, "IC") {
			t.Errorf("expected IC prefix, got %q", obj.Name)
		}
	}
}

func TestGetListNilExclusion(t *testing.T) {
	results, err := GetList(WithUpToBMag(15.0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, obj := range results {
		if obj.BMag == nil {
			t.Errorf("nil BMag object included: %s", obj.Name)
			continue
		}
		if *obj.BMag > 15.0 {
			t.Errorf("BMag %f > 15.0 for %s", *obj.BMag, obj.Name)
		}
	}
}

// TestWithMinMaxRANormalizesNegative locks in the F3 invariant:
// negative RA degree inputs are normalized modulo 360 (Astropy convention)
// instead of silently no-opping because every catalog RA ≥ 0 satisfies
// ra >= negative-threshold. Lets callers express wrap-around boxes as
// WithMinRA(-10), WithMaxRA(10).
func TestWithMinMaxRANormalizesNegative(t *testing.T) {
	// A value ~5° below zero should wrap to ~355°. After applying the
	// option, opt.minRA in radians must equal WithMinRA(355) in radians.
	negOpt := WithMinRA(-5)
	posOpt := WithMinRA(355)

	var negOpts, posOpts listOptions
	negOpt(&negOpts)
	posOpt(&posOpts)

	if negOpts.minRA == nil || posOpts.minRA == nil {
		t.Fatal("minRA not set")
	}
	// Tolerance covers any rounding through math.Mod.
	if math.Abs(*negOpts.minRA-*posOpts.minRA) > 1e-12 {
		t.Errorf("WithMinRA(-5) rad = %.12f, want WithMinRA(355) rad = %.12f",
			*negOpts.minRA, *posOpts.minRA)
	}

	// End-to-end: a wrap-around box [-10°, +10°] must return at least as
	// many objects as [0°, 10°] since it also covers [350°, 360°).
	resWrap, err := GetList(WithMinRA(-10), WithMaxRA(10))
	if err != nil {
		t.Fatalf("GetList wrap: %v", err)
	}
	resBase, err := GetList(WithMinRA(0), WithMaxRA(10))
	if err != nil {
		t.Fatalf("GetList base: %v", err)
	}
	if len(resWrap) <= len(resBase) {
		t.Errorf("wrap-around filter len=%d should exceed base len=%d — negative RA input no-opped?",
			len(resWrap), len(resBase))
	}
}

// TestNormalizeRARad covers the helper directly so the modulo behavior
// can't be broken without a test noticing.
func TestNormalizeRARad(t *testing.T) {
	const twoPi = 2 * math.Pi
	cases := []struct {
		in, want float64
	}{
		{0, 0},
		{math.Pi, math.Pi},
		{twoPi, 0}, // exactly 2π wraps to 0
		{-math.Pi / 2, 3 * math.Pi / 2},
		{3 * twoPi, 0},  // multiple wraps
		{-3 * twoPi, 0}, // multiple negative wraps
		{-math.Pi / 6, twoPi - math.Pi/6},
	}
	for _, tc := range cases {
		got := normalizeRARad(tc.in)
		if math.Abs(got-tc.want) > 1e-12 {
			t.Errorf("normalizeRARad(%.6f) = %.6f, want %.6f", tc.in, got, tc.want)
		}
		if got < 0 || got >= twoPi {
			t.Errorf("normalizeRARad(%.6f) = %.6f outside [0, 2π)", tc.in, got)
		}
	}
}

// TestFilterExcludesNaNCoordObjectsOnCoordFilter locks in the F2 invariant:
// objects with missing catalog coordinates (RA/Dec = NaN) must not leak
// through any filter that engages a RA or Dec bound. The previous code
// relied on `obj.Dec < *opt.minDec` / `obj.Dec > *opt.maxDec`, both of
// which silently evaluate false for NaN and let those rows through.
func TestFilterExcludesNaNCoordObjectsOnCoordFilter(t *testing.T) {
	// Seven NonEx IC rows in the catalog ship with empty RA/Dec fields;
	// pick one to use as the positive assertion target.
	const nanName = "IC1064"
	nanObj, ok := byName[nanName]
	if !ok {
		t.Skipf("%s no longer in catalog; update fixture to any object with NaN coords", nanName)
	}
	if !math.IsNaN(nanObj.Dec) {
		t.Fatalf("%s.Dec is not NaN — fixture precondition broken", nanName)
	}

	// GetList with just Dec bounds: NaN-Dec rows must not appear.
	results, err := GetList(WithMinDec(-90), WithMaxDec(90))
	if err != nil {
		t.Fatalf("GetList: %v", err)
	}
	for _, obj := range results {
		if obj.Name == nanName {
			t.Errorf("NaN-Dec object %q leaked into Dec-filtered result", obj.Name)
		}
	}

	// Also confirm the combined RA+Dec filter behaves correctly.
	results, err = GetList(WithMinRA(0), WithMaxRA(360), WithMinDec(-90), WithMaxDec(90))
	if err != nil {
		t.Fatalf("GetList: %v", err)
	}
	for _, obj := range results {
		if obj.Name == nanName {
			t.Errorf("NaN-coord object %q leaked into RA+Dec filter result", obj.Name)
		}
	}

	// Same query without any coord filter must still include the NaN row
	// (F2 should only exclude NaN coords when a coord filter is engaged).
	results, err = GetList()
	if err != nil {
		t.Fatalf("GetList unfiltered: %v", err)
	}
	found := false
	for _, obj := range results {
		if obj.Name == nanName {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("unfiltered GetList should still include %q (NaN-coord row)", nanName)
	}
}
