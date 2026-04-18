// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

package ongc

import (
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ParseRA parses "HH:MM:SS.ss" → float64 radians.
// Returns *InvalidCoordinates if the format is invalid or values are out of range.
func ParseRA(s string) (float64, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return 0, &InvalidCoordinates{Msg: fmt.Sprintf("invalid RA format: %q", s)}
	}

	h, err := strconv.Atoi(parts[0])
	if err != nil || h < 0 || h > 23 {
		return 0, &InvalidCoordinates{Msg: fmt.Sprintf("invalid RA hours in %q", s)}
	}

	m, err := strconv.Atoi(parts[1])
	if err != nil || m < 0 || m > 59 {
		return 0, &InvalidCoordinates{Msg: fmt.Sprintf("invalid RA minutes in %q", s)}
	}

	sec, err := strconv.ParseFloat(parts[2], 64)
	if err != nil || sec < 0 || sec >= 60 {
		return 0, &InvalidCoordinates{Msg: fmt.Sprintf("invalid RA seconds in %q", s)}
	}

	totalHours := float64(h) + float64(m)/60 + sec/3600
	return totalHours * math.Pi / 12, nil
}

// ParseDec parses "±DD:MM:SS.s" → float64 radians.
// Sign character (+ or -) is required.
// Returns *InvalidCoordinates if the format is invalid or values are out of range.
func ParseDec(s string) (float64, error) {
	if len(s) < 2 {
		return 0, &InvalidCoordinates{Msg: fmt.Sprintf("invalid Dec format: %q", s)}
	}

	sign := 1.0
	switch s[0] {
	case '+':
		// sign stays 1.0
	case '-':
		sign = -1.0
	default:
		return 0, &InvalidCoordinates{Msg: fmt.Sprintf("invalid Dec format (missing sign): %q", s)}
	}

	parts := strings.Split(s[1:], ":")
	if len(parts) != 3 {
		return 0, &InvalidCoordinates{Msg: fmt.Sprintf("invalid Dec format: %q", s)}
	}

	// Degree field must be a bare unsigned integer — reject extra sign prefix.
	if len(parts[0]) == 0 || parts[0][0] < '0' || parts[0][0] > '9' {
		return 0, &InvalidCoordinates{Msg: fmt.Sprintf("invalid Dec degrees in %q", s)}
	}

	d, err := strconv.Atoi(parts[0])
	if err != nil || d < 0 || d > 90 {
		return 0, &InvalidCoordinates{Msg: fmt.Sprintf("invalid Dec degrees in %q", s)}
	}

	m, err := strconv.Atoi(parts[1])
	if err != nil || m < 0 || m > 59 {
		return 0, &InvalidCoordinates{Msg: fmt.Sprintf("invalid Dec minutes in %q", s)}
	}

	sec, err := strconv.ParseFloat(parts[2], 64)
	if err != nil || sec < 0 || sec >= 60 {
		return 0, &InvalidCoordinates{Msg: fmt.Sprintf("invalid Dec seconds in %q", s)}
	}

	totalDeg := float64(d) + float64(m)/60 + sec/3600
	if totalDeg > 90 {
		return 0, &InvalidCoordinates{Msg: fmt.Sprintf("Dec out of range in %q", s)}
	}

	return sign * totalDeg * math.Pi / 180, nil
}

// FormatRA formats float64 radians as "HH:MM:SS.ss" (2 decimal seconds).
// Returns "N/A" if rad is NaN (missing catalog coordinate).
func FormatRA(rad float64) string {
	if math.IsNaN(rad) {
		return "N/A"
	}
	// deg × 240 s/deg × 100 cs/s = total centiseconds of RA time.
	// 24h = 360° × 240 × 100 = 8 640 000 cs.
	totalCS := math.Round(rad * 180 / math.Pi * 240 * 100)
	totalCS = math.Mod(totalCS, 8640000)
	if totalCS < 0 {
		totalCS += 8640000 // math.Mod preserves sign; normalize negative RA
	}
	cs := int(totalCS)
	h := cs / 360000
	cs %= 360000
	m := cs / 6000
	cs %= 6000
	s := float64(cs) / 100
	return fmt.Sprintf("%02d:%02d:%05.2f", h, m, s)
}

// FormatDec formats float64 radians as "±DD:MM:SS.S" (1 decimal second, sign always shown).
// Clamps to [-π/2, π/2]; inputs outside this range snap to the nearest pole.
// Returns "N/A" if rad is NaN (missing catalog coordinate).
func FormatDec(rad float64) string {
	if math.IsNaN(rad) {
		return "N/A"
	}
	if rad > math.Pi/2 {
		rad = math.Pi / 2
	}
	if rad < -math.Pi/2 {
		rad = -math.Pi / 2
	}
	sign := "+"
	if rad < 0 {
		sign = "-"
		rad = -rad
	}
	// deciseconds of arc
	totalDS := math.Round(rad * 180 / math.Pi * 3600 * 10)
	ds := int(totalDS)
	d := ds / 36000
	ds %= 36000
	m := ds / 600
	ds %= 600
	s := float64(ds) / 10
	return fmt.Sprintf("%s%02d:%02d:%04.1f", sign, d, m, s)
}

// RAString returns the object's RA formatted as "HH:MM:SS.ss".
func (o *Object) RAString() string { return FormatRA(o.RA) }

// DecString returns the object's Dec formatted as "±DD:MM:SS.S".
func (o *Object) DecString() string { return FormatDec(o.Dec) }

// RADeg returns the object's RA in degrees.
func (o *Object) RADeg() float64 { return o.RA * 180 / math.Pi }

// DecDeg returns the object's Dec in degrees.
func (o *Object) DecDeg() float64 { return o.Dec * 180 / math.Pi }
