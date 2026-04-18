// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

package ongc

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
)

func TestGet(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		wantErr  bool
		wantName string
	}{
		{
			name:     "direct NGC lookup",
			input:    "NGC0001",
			wantName: "NGC0001",
		},
		{
			name:     "Messier alias resolves to NGC0224",
			input:    "M31",
			wantName: "NGC0224",
		},
		{
			name:     "IC object lookup",
			input:    "IC0673",
			wantName: "IC0673",
		},
		{
			name:    "unknown name returns ObjectNotFound",
			input:   "NOTREAL",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			obj, err := Get(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				var notFound *ObjectNotFound
				if !errors.As(err, &notFound) {
					t.Errorf("error type: got %T, want *ObjectNotFound", err)
				}
				want := "Object named NOTREAL not found in the database."
				if err.Error() != want {
					t.Errorf("error message: got %q, want %q", err.Error(), want)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if obj.Name != tc.wantName {
				t.Errorf("Name: got %q, want %q", obj.Name, tc.wantName)
			}
		})
	}
}

func TestGetNormalization(t *testing.T) {
	obj1, err := Get("ngc1")
	if err != nil {
		t.Fatalf("Get(ngc1): %v", err)
	}
	obj2, err := Get("NGC0001")
	if err != nil {
		t.Fatalf("Get(NGC0001): %v", err)
	}
	if obj1 != obj2 {
		t.Errorf("Get(ngc1) and Get(NGC0001) returned different objects: %p vs %p", obj1, obj2)
	}

	obj3, err := Get("m31")
	if err != nil {
		t.Fatalf("Get(m31): %v", err)
	}
	if obj3.Name != "NGC0224" {
		t.Errorf("Get(m31) Name: got %q, want NGC0224", obj3.Name)
	}
}

func TestMessier(t *testing.T) {
	cases := []struct {
		n        int
		wantName string
		wantErr  bool
	}{
		{31, "NGC0224", false},
		{1, "NGC1952", false},
		{999, "", true},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(fmt.Sprintf("M%d", tc.n), func(t *testing.T) {
			obj, err := Messier(tc.n)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				var notFound *ObjectNotFound
				if !errors.As(err, &notFound) {
					t.Errorf("error type: got %T, want *ObjectNotFound", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if obj.Name != tc.wantName {
				t.Errorf("Name: got %q, want %q", obj.Name, tc.wantName)
			}
		})
	}
}

func TestGetMany(t *testing.T) {
	names := []string{"NGC0001", "NOTREAL", "M31"}
	results := GetMany(names)

	if len(results) != len(names) {
		t.Fatalf("len(results): got %d, want %d", len(results), len(names))
	}

	for i, r := range results {
		if r.Object != nil && r.Err != nil {
			t.Errorf("results[%d]: both Object and Err are non-nil", i)
		}
		if r.Object == nil && r.Err == nil {
			t.Errorf("results[%d]: both Object and Err are nil", i)
		}
	}

	if results[0].Object == nil {
		t.Error("results[0] (NGC0001): expected non-nil Object")
	}
	if results[1].Err == nil {
		t.Error("results[1] (NOTREAL): expected non-nil Err")
	}
	if results[2].Object == nil {
		t.Error("results[2] (M31): expected non-nil Object")
	}
}

func TestSearch(t *testing.T) {
	t.Run("Andromeda contains NGC0224", func(t *testing.T) {
		results := Search("Andromeda")
		found := false
		for _, obj := range results {
			if obj.Name == "NGC0224" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Search(Andromeda) did not return NGC0224")
		}
	})

	t.Run("empty query returns empty slice", func(t *testing.T) {
		results := Search("")
		if len(results) != 0 {
			t.Errorf("Search(\"\") len: got %d, want 0", len(results))
		}
	})

	// F6: whitespace-only queries are treated as empty so callers who
	// forget to trim user input don't trigger a silent full-catalog scan.
	t.Run("whitespace-only query returns empty slice", func(t *testing.T) {
		for _, q := range []string{"  ", "\t", "\n", " \t\n "} {
			if got := Search(q); len(got) != 0 {
				t.Errorf("Search(%q) len: got %d, want 0", q, len(got))
			}
		}
	})

	t.Run("leading/trailing whitespace is trimmed", func(t *testing.T) {
		// "  NGC0001  " matches the same set as "NGC0001".
		trimmed := Search("NGC0001")
		padded := Search("  NGC0001  ")
		if len(trimmed) != len(padded) {
			t.Errorf("padded query should match trimmed: got %d vs %d", len(padded), len(trimmed))
		}
	})

	t.Run("name match NGC0001", func(t *testing.T) {
		results := Search("NGC0001")
		found := false
		for _, obj := range results {
			if obj.Name == "NGC0001" {
				found = true
				break
			}
		}
		if !found {
			t.Error("Search(NGC0001) did not return NGC0001")
		}
	})

	t.Run("results in catalog order", func(t *testing.T) {
		results := Search("NGC000")
		for i := 1; i < len(results); i++ {
			if results[i].ID < results[i-1].ID {
				t.Errorf("results not in catalog order: results[%d].ID=%d < results[%d].ID=%d",
					i, results[i].ID, i-1, results[i-1].ID)
			}
		}
	})
}

type errorFixture struct {
	Op    string `json:"op"`
	Input string `json:"input"`
	Error string `json:"error"`
}

func TestErrorMessageParity(t *testing.T) {
	if _, err := os.Stat("../testdata/errors.json"); os.IsNotExist(err) {
		t.Skip("testdata/errors.json not found; run scripts/generate_testdata.py")
	}

	data, err := os.ReadFile("../testdata/errors.json")
	if err != nil {
		t.Fatalf("reading errors.json: %v", err)
	}

	var fixtures []errorFixture
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatalf("unmarshaling errors.json: %v", err)
	}

	for _, fix := range fixtures {
		fix := fix
		t.Run(fix.Op+"/"+fix.Input, func(t *testing.T) {
			switch fix.Op {
			case "get":
				_, err := Get(fix.Input)
				if err == nil {
					t.Fatalf("Get(%q): expected error, got nil", fix.Input)
				}
				if err.Error() != fix.Error {
					t.Errorf("error message:\n got  %q\n want %q", err.Error(), fix.Error)
				}
			default:
				t.Skipf("unhandled op %q", fix.Op)
			}
		})
	}
}

// TestMessierOutOfRangeReturnsObjectNotFound locks in the B12 parity invariant:
// PyOngc v1.2.0 raises ObjectNotFound (not UnknownIdentifier) for Messier
// numbers that don't resolve — including 0, negatives, and values above 110.
// Matching PyOngc's error class is part of the byte-exact parity contract.
func TestMessierOutOfRangeReturnsObjectNotFound(t *testing.T) {
	cases := []int{0, -1, 111, 999}
	for _, n := range cases {
		t.Run(fmt.Sprintf("M%d", n), func(t *testing.T) {
			_, err := Messier(n)
			if err == nil {
				t.Fatalf("Messier(%d): expected error, got nil", n)
			}
			var notFound *ObjectNotFound
			if !errors.As(err, &notFound) {
				t.Errorf("Messier(%d): got %T (%v), want *ObjectNotFound", n, err, err)
			}
		})
	}

	// Valid Messier numbers still resolve.
	if _, err := Messier(31); err != nil {
		t.Errorf("Messier(31): unexpected error %v", err)
	}
}
