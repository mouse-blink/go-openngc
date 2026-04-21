// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

// Package ongc provides lookup, search, filtering, and spatial queries for the
// OpenNGC deep-sky object catalog (~14K NGC/IC/Messier objects from professional
// catalogs of galaxies, nebulae, and clusters). The entire catalog is compiled
// into your binary at build time—zero network calls, no database dependency,
// and no file I/O at runtime.
//
// # Quick start
//
//	obj, err := ongc.Get("NGC1")    // "NGC1", "ngc1", "ngc 1" all resolve
//	obj, err := ongc.Messier(31)    // Andromeda Galaxy
//	results  := ongc.Search("Crab") // substring across names and identifiers
//
// # Name normalization
//
// All lookup functions normalize names before matching: input is uppercased,
// NGC/IC numbers are zero-padded to 4 digits, and trailing component letters
// are canonicalized. For example, "NGC1" and "ngc 1" both resolve to "NGC0001".
// Messier numbers ("M31") are resolved via an alias table. Use [NormalizeName]
// to see the canonical form of any input.
//
// # Lookup
//
// [Get] resolves a recognized name to an [Object] by canonical NGC, IC, or
// Messier identifier. Common names and cross-catalog IDs (e.g., PGC) may map
// to multiple objects — for instance, "Antennae Galaxies" refers to two
// different catalog rows. When a name is ambiguous, [Get] returns the first
// match in catalog order; call [Search] to retrieve all matches. [GetMany]
// performs batch lookups.
//
// # Filtering
//
// [GetList] returns a slice of [*Object] pointers into the shared catalog
// (see "Returned pointers" below). Combine filters using functional options:
// [WithType], [WithConstellation], [WithUpToVMag], and coordinate ranges
// ([WithMinRA], [WithMaxRA], [WithMinDec], [WithMaxDec]).
//
// # Spatial queries
//
// [GetNeighbors] finds objects within an angular radius of a named object.
// [GetNearby] does the same from arbitrary sky coordinates (given as sexagesimal
// RA/Dec strings). [InRegion] returns all objects in a rectangular RA/Dec box.
// [GetSeparation] computes the angular distance between two objects.
//
// # Output format
//
// [Object.String] returns a short one-line description:
//
//	"NGC0001, Galaxy in Peg"
//
// [Object.PrintDetails] returns a 79-character wide detail box, byte-compatible
// with PyOngc v1.2.0 output.
//
// # Returned pointers
//
// All [*Object] pointers reference the shared catalog slice. Treat them as
// read-only: modifying a field corrupts every subsequent lookup for that object.
// Copy the struct if you need mutable access.
package ongc
