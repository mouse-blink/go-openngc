// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

package ongc

import (
	"fmt"
	"math"
	"sort"
)

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

	radiusDeg := radiusArcmin / 60.0
	var neighbors []neighbor

	for _, obj := range objects {
		// Compare by ID so the identity check holds even if a caller passes in
		// a copy of the reference object (defensive — the catalog slice stores
		// one canonical pointer per ID today).
		if obj.ID == ref.ID {
			continue
		}
		if !hasCoords(obj) {
			continue
		}
		if !matchCatalog(obj, catalog) {
			continue
		}
		dist := angularSep(ref.RA, ref.Dec, obj.RA, obj.Dec)
		if dist <= radiusDeg {
			neighbors = append(neighbors, neighbor{obj: obj, dist: dist})
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

	radiusDeg := radiusArcmin / 60.0
	var neighbors []neighbor

	for _, obj := range objects {
		if !hasCoords(obj) {
			continue
		}
		if !matchCatalog(obj, catalog) {
			continue
		}
		dist := angularSep(ra, dec, obj.RA, obj.Dec)
		if dist <= radiusDeg {
			neighbors = append(neighbors, neighbor{obj: obj, dist: dist})
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
