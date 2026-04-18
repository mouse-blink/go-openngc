// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

package ongc

import (
	"math"
	"strings"
)

// ListOption is a functional option for GetList.
type ListOption func(*listOptions)

// listOption is an alias kept for internal readability.
type listOption = ListOption

type listOptions struct {
	catalog        string
	types          []ObjectType
	constellations []string
	minSize        *float64
	maxSize        *float64
	upToBMag       *float64
	upToVMag       *float64
	minRA          *float64
	maxRA          *float64
	minDec         *float64
	maxDec         *float64
	namedSubstr    string
	withName       bool
}

func WithCatalog(c string) ListOption {
	return func(o *listOptions) {
		o.catalog = c
	}
}

func WithType(codes ...string) ListOption {
	return func(o *listOptions) {
		for _, c := range codes {
			o.types = append(o.types, ObjectType(c))
		}
	}
}

func WithConstellation(codes ...string) ListOption {
	return func(o *listOptions) {
		o.constellations = append(o.constellations, codes...)
	}
}

func WithMinSize(arcmin float64) ListOption {
	return func(o *listOptions) {
		o.minSize = &arcmin
	}
}

func WithMaxSize(arcmin float64) ListOption {
	return func(o *listOptions) {
		o.maxSize = &arcmin
	}
}

func WithUpToBMag(mag float64) ListOption {
	return func(o *listOptions) {
		o.upToBMag = &mag
	}
}

func WithUpToVMag(mag float64) ListOption {
	return func(o *listOptions) {
		o.upToVMag = &mag
	}
}

// WithMinRA sets the lower RA bound in degrees. Values outside [0, 360)
// are normalized modulo 360 before conversion to radians — this matches
// Astropy's default Angle wrap_at='360 deg' behavior and lets callers
// express an equator-straddling box as e.g. WithMinRA(-10), WithMaxRA(10).
func WithMinRA(deg float64) ListOption {
	rad := normalizeRARad(deg * math.Pi / 180)
	return func(o *listOptions) {
		o.minRA = &rad
	}
}

// WithMaxRA sets the upper RA bound. See WithMinRA for range semantics.
func WithMaxRA(deg float64) ListOption {
	rad := normalizeRARad(deg * math.Pi / 180)
	return func(o *listOptions) {
		o.maxRA = &rad
	}
}

// normalizeRARad wraps a radian RA into [0, 2π). Catalog RA values are
// always in that range; filter bounds outside it would otherwise silently
// misbehave (negative bounds would no-op, values > 2π would never match).
func normalizeRARad(rad float64) float64 {
	const twoPi = 2 * math.Pi
	rad = math.Mod(rad, twoPi)
	if rad < 0 {
		rad += twoPi
	}
	return rad
}

func WithMinDec(deg float64) ListOption {
	rad := deg * math.Pi / 180
	return func(o *listOptions) {
		o.minDec = &rad
	}
}

func WithMaxDec(deg float64) ListOption {
	rad := deg * math.Pi / 180
	return func(o *listOptions) {
		o.maxDec = &rad
	}
}

func WithNamed(substr string) ListOption {
	return func(o *listOptions) {
		o.namedSubstr = substr
	}
}

func WithName() ListOption {
	return func(o *listOptions) {
		o.withName = true
	}
}

// GetList returns objects matching all supplied options.
// Returns *UnknownIdentifier if any type code, constellation code, or catalog value is invalid.
func GetList(opts ...listOption) ([]*Object, error) {
	opt := listOptions{catalog: "all"}
	for _, fn := range opts {
		fn(&opt)
	}

	if opt.catalog != "all" && opt.catalog != "NGC" && opt.catalog != "IC" {
		return nil, &UnknownIdentifier{Msg: "unknown catalog: " + opt.catalog}
	}

	for _, code := range opt.types {
		if !IsKnownType(code) {
			return nil, &UnknownIdentifier{Msg: "unknown type: " + string(code)}
		}
	}

	for _, code := range opt.constellations {
		if _, ok := IAUConstellations[code]; !ok {
			return nil, &UnknownIdentifier{Msg: "unknown constellation: " + code}
		}
	}

	typeSet := make(map[ObjectType]struct{}, len(opt.types))
	for _, t := range opt.types {
		typeSet[t] = struct{}{}
	}

	constSet := make(map[string]struct{}, len(opt.constellations))
	for _, c := range opt.constellations {
		constSet[c] = struct{}{}
	}

	namedLower := strings.ToLower(opt.namedSubstr)

	var results []*Object
	for _, obj := range objects {
		if !matchCatalog(obj, opt.catalog) {
			continue
		}
		if len(typeSet) > 0 {
			if _, ok := typeSet[obj.Type]; !ok {
				continue
			}
		}
		if len(constSet) > 0 {
			if _, ok := constSet[obj.Const]; !ok {
				continue
			}
		}
		if opt.minSize != nil || opt.maxSize != nil {
			if obj.MajAx == nil {
				continue
			}
			if opt.minSize != nil && *obj.MajAx < *opt.minSize {
				continue
			}
			if opt.maxSize != nil && *obj.MajAx > *opt.maxSize {
				continue
			}
		}
		if opt.upToBMag != nil {
			if obj.BMag == nil || *obj.BMag > *opt.upToBMag {
				continue
			}
		}
		if opt.upToVMag != nil {
			if obj.VMag == nil || *obj.VMag > *opt.upToVMag {
				continue
			}
		}
		// Objects with missing catalog coordinates (RA/Dec = NaN) must be
		// excluded from any coord-bound filter — IEEE 754 NaN comparisons
		// return false in both directions, so `NaN < x` AND `NaN > x` are
		// false, which would otherwise let a NaN row pass a Dec band silently.
		// matchRARange below already handles NaN correctly via its && / ||
		// logic, so only the Dec side needs an explicit guard. Applying the
		// coord guard only when a coord filter is engaged keeps the hot
		// path unaffected for callers that don't filter by sky position.
		if opt.minRA != nil || opt.maxRA != nil || opt.minDec != nil || opt.maxDec != nil {
			if !hasCoords(obj) {
				continue
			}
		}
		if opt.minRA != nil || opt.maxRA != nil {
			if !matchRARange(obj.RA, opt.minRA, opt.maxRA) {
				continue
			}
		}
		if opt.minDec != nil && obj.Dec < *opt.minDec {
			continue
		}
		if opt.maxDec != nil && obj.Dec > *opt.maxDec {
			continue
		}
		if opt.withName && len(obj.CommonNames) == 0 {
			continue
		}
		if opt.namedSubstr != "" && !matchNamed(obj, namedLower) {
			continue
		}
		results = append(results, obj)
	}
	return results, nil
}

func matchCatalog(obj *Object, catalog string) bool {
	switch catalog {
	case "NGC":
		return strings.HasPrefix(obj.Name, "NGC")
	case "IC":
		return strings.HasPrefix(obj.Name, "IC")
	default:
		return true
	}
}

func matchRARange(ra float64, minRA, maxRA *float64) bool {
	if minRA == nil && maxRA == nil {
		return true
	}
	if minRA == nil {
		return ra <= *maxRA
	}
	if maxRA == nil {
		return ra >= *minRA
	}
	if *minRA <= *maxRA {
		return ra >= *minRA && ra <= *maxRA
	}
	// wraps through 0h
	return ra >= *minRA || ra <= *maxRA
}

func matchNamed(obj *Object, lowerSubstr string) bool {
	for _, n := range obj.CommonNames {
		if strings.Contains(strings.ToLower(n), lowerSubstr) {
			return true
		}
	}
	for _, id := range obj.Identifiers {
		if strings.Contains(strings.ToLower(id), lowerSubstr) {
			return true
		}
	}
	return false
}
