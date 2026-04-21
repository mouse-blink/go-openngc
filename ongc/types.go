// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

package ongc

// ObjectType represents the classification code for a deep-sky object as used in OpenNGC CSV data.
type ObjectType string

// ObjectType constants enumerate the 21 OpenNGC classification codes.
// Use TypeName to get the human-readable form and IsKnownType to validate.
const (
	TypeStar          ObjectType = "*"
	TypeDoubleStar    ObjectType = "**"
	TypeStarAssoc     ObjectType = "*Ass"
	TypeOpenCluster   ObjectType = "OCl"
	TypeGlobCluster   ObjectType = "GCl"
	TypeClusterNeb    ObjectType = "Cl+N"
	TypeGalaxy        ObjectType = "G"
	TypeGalaxyPair    ObjectType = "GPair"
	TypeGalaxyTriplet ObjectType = "GTrpl"
	TypeGalaxyGroup   ObjectType = "GGroup"
	TypePlanetaryNeb  ObjectType = "PN"
	TypeHII           ObjectType = "HII"
	TypeDarkNeb       ObjectType = "DrkN"
	TypeEmissionNeb   ObjectType = "EmN"
	TypeNebula        ObjectType = "Neb"
	TypeReflectionNeb ObjectType = "RfN"
	TypeSNR           ObjectType = "SNR"
	TypeNova          ObjectType = "Nova"
	TypeNonExistent   ObjectType = "NonEx"
	TypeDuplicated    ObjectType = "Dup"
	TypeOther         ObjectType = "Other"
)

// typeNames maps CSV short codes to PyOngc-compatible display strings.
// Unexported so callers cannot mutate it and poison String() / PrintDetails
// output globally. Use TypeName / TypeCodes / IsKnownType for access.
var typeNames = map[ObjectType]string{
	TypeStar:          "Star",
	TypeDoubleStar:    "Double star",
	TypeStarAssoc:     "Association of stars",
	TypeOpenCluster:   "Open Cluster",
	TypeGlobCluster:   "Globular Cluster",
	TypeClusterNeb:    "Star cluster + Nebula",
	TypeGalaxy:        "Galaxy",
	TypeGalaxyPair:    "Galaxy Pair",
	TypeGalaxyTriplet: "Galaxy Triplet",
	TypeGalaxyGroup:   "Group of galaxies",
	TypePlanetaryNeb:  "Planetary Nebula",
	TypeHII:           "HII Ionized region",
	TypeDarkNeb:       "Dark Nebula",
	TypeEmissionNeb:   "Emission Nebula",
	TypeNebula:        "Nebula",
	TypeReflectionNeb: "Reflection Nebula",
	TypeSNR:           "Supernova remnant",
	TypeNova:          "Nova star",
	TypeNonExistent:   "Nonexistent object",
	TypeDuplicated:    "Duplicated record",
	TypeOther:         "Object of other/unknown type",
}

// TypeName returns the PyOngc-compatible display name for a known type code.
// The second return is false for unknown codes (e.g. a future upstream
// addition), in which case the first return is empty. Callers that want a
// graceful fallback to the raw code string should use Object.TypeLongName.
func TypeName(t ObjectType) (string, bool) {
	name, ok := typeNames[t]
	return name, ok
}

// TypeCodes returns every type code this library knows a display name for.
// Order is unspecified; sort if you need a stable iteration.
func TypeCodes() []ObjectType {
	out := make([]ObjectType, 0, len(typeNames))
	for code := range typeNames {
		out = append(out, code)
	}
	return out
}

// IsKnownType reports whether the type code is one of the OpenNGC types
// this library recognizes. Unknown codes are still carried through object
// data and display, but filters and validators use this to reject typos.
func IsKnownType(t ObjectType) bool {
	_, ok := typeNames[t]
	return ok
}

// IsGalaxy reports whether the type code is any galaxy-family classification
// (single galaxy, pair, triplet, or group).
//
// This is a general-purpose classifier, NOT the gate for the "Surface
// brightness / Hubble classification" row in PrintDetails. PyOngc v1.2.0
// emits that row only for the bare Galaxy type and suppresses it for
// pairs/triplets/groups, so PrintDetails matches `o.Type == TypeGalaxy`
// directly to preserve byte-exact parity. Confirmed against PyOngc 1.2.0
// for IC0409 (GPair, Hubble=E), NGC0247C (GPair, Sb), IC1835 (GGroup),
// and IC2308 (GTrpl) — none of which render the row.
func (t ObjectType) IsGalaxy() bool {
	switch t {
	case TypeGalaxy, TypeGalaxyPair, TypeGalaxyTriplet, TypeGalaxyGroup:
		return true
	default:
		return false
	}
}

// IsPlanetaryNebula reports whether the type code is the bare Planetary
// Nebula type. PrintDetails uses this as one of the triggers for the
// central-star section (the other is having any cstar data).
func (t ObjectType) IsPlanetaryNebula() bool {
	return t == TypePlanetaryNeb
}

// ObjectNotFound is returned when a named object is not present in the catalog.
type ObjectNotFound struct{ Name string }

// Error implements the error interface, returning a PyOngc-compatible message.
func (e *ObjectNotFound) Error() string {
	return "Object named " + e.Name + " not found in the database."
}

// InvalidCoordinates is returned when coordinate values are out of range or malformed.
type InvalidCoordinates struct{ Msg string }

// Error implements the error interface.
func (e *InvalidCoordinates) Error() string { return e.Msg }

// UnknownIdentifier is returned when an object name cannot be parsed into a known catalog prefix.
type UnknownIdentifier struct{ Msg string }

// Error implements the error interface.
func (e *UnknownIdentifier) Error() string { return e.Msg }

// IAUConstellations is the set of all 88 IAU 3-letter constellation abbreviations.
// Used by filter.go to validate WithConstellation inputs.
// Ser covers both Serpens Caput and Serpens Cauda (single abbreviation in OpenNGC data).
var IAUConstellations = map[string]struct{}{
	"And": {}, "Ant": {}, "Aps": {}, "Aql": {}, "Aqr": {}, "Ara": {}, "Ari": {},
	"Aur": {}, "Boo": {}, "CMa": {}, "CMi": {}, "CVn": {}, "Cae": {}, "Cam": {},
	"Cap": {}, "Car": {}, "Cas": {}, "Cen": {}, "Cep": {}, "Cet": {}, "Cha": {},
	"Cir": {}, "Cnc": {}, "Col": {}, "Com": {}, "CrA": {}, "CrB": {}, "Crt": {},
	"Cru": {}, "Crv": {}, "Cyg": {}, "Del": {}, "Dor": {}, "Dra": {}, "Equ": {},
	"Eri": {}, "For": {}, "Gem": {}, "Gru": {}, "Her": {}, "Hor": {}, "Hya": {},
	"Hyi": {}, "Ind": {}, "LMi": {}, "Lac": {}, "Leo": {}, "Lep": {}, "Lib": {},
	"Lup": {}, "Lyn": {}, "Lyr": {}, "Men": {}, "Mic": {}, "Mon": {}, "Mus": {},
	"Nor": {}, "Oct": {}, "Oph": {}, "Ori": {}, "Pav": {}, "Peg": {}, "Per": {},
	"Phe": {}, "Pic": {}, "PsA": {}, "Psc": {}, "Pup": {}, "Pyx": {}, "Ret": {},
	"Scl": {}, "Sco": {}, "Sct": {}, "Ser": {}, "Sex": {}, "Sge": {}, "Sgr": {},
	"Tau": {}, "Tel": {}, "TrA": {}, "Tri": {}, "Tuc": {}, "UMa": {}, "UMi": {},
	"Vel": {}, "Vir": {}, "Vol": {}, "Vul": {},
}
