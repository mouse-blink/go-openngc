// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

package ongc_test

import (
	"fmt"
	"log"

	"github.com/xen0l/go-openngc/ongc"
)

// ExampleGet demonstrates looking up a deep-sky object by any recognized name.
// The name is normalized before lookup, so "NGC1", "ngc1", and "ngc 1" all
// resolve to the same object.
func ExampleGet() {
	obj, err := ongc.Get("NGC1")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(obj)
	// Output:
	// NGC0001, Galaxy in Peg
}

// ExampleMessier demonstrates looking up an object by Messier number.
func ExampleMessier() {
	obj, err := ongc.Messier(31)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(obj)
	// Output:
	// NGC0224, Galaxy in And
}

// ExampleSearch demonstrates case-insensitive substring search across names
// and cross-identifiers. An empty query returns an empty slice.
func ExampleSearch() {
	results := ongc.Search("Crab")
	for _, obj := range results {
		fmt.Println(obj)
	}
}

// ExampleGetMany demonstrates batch lookup. Each result carries either a
// non-nil Object or a non-nil Err, never both.
func ExampleGetMany() {
	results := ongc.GetMany([]string{"NGC1", "M31"})
	for _, r := range results {
		if r.Err != nil {
			fmt.Println("error:", r.Err)
			continue
		}
		fmt.Println(r.Object)
	}
	// Output:
	// NGC0001, Galaxy in Peg
	// NGC0224, Galaxy in And
}

// ExampleGetList demonstrates filtering the catalog with functional options.
// Options compose: all supplied filters must match.
func ExampleGetList() {
	// All galaxies in Virgo brighter than V magnitude 12.
	galaxies, err := ongc.GetList(
		ongc.WithType("G"),
		ongc.WithConstellation("Vir"),
		ongc.WithUpToVMag(12),
	)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("%d galaxies matched\n", len(galaxies))
}

// ExampleGetSeparation demonstrates computing the angular distance between
// two named objects. The result is in degrees.
func ExampleGetSeparation() {
	sep, err := ongc.GetSeparation("NGC0001", "NGC0002")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("separation: %.4f°\n", sep)
}

// ExampleGetNeighbors demonstrates finding objects within a radius of a named
// object. Distances are in degrees, results sorted nearest-first.
func ExampleGetNeighbors() {
	objs, dists, err := ongc.GetNeighbors("M31", 30, "NGC")
	if err != nil {
		log.Fatal(err)
	}
	for i, obj := range objs {
		fmt.Printf("%.2f° → %s\n", dists[i], obj.Name)
	}
}

// ExampleGetNearby demonstrates finding objects near arbitrary sky coordinates.
// Coordinates are sexagesimal strings: "HH:MM:SS.ss" for RA, "±DD:MM:SS.s" for Dec.
func ExampleGetNearby() {
	// Coordinates of M31 (Andromeda Galaxy).
	objs, dists, err := ongc.GetNearby("00:42:44.33", "+41:16:09.4", 30, "NGC")
	if err != nil {
		log.Fatal(err)
	}
	for i, obj := range objs {
		fmt.Printf("%.2f° → %s\n", dists[i], obj.Name)
	}
}

// ExampleInRegion demonstrates selecting all objects inside a rectangular sky
// region bounded by RA and Dec in degrees. RA wrap-around is supported
// (ramin > ramax spans the 0h meridian).
func ExampleInRegion() {
	// 2°×2° box centered on the Virgo Cluster (RA 186–188°, Dec 12–14°).
	objs := ongc.InRegion(186, 188, 12, 14)
	fmt.Printf("%d objects in region\n", len(objs))
}

// ExampleObject_String demonstrates the one-line string representation.
func ExampleObject_String() {
	obj, err := ongc.Get("NGC0001")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(obj.String())
	// Output:
	// NGC0001, Galaxy in Peg
}

// ExampleObject_PrintDetails demonstrates the full 79-column detail box,
// equivalent to PyOngc's printDetails output.
func ExampleObject_PrintDetails() {
	obj, err := ongc.Get("NGC0001")
	if err != nil {
		log.Fatal(err)
	}
	fmt.Print(obj.PrintDetails())
}

// ExampleNormalizeName demonstrates canonical name normalization.
// NGC/IC numbers are zero-padded to 4 digits; the prefix and any suffix
// are uppercased. Messier names ("M31") pass through unchanged — alias
// lookup in Get handles the resolution.
func ExampleNormalizeName() {
	fmt.Println(ongc.NormalizeName("NGC1"))
	fmt.Println(ongc.NormalizeName("ngc 1 a"))
	fmt.Println(ongc.NormalizeName("ic 673"))
	// Output:
	// NGC0001
	// NGC0001A
	// IC0673
}
