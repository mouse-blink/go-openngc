// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

package ongc

import "testing"

func TestTypeNames(t *testing.T) {
	types := []ObjectType{
		TypeStar,
		TypeDoubleStar,
		TypeStarAssoc,
		TypeOpenCluster,
		TypeGlobCluster,
		TypeClusterNeb,
		TypeGalaxy,
		TypeGalaxyPair,
		TypeGalaxyTriplet,
		TypeGalaxyGroup,
		TypePlanetaryNeb,
		TypeHII,
		TypeDarkNeb,
		TypeEmissionNeb,
		TypeNebula,
		TypeReflectionNeb,
		TypeSNR,
		TypeNova,
		TypeNonExistent,
		TypeDuplicated,
		TypeOther,
	}

	if got := len(TypeCodes()); got != len(types) {
		t.Errorf("TypeCodes() has %d entries, want %d", got, len(types))
	}

	for _, ot := range types {
		name, ok := TypeName(ot)
		if !ok {
			t.Errorf("TypeName(%q): ok=false, want true", ot)
			continue
		}
		if name == "" {
			t.Errorf("TypeName(%q) returned empty string", ot)
		}
	}

	// B7: TypeName must return empty + false for an unknown code (future
	// upstream type). Ensures the accessor signature stays trustworthy.
	if name, ok := TypeName(ObjectType("QSO")); ok || name != "" {
		t.Errorf(`TypeName("QSO") = (%q, %v), want ("", false)`, name, ok)
	}

	// IsKnownType mirrors the ok return.
	if IsKnownType(ObjectType("QSO")) {
		t.Error(`IsKnownType("QSO") = true, want false`)
	}
	if !IsKnownType(TypeGalaxy) {
		t.Error("IsKnownType(TypeGalaxy) = false, want true")
	}
}

func TestIsGalaxy(t *testing.T) {
	cases := []struct {
		ot   ObjectType
		want bool
	}{
		{TypeGalaxy, true},
		{TypeGalaxyPair, true},
		{TypeGalaxyTriplet, true},
		{TypeGalaxyGroup, true},
		{TypeStar, false},
		{TypeDoubleStar, false},
		{TypeStarAssoc, false},
		{TypeOpenCluster, false},
		{TypeGlobCluster, false},
		{TypeClusterNeb, false},
		{TypePlanetaryNeb, false},
		{TypeHII, false},
		{TypeDarkNeb, false},
		{TypeEmissionNeb, false},
		{TypeNebula, false},
		{TypeReflectionNeb, false},
		{TypeSNR, false},
		{TypeNova, false},
		{TypeNonExistent, false},
		{TypeDuplicated, false},
		{TypeOther, false},
	}

	for _, tc := range cases {
		t.Run(string(tc.ot), func(t *testing.T) {
			if got := tc.ot.IsGalaxy(); got != tc.want {
				t.Errorf("IsGalaxy() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestIsPlanetaryNebula(t *testing.T) {
	cases := []struct {
		ot   ObjectType
		want bool
	}{
		{TypePlanetaryNeb, true},
		{TypeStar, false},
		{TypeDoubleStar, false},
		{TypeStarAssoc, false},
		{TypeOpenCluster, false},
		{TypeGlobCluster, false},
		{TypeClusterNeb, false},
		{TypeGalaxy, false},
		{TypeGalaxyPair, false},
		{TypeGalaxyTriplet, false},
		{TypeGalaxyGroup, false},
		{TypeHII, false},
		{TypeDarkNeb, false},
		{TypeEmissionNeb, false},
		{TypeNebula, false},
		{TypeReflectionNeb, false},
		{TypeSNR, false},
		{TypeNova, false},
		{TypeNonExistent, false},
		{TypeDuplicated, false},
		{TypeOther, false},
	}

	for _, tc := range cases {
		t.Run(string(tc.ot), func(t *testing.T) {
			if got := tc.ot.IsPlanetaryNebula(); got != tc.want {
				t.Errorf("IsPlanetaryNebula() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestObjectNotFoundError(t *testing.T) {
	err := &ObjectNotFound{Name: "NGC9999"}
	want := "Object named NGC9999 not found in the database."
	if got := err.Error(); got != want {
		t.Errorf("Error() = %q, want %q", got, want)
	}
}

func TestIAUConstellations(t *testing.T) {
	const wantLen = 88
	if got := len(IAUConstellations); got != wantLen {
		t.Errorf("len(IAUConstellations) = %d, want %d", got, wantLen)
	}
}
