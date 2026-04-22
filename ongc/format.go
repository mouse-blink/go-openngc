// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

package ongc

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"unicode/utf8"
)

// na is the placeholder string for nil or NaN fields in formatted output.
const na = "N/A"

// String returns the short one-line representation: "NGC0001, Galaxy in Peg".
// Unknown type codes fall through via TypeLongName, matching PyOngc which
// passes raw type strings unchanged rather than dropping the row.
func (o *Object) String() string {
	return fmt.Sprintf("%s, %s in %s", o.Name, o.TypeLongName(), o.Const)
}

// pyFloat formats a float64 like Python's str(float): always includes a decimal
// point (e.g. 120.0 → "120.0", 1.57 → "1.57"). Used for byte-exact PyOngc parity.
//
// NaN and Inf are lowercased to match Python's str(float('nan'))='nan' and
// str(float('inf'))='inf' / str(float('-inf'))='-inf'. Go's FormatFloat emits
// "NaN" and "+Inf"/"-Inf", which would diverge.
func pyFloat(v float64) string {
	switch {
	case math.IsNaN(v):
		return "nan"
	case math.IsInf(v, 1):
		return "inf"
	case math.IsInf(v, -1):
		return "-inf"
	}
	s := strconv.FormatFloat(v, 'g', -1, 64)
	if !strings.Contains(s, ".") && !strings.Contains(s, "e") {
		s += ".0"
	}
	return s
}

// padVisual pads s to the given width using Unicode code point count, not bytes.
// This matches Python's string formatting which counts characters, not UTF-8 bytes.
func padVisual(s string, width int) string {
	n := utf8.RuneCountInString(s)
	if n >= width {
		return s
	}
	return s + strings.Repeat(" ", width-n)
}

// justifyText wraps text into 79-char box rows with 4-space indent.
// Splits on whitespace (matching PyOngc's _justifyText), wraps at 73 visual
// columns. Column counts go through utf8.RuneCountInString so multibyte
// characters in common names or notes don't break the box alignment — Go's
// fmt "%-73s" pads by bytes, which would under-pad any line with non-ASCII.
func justifyText(text string) string {
	var result strings.Builder
	chunks := strings.Fields(text)
	var line []string
	lineLength := 0
	for _, chunk := range chunks {
		chunkLen := utf8.RuneCountInString(chunk)
		lineLength += chunkLen + 1
		if lineLength <= 73 {
			line = append(line, chunk)
		} else {
			fmt.Fprintf(&result, "|    %s|\n", padVisual(strings.Join(line, " "), 73))
			line = []string{chunk}
			lineLength = chunkLen + 1
		}
	}
	if len(line) > 0 {
		fmt.Fprintf(&result, "|    %s|\n", padVisual(strings.Join(line, " "), 73))
	}
	return result.String()
}

// detailMag formats a magnitude pointer as a Python float string or "N/A".
func detailMag(v *float64) string {
	if v == nil {
		return na
	}
	return pyFloat(*v)
}

// PrintDetails returns a 79-character-wide box matching PyOngc v1.2.0 printDetails output.
func (o *Object) PrintDetails() string {
	var sb strings.Builder
	sep := "+" + strings.Repeat("-", 77) + "+\n"

	addRow := func(s string) {
		fmt.Fprintf(&sb, "| %-76s|\n", s)
	}
	blankRow := func() {
		sb.WriteString("|" + strings.Repeat(" ", 77) + "|\n")
	}

	// Header
	sb.WriteString(sep)
	fmt.Fprintf(&sb, "| Id: %-10dName: %-18sType: %-32s|\n",
		o.ID, o.Name, o.TypeLongName())
	fmt.Fprintf(&sb, "| R.A.: %-17sDec.: %-17sConstellation: %-15s|\n",
		o.RAString(), o.DecString(), o.Const)

	// "Also known as" section: Messier + NGC cross-ref + IC cross-ref
	var knownAs []string
	if o.Messier != "" {
		if n, err := strconv.Atoi(o.Messier); err == nil {
			knownAs = append(knownAs, fmt.Sprintf("M%03d", n))
		}
	}
	if o.NGC != "" {
		knownAs = append(knownAs, NormalizeName("NGC"+o.NGC))
	}
	if o.IC != "" {
		knownAs = append(knownAs, NormalizeName("IC"+o.IC))
	}
	if len(knownAs) > 0 {
		addRow("Also known as:")
		sb.WriteString(justifyText(strings.Join(knownAs, ", ")))
	}

	// "Common names" section
	if len(o.CommonNames) > 0 {
		addRow("Common names:")
		sb.WriteString(justifyText(strings.Join(o.CommonNames, ", ")))
	}

	sb.WriteString(sep)

	// Dimensions
	majStr, minStr := na, na
	if o.MajAx != nil {
		majStr = pyFloat(*o.MajAx) + "'"
	}
	if o.MinAx != nil {
		minStr = pyFloat(*o.MinAx) + "'"
	}
	paStr := na
	if o.Pa != nil {
		paStr = fmt.Sprintf("%d°", int(*o.Pa))
	}
	sb.WriteString("| Major axis: " + padVisual(majStr, 11) +
		"Minor axis: " + padVisual(minStr, 11) +
		"Position angle: " + padVisual(paStr, 14) + "|\n")

	// Magnitudes
	fmt.Fprintf(&sb, "| B-mag: %-8sV-mag: %-8sJ-mag: %-8sH-mag: %-8sK-mag: %-9s|\n",
		detailMag(o.BMag), detailMag(o.VMag), detailMag(o.JMag),
		detailMag(o.HMag), detailMag(o.KMag))

	blankRow()

	// Kinematic data
	parallaxStr := na
	if o.Parallax != nil {
		parallaxStr = pyFloat(*o.Parallax) + "mas"
	}
	radvelStr := na
	if o.RadVel != nil {
		radvelStr = fmt.Sprintf("%dkm/s", int(*o.RadVel))
	}
	redshiftStr := na
	if o.Redshift != nil {
		redshiftStr = fmt.Sprintf("%.6f", *o.Redshift)
	}
	fmt.Fprintf(&sb, "| Parallax: %-13sRadial velocity: %-14sRedshift: %-12s|\n",
		parallaxStr, radvelStr, redshiftStr)

	blankRow()

	// Proper motion
	pmRAStr := na
	if o.PmRA != nil {
		pmRAStr = pyFloat(*o.PmRA) + "mas/yr"
	}
	pmDecStr := na
	if o.PmDec != nil {
		pmDecStr = pyFloat(*o.PmDec) + "mas/yr"
	}
	fmt.Fprintf(&sb, "| Proper apparent motion in RA: %-46s|\n", pmRAStr)
	fmt.Fprintf(&sb, "| Proper apparent motion in Dec: %-45s|\n", pmDecStr)

	blankRow()

	// Galaxy-only section
	if o.Type == TypeGalaxy {
		sbStr := na
		if o.SBrightn != nil {
			sbStr = pyFloat(*o.SBrightn)
		}
		hubble := o.Hubble
		if hubble == "" {
			hubble = na
		}
		fmt.Fprintf(&sb, "| Surface brightness: %-10sHubble classification: %-23s|\n",
			sbStr, hubble)
	}

	// Central star section (PNs and anything with cstar data)
	hasCstar := o.CstarNames != "" || o.CstarUMag != nil || o.CstarBMag != nil || o.CstarVMag != nil
	if hasCstar {
		if o.CstarNames != "" {
			addRow("Central star identifiers:")
			parts := strings.Split(o.CstarNames, ",")
			for i, p := range parts {
				parts[i] = strings.TrimSpace(p)
			}
			sb.WriteString(justifyText(strings.Join(parts, ", ")))
			blankRow()
		}
		addRow("Central star magnitudes:")
		fmt.Fprintf(&sb, "|    U-mag: %-17sB-mag: %-17sV-mag: %-18s|\n",
			detailMag(o.CstarUMag), detailMag(o.CstarBMag), detailMag(o.CstarVMag))
	}

	sb.WriteString(sep)

	// Other identifiers
	if len(o.Identifiers) > 0 {
		addRow("Other identifiers:")
		sb.WriteString(justifyText(strings.Join(o.Identifiers, ", ")))
		sb.WriteString(sep)
	}

	// NED notes
	if o.NedNotes != "" {
		addRow("NED notes:")
		sb.WriteString(justifyText(o.NedNotes))
		sb.WriteString(sep)
	}

	// OpenNGC notes
	if o.OngNotes != "" {
		addRow("OpenNGC notes:")
		sb.WriteString(justifyText(o.OngNotes))
		sb.WriteString(sep)
	}

	return sb.String()
}
