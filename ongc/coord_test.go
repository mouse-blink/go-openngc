// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

package ongc

import (
	"errors"
	"math"
	"math/rand"
	"testing"
)

func TestFormatRA(t *testing.T) {
	// NGC0001 RA computed from "00:07:15.84"
	// totalHours = 0 + 7/60 + 15.84/3600 = 0.12106666...
	// rad = totalHours * π / 12
	ngc0001RA := (0.0 + 7.0/60 + 15.84/3600) * math.Pi / 12

	// near-2π value: just under one full rotation
	nearWrap := (2*math.Pi - 1e-9)

	cases := []struct {
		name string
		rad  float64
		want string
	}{
		{"NGC0001", ngc0001RA, "00:07:15.84"},
		{"zero", 0.0, "00:00:00.00"},
		{"near-wrap", nearWrap, "00:00:00.00"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatRA(tc.rad)
			if got != tc.want {
				t.Errorf("FormatRA(%v) = %q, want %q", tc.rad, got, tc.want)
			}
			// Ensure result never starts with "24:"
			if len(got) >= 2 && got[:2] == "24" {
				t.Errorf("FormatRA(%v) produced out-of-range value %q", tc.rad, got)
			}
		})
	}
}

func TestFormatDec(t *testing.T) {
	// NGC0001 Dec: +27:42:29.1
	ngc0001Dec := (27.0 + 42.0/60 + 29.1/3600) * math.Pi / 180
	// Negative example: -00:09:01.3
	negDec := -(0.0 + 9.0/60 + 1.3/3600) * math.Pi / 180

	cases := []struct {
		name string
		rad  float64
		want string
	}{
		{"NGC0001", ngc0001Dec, "+27:42:29.1"},
		{"negative", negDec, "-00:09:01.3"},
		{"zero", 0.0, "+00:00:00.0"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := FormatDec(tc.rad)
			if got != tc.want {
				t.Errorf("FormatDec(%v) = %q, want %q", tc.rad, got, tc.want)
			}
		})
	}
}

func TestParseRA(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		got, err := ParseRA("00:07:15.84")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := (0.0 + 7.0/60 + 15.84/3600) * math.Pi / 12
		if math.Abs(got-want) > 7.3e-7 {
			t.Errorf("ParseRA(\"00:07:15.84\") = %v, want ~%v (diff %v)", got, want, math.Abs(got-want))
		}
	})

	invalids := []string{"25:00:00.00", "abc", "", "00:60:00.00", "00:00:60.00", "12:30"}
	for _, s := range invalids {
		t.Run("invalid_"+s, func(t *testing.T) {
			_, err := ParseRA(s)
			if err == nil {
				t.Fatalf("ParseRA(%q) expected error, got nil", s)
			}
			var ic *InvalidCoordinates
			if !errors.As(err, &ic) {
				t.Errorf("ParseRA(%q) error type = %T, want *InvalidCoordinates", s, err)
			}
		})
	}
}

func TestParseDec(t *testing.T) {
	t.Run("valid", func(t *testing.T) {
		got, err := ParseDec("+27:42:29.1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		want := (27.0 + 42.0/60 + 29.1/3600) * math.Pi / 180
		if math.Abs(got-want) > 4.9e-7 {
			t.Errorf("ParseDec(\"+27:42:29.1\") = %v, want ~%v (diff %v)", got, want, math.Abs(got-want))
		}
	})

	invalids := []string{"91:00:00.0", "abc", "", "00:60:00.0", "00:00:60.0", "+12:30", "12:00:00.0"}
	for _, s := range invalids {
		t.Run("invalid_"+s, func(t *testing.T) {
			_, err := ParseDec(s)
			if err == nil {
				t.Fatalf("ParseDec(%q) expected error, got nil", s)
			}
			var ic *InvalidCoordinates
			if !errors.As(err, &ic) {
				t.Errorf("ParseDec(%q) error type = %T, want *InvalidCoordinates", s, err)
			}
		})
	}
}

func TestRoundTrip(t *testing.T) {
	const raTol = 7.3e-7  // 0.01s of time in radians
	const decTol = 4.9e-7 // 0.1 arcsec in radians

	// Grid of RA values: 20 points across [0, 2π)
	for i := 0; i < 20; i++ {
		ra := float64(i) * (2 * math.Pi / 20)
		formatted := FormatRA(ra)
		parsed, err := ParseRA(formatted)
		if err != nil {
			t.Errorf("ParseRA(FormatRA(%v)) error: %v", ra, err)
			continue
		}
		diff := math.Abs(parsed - ra)
		// handle wrap-around
		if diff > math.Pi {
			diff = 2*math.Pi - diff
		}
		if diff > raTol {
			t.Errorf("RA round-trip diff too large: input=%v formatted=%q parsed=%v diff=%v", ra, formatted, parsed, diff)
		}
	}

	// Grid of Dec values: 20 points across [-π/2, π/2]
	for i := 0; i < 20; i++ {
		dec := -math.Pi/2 + float64(i)*(math.Pi/19)
		formatted := FormatDec(dec)
		parsed, err := ParseDec(formatted)
		if err != nil {
			t.Errorf("ParseDec(FormatDec(%v)) error: %v", dec, err)
			continue
		}
		if math.Abs(parsed-dec) > decTol {
			t.Errorf("Dec round-trip diff too large: input=%v formatted=%q parsed=%v diff=%v", dec, formatted, parsed, math.Abs(parsed-dec))
		}
	}
}

func TestRoundTripFuzz(t *testing.T) {
	const raTol = 7.3e-7
	const decTol = 4.9e-7
	const n = 1000

	// Fixed seed for reproducibility.
	rng := rand.New(rand.NewSource(42)) //nolint:gosec

	for i := 0; i < n; i++ {
		ra := rng.Float64() * 2 * math.Pi
		formatted := FormatRA(ra)
		parsed, err := ParseRA(formatted)
		if err != nil {
			t.Errorf("RA fuzz[%d]: ParseRA(FormatRA(%v)) error: %v", i, ra, err)
			continue
		}
		diff := math.Abs(parsed - ra)
		if diff > math.Pi {
			diff = 2*math.Pi - diff
		}
		if diff > raTol {
			t.Errorf("RA fuzz[%d]: round-trip diff too large: input=%v formatted=%q parsed=%v diff=%v", i, ra, formatted, parsed, diff)
		}
	}

	for i := 0; i < n; i++ {
		dec := (rng.Float64() - 0.5) * math.Pi
		formatted := FormatDec(dec)
		parsed, err := ParseDec(formatted)
		if err != nil {
			t.Errorf("Dec fuzz[%d]: ParseDec(FormatDec(%v)) error: %v", i, dec, err)
			continue
		}
		if math.Abs(parsed-dec) > decTol {
			t.Errorf("Dec fuzz[%d]: round-trip diff too large: input=%v formatted=%q parsed=%v diff=%v", i, dec, formatted, parsed, math.Abs(parsed-dec))
		}
	}
}
