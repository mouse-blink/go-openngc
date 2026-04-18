// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

package ongc

import "testing"

// Run with: rtk go test ./ongc/... -bench=. -benchmem -count=5
//
// These benchmarks are the regression surface for the perf work in this
// round. Key targets after G1-G3:
//   BenchmarkGetNeighbors   — Dec-band index should cut allocs/op and ns/op
//                             by ~40× compared to a brute-force baseline.
//   BenchmarkSearchIdentifier — pre-uppercased searchKeys should drop
//                             allocs/op from ~51K to ≤2 (query ToUpper +
//                             result slice growth).
//   BenchmarkInit-like — measured indirectly via BenchmarkGetByName, which
//                        dominates the startup hot path shape.
//
// The benchmarks are intentionally narrow — each exercises one public
// entry point so regressions are easy to localize.

func BenchmarkGetByName(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, _ = Get("NGC0224")
	}
}

func BenchmarkGetByAlias(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, _ = Get("M31")
	}
}

func BenchmarkGetMany(b *testing.B) {
	names := []string{"NGC0001", "M31", "IC0010", "NGC7000", "M45"}
	b.ReportAllocs()
	for b.Loop() {
		_ = GetMany(names)
	}
}

// BenchmarkSearchName exercises the name path — every object's searchKeys[0]
// is the already-uppercase Name, so no ToUpper per row should fire.
func BenchmarkSearchName(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = Search("NGC")
	}
}

// BenchmarkSearchIdentifier exercises the identifier path where the
// old code ran strings.ToUpper on ~51K mixed-case identifier strings
// per call. Post-G2 this should be near-zero allocs.
func BenchmarkSearchIdentifier(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = Search("2MASX")
	}
}

// BenchmarkGetNearbyEquator / _PolarCap / _MidLat stratify the spatial
// query cost across the declination bands the Dec-band index splits on.
// Polar caps have fewer objects per band, so expect different scan costs.
func BenchmarkGetNearbyEquator(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, _, _ = GetNearby("12:00:00.00", "+00:00:00.0", 60, "all")
	}
}

func BenchmarkGetNearbyMidLat(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, _, _ = GetNearby("05:00:00.00", "+45:00:00.0", 60, "all")
	}
}

func BenchmarkGetNearbyPolarCap(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, _, _ = GetNearby("06:00:00.00", "+85:00:00.0", 120, "all")
	}
}

func BenchmarkGetNeighborsEquator(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, _, _ = GetNeighbors("NGC7000", 60, "all")
	}
}

func BenchmarkGetNeighborsHighDec(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, _, _ = GetNeighbors("NGC0224", 60, "all")
	}
}

// BenchmarkGetList with and without filters; the unfiltered path is
// effectively a catalog copy into a slice and bounds the overhead of
// the option framework itself.
func BenchmarkGetListUnfiltered(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, _ = GetList()
	}
}

func BenchmarkGetListFilteredGalaxyConst(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, _ = GetList(WithType("G"), WithConstellation("Vir"), WithUpToBMag(13.0))
	}
}

func BenchmarkInRegion(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = InRegion(180, 185, -5, 5)
	}
}

func BenchmarkInRegionAntimeridian(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = InRegion(350, 10, -90, 90)
	}
}

func BenchmarkSeparation(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_, _ = GetSeparation("NGC0869", "NGC0884")
	}
}
