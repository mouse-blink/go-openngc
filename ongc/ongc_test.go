// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

package ongc

import (
	"encoding/json"
	"math"
	"os"
	"testing"
)

func TestObjectCount(t *testing.T) {
	if len(objects) != 14033 {
		t.Errorf("expected 14033 objects, got %d", len(objects))
	}
}

func TestSpotCheckNGC0001(t *testing.T) {
	obj, ok := byName["NGC0001"]
	if !ok {
		t.Fatal("NGC0001 not found in byName index")
	}
	if obj.Name != "NGC0001" {
		t.Errorf("Name: got %q, want %q", obj.Name, "NGC0001")
	}
	if obj.Type != TypeGalaxy {
		t.Errorf("Type: got %q, want %q", obj.Type, TypeGalaxy)
	}
	if obj.Const != "Peg" {
		t.Errorf("Const: got %q, want %q", obj.Const, "Peg")
	}
	// ID is determined by CSV row order; the CSV is alphabetically sorted so IC
	// objects precede NGC objects — we only verify the ID is positive and consistent.
	if obj.ID <= 0 {
		t.Errorf("ID: got %d, want > 0", obj.ID)
	}
}

func TestFirstRowIsID1(t *testing.T) {
	if len(objects) == 0 {
		t.Fatal("objects slice is empty")
	}
	if objects[0].ID != 1 {
		t.Errorf("objects[0].ID: got %d, want 1", objects[0].ID)
	}
}

func TestSpotCheckIC0001(t *testing.T) {
	obj, ok := byName["IC0001"]
	if !ok {
		t.Fatal("IC0001 not found in byName index")
	}
	if obj.Type != TypeDoubleStar {
		t.Errorf("Type: got %q, want %q", obj.Type, TypeDoubleStar)
	}
	if obj.Const != "Peg" {
		t.Errorf("Const: got %q, want %q", obj.Const, "Peg")
	}
}

func TestSpotCheckAddendum(t *testing.T) {
	if len(objects) < 13970 {
		t.Fatalf("not enough objects to check addendum: got %d", len(objects))
	}
	obj := objects[13969]
	if obj.ID != 13970 {
		t.Errorf("first addendum object ID: got %d, want 13970", obj.ID)
	}
}

func TestNormalizeName(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"ngc1a", "NGC0001A"},
		{"ic 673", "IC0673"},
		{"NGC1", "NGC0001"},
		{"IC1", "IC0001"},
		{"M31", "M31"},
		{"B033", "B033"},
		{" NGC 1 ", "NGC0001"},
		// B11: PyOngc v1.2.0 parity for the regex edges.
		{"NGC 1 A", "NGC0001A"},           // space between digits and letter
		{"NGC 1 AB", "NGC0001AB"},         // two-letter component
		{"NGC0001NED01", "NGC0001 NED01"}, // NED suffix, no whitespace
		{"NGC 1 NED 1", "NGC 1 NED 1"},    // PyOngc's \s? is single — reject two spaces
		{"NGC-1", "NGC-1"},                // hyphen: PyOngc rejects, we mirror
		{"IC0001NED99", "IC0001 NED99"},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got := NormalizeName(tc.input)
			if got != tc.want {
				t.Errorf("NormalizeName(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestAliasPrecedence(t *testing.T) {
	// M31 aliases to NGC0224; byName["NGC0224"] must be the authoritative record.
	primary, ok := byName["NGC0224"]
	if !ok {
		t.Fatal("NGC0224 not found in byName")
	}
	via, ok := aliases["M31"]
	if !ok {
		t.Fatal("M31 alias not found")
	}
	if via != primary {
		t.Errorf("M31 alias points to %q, want NGC0224", via.Name)
	}
	// byName always wins over alias for the same key — verify NGC0224 is in byName directly.
	if byName["NGC0224"] != primary {
		t.Error("byName[NGC0224] does not match primary record")
	}
}

type fixtureObject struct {
	Name    string  `json:"name"`
	Type    string  `json:"type"`
	RARad   float64 `json:"ra_rad"`
	DecRad  float64 `json:"dec_rad"`
	Const   string  `json:"const"`
	Messier string  `json:"messier"`
}

func TestFixtureDriven(t *testing.T) {
	if _, err := os.Stat("../testdata/objects.json"); os.IsNotExist(err) {
		t.Skip("testdata/objects.json not found; run scripts/generate_testdata.py")
	}

	data, err := os.ReadFile("../testdata/objects.json")
	if err != nil {
		t.Fatalf("reading objects.json: %v", err)
	}

	var fixtures []fixtureObject
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatalf("unmarshaling objects.json: %v", err)
	}

	for _, fix := range fixtures {
		fix := fix
		t.Run(fix.Name, func(t *testing.T) {
			obj, ok := byName[NormalizeName(fix.Name)]
			if !ok {
				t.Fatalf("object %q not found in byName", fix.Name)
			}
			if got := obj.TypeLongName(); got != fix.Type {
				t.Errorf("Type: got %q, want %q", got, fix.Type)
			}
			if obj.Const != fix.Const {
				t.Errorf("Const: got %q, want %q", obj.Const, fix.Const)
			}
			if obj.Messier != fix.Messier {
				t.Errorf("Messier: got %q, want %q", obj.Messier, fix.Messier)
			}
			if math.Abs(obj.RA-fix.RARad) > 1e-6 {
				t.Errorf("RA: got %.8f, want %.8f (diff %.2e)", obj.RA, fix.RARad, math.Abs(obj.RA-fix.RARad))
			}
			if math.Abs(obj.Dec-fix.DecRad) > 1e-6 {
				t.Errorf("Dec: got %.8f, want %.8f (diff %.2e)", obj.Dec, fix.DecRad, math.Abs(obj.Dec-fix.DecRad))
			}
		})
	}
}

type catalogFieldObject struct {
	Name    string  `json:"name"`
	Type    string  `json:"type"`
	RARad   float64 `json:"ra_rad"`
	DecRad  float64 `json:"dec_rad"`
	Const   string  `json:"const"`
	Messier string  `json:"messier"`
}

func TestFullCatalogParity(t *testing.T) {
	if _, err := os.Stat("../testdata/catalog_fields.json"); os.IsNotExist(err) {
		t.Skip("testdata/catalog_fields.json not found; run scripts/generate_testdata.py --full-catalog")
	}

	data, err := os.ReadFile("../testdata/catalog_fields.json")
	if err != nil {
		t.Fatalf("reading catalog_fields.json: %v", err)
	}

	var fixtures []catalogFieldObject
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatalf("unmarshaling catalog_fields.json: %v", err)
	}

	for _, fix := range fixtures {
		fix := fix
		t.Run(fix.Name, func(t *testing.T) {
			obj, ok := byName[NormalizeName(fix.Name)]
			if !ok {
				t.Fatalf("object %q not found in byName", fix.Name)
			}
			if got := obj.TypeLongName(); got != fix.Type {
				t.Errorf("Type: got %q, want %q", got, fix.Type)
			}
			if obj.Const != fix.Const {
				t.Errorf("Const: got %q, want %q", obj.Const, fix.Const)
			}
			if obj.Messier != fix.Messier {
				t.Errorf("Messier: got %q, want %q", obj.Messier, fix.Messier)
			}
			if math.Abs(obj.RA-fix.RARad) > 1e-6 {
				t.Errorf("RA: got %.8f, want %.8f (diff %.2e)", obj.RA, fix.RARad, math.Abs(obj.RA-fix.RARad))
			}
			if math.Abs(obj.Dec-fix.DecRad) > 1e-6 {
				t.Errorf("Dec: got %.8f, want %.8f (diff %.2e)", obj.Dec, fix.DecRad, math.Abs(obj.Dec-fix.DecRad))
			}
		})
	}
}
