// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

package ongc

import (
	"fmt"
	"math"
	"sort"
)

// decBandCount partitions the full declination range [-π/2, +π/2] into
// fixed 10° bands. Each object is assigned to exactly one band at init
// time (see ongc.go's init) and spatial queries scan only the bands
// their search radius can reach — roughly 2-3 bands out of 18 for a
// typical 60-arcmin cone, versus a full-catalog brute-force scan.
const decBandCount = 18

// decBandWidthRad is the angular width of a single band (10°).
const decBandWidthRad = math.Pi / 18

// decBands[k] is the slice of coord-bearing objects whose Dec falls in
// band k. Populated in init; never mutated at runtime.
var decBands [decBandCount][]*Object

// decBandIndex maps a Dec in radians to a band index in [0, decBandCount).
// Objects at exactly +π/2 (north pole) map to the last band, not beyond.
func decBandIndex(decRad float64) int {
	// Shift to [0, π] then divide.
	idx := int(math.Floor((decRad + math.Pi/2) / decBandWidthRad))
	if idx < 0 {
		return 0
	}
	if idx >= decBandCount {
		return decBandCount - 1
	}
	return idx
}

// decBandRange returns the inclusive [lo, hi] range of band indices that
// could possibly contain an object within radiusRad of decRad, clamped
// to the valid range. Used to prune the scan set.
func decBandRange(decRad, radiusRad float64) (int, int) {
	lo := decBandIndex(decRad - radiusRad)
	hi := decBandIndex(decRad + radiusRad)
	return lo, hi
}

// addToDecBand appends obj to the appropriate band; skips NaN-coord rows.
// Called from ongc.go's init loop once per object.
func addToDecBand(obj *Object) {
	if !hasCoords(obj) {
		return
	}
	decBands[decBandIndex(obj.Dec)] = append(decBands[decBandIndex(obj.Dec)], obj)
}

// angularSep returns the angular separation in degrees between two points
// expressed in radians using the Vincenty formula (numerically stable).
func angularSep(ra1, dec1, ra2, dec2 float64) float64 {
	dRA := ra2 - ra1
	num := math.Sqrt(
		math.Pow(math.Cos(dec2)*math.Sin(dRA), 2) +
			math.Pow(math.Cos(dec1)*math.Sin(dec2)-math.Sin(dec1)*math.Cos(dec2)*math.Cos(dRA), 2),
	)
	den := math.Sin(dec1)*math.Sin(dec2) + math.Cos(dec1)*math.Cos(dec2)*math.Cos(dRA)
	return math.Atan2(num, den) * 180 / math.Pi
}

// validateCatalog rejects any catalog selector not in the known set.
// Returns nil for "all", "NGC", or "IC"; *UnknownIdentifier otherwise.
// Mirrors GetList's up-front validation so spatial-API library callers can't
// silently fall through to "all" on a typo.
func validateCatalog(catalog string) error {
	switch catalog {
	case "all", "NGC", "IC":
		return nil
	default:
		return &UnknownIdentifier{Msg: "unknown catalog: " + catalog}
	}
}

// hasCoords reports whether obj has parseable RA and Dec values.
// Rows with missing or malformed catalog coords are stored as NaN.
func hasCoords(obj *Object) bool {
	return !math.IsNaN(obj.RA) && !math.IsNaN(obj.Dec)
}

// GetSeparation returns the angular separation in degrees between two named objects.
func GetSeparation(name1, name2 string) (float64, error) {
	obj1, err := Get(name1)
	if err != nil {
		return 0, err
	}
	obj2, err := Get(name2)
	if err != nil {
		return 0, err
	}
	if !hasCoords(obj1) {
		return 0, &InvalidCoordinates{Msg: fmt.Sprintf("object %q has no coordinates", obj1.Name)}
	}
	if !hasCoords(obj2) {
		return 0, &InvalidCoordinates{Msg: fmt.Sprintf("object %q has no coordinates", obj2.Name)}
	}
	return angularSep(obj1.RA, obj1.Dec, obj2.RA, obj2.Dec), nil
}

// neighbor pairs an Object with its angular distance in degrees from a reference point.
type neighbor struct {
	obj  *Object
	dist float64
}

// GetNeighbors returns objects within radiusArcmin of the named object,
// sorted nearest-first, excluding the reference object itself.
// catalog filters by "all", "NGC", or "IC". Distances are in degrees.
func GetNeighbors(name string, radiusArcmin float64, catalog string) ([]*Object, []float64, error) {
	if err := validateCatalog(catalog); err != nil {
		return nil, nil, err
	}
	ref, err := Get(name)
	if err != nil {
		return nil, nil, err
	}
	if !hasCoords(ref) {
		return nil, nil, &InvalidCoordinates{Msg: fmt.Sprintf("object %q has no coordinates", ref.Name)}
	}

	return coneSearch(ref.RA, ref.Dec, radiusArcmin, catalog, ref.ID)
}

// GetNearby is like GetNeighbors but takes sexagesimal coordinate strings
// ("HH:MM:SS.ss" for RA, "±DD:MM:SS.s" for Dec) instead of a named object.
func GetNearby(raStr, decStr string, radiusArcmin float64, catalog string) ([]*Object, []float64, error) {
	if err := validateCatalog(catalog); err != nil {
		return nil, nil, err
	}
	ra, err := ParseRA(raStr)
	if err != nil {
		return nil, nil, err
	}
	dec, err := ParseDec(decStr)
	if err != nil {
		return nil, nil, err
	}
	return coneSearch(ra, dec, radiusArcmin, catalog, 0)
}

// coneSearch is the shared worker for GetNeighbors and GetNearby. It
// uses the decBands index to prune the candidate set to just the bands
// the radius can reach — for a typical 60-arcmin query, that is 2-3
// bands (~1-2K objects) instead of the full 14K. excludeID, when
// non-zero, skips the reference object from the result (used by
// GetNeighbors). Catalog must already be validated by the caller.
func coneSearch(raRad, decRad, radiusArcmin float64, catalog string, excludeID int) ([]*Object, []float64, error) {
	radiusDeg := radiusArcmin / 60.0
	radiusRad := radiusDeg * math.Pi / 180
	loBand, hiBand := decBandRange(decRad, radiusRad)

	// Typical equatorial-field neighbor counts stay under 32; pre-sizing
	// eliminates the first 3-5 slice reallocations for the common case.
	neighbors := make([]neighbor, 0, 32)

	for band := loBand; band <= hiBand; band++ {
		for _, obj := range decBands[band] {
			if excludeID != 0 && obj.ID == excludeID {
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
	}

	sort.Slice(neighbors, func(i, j int) bool {
		return neighbors[i].dist < neighbors[j].dist
	})

	objs := make([]*Object, len(neighbors))
	dists := make([]float64, len(neighbors))
	for i, n := range neighbors {
		objs[i] = n.obj
		dists[i] = n.dist
	}
	return objs, dists, nil
}

// InRegion returns objects whose RA and Dec fall within the given bounds (in degrees).
// Bounds are inclusive. Supports RA wrap-around when ramin > ramax (e.g. 350° to 10°).
func InRegion(ramin, ramax, decmin, decmax float64) []*Object {
	const toRad = math.Pi / 180

	raminRad := ramin * toRad
	ramaxRad := ramax * toRad
	decminRad := decmin * toRad
	decmaxRad := decmax * toRad

	var result []*Object
	for _, obj := range objects {
		if !hasCoords(obj) {
			continue
		}
		if obj.Dec < decminRad || obj.Dec > decmaxRad {
			continue
		}
		var inRA bool
		if raminRad <= ramaxRad {
			inRA = obj.RA >= raminRad && obj.RA <= ramaxRad
		} else {
			// wrap-around: range straddles 0h
			inRA = obj.RA >= raminRad || obj.RA <= ramaxRad
		}
		if inRA {
			result = append(result, obj)
		}
	}
	return result
}
