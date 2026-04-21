// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

// Command gen produces ongc/catalog_generated.go from the CSV files in
// ongc/data/. Invoke via `go generate ./...`; the generated file is
// committed so regular builds have no runtime CSV parsing cost.
//
// This file intentionally duplicates the small CSV parsing helpers rather
// than importing the ongc package — doing so would create a bootstrap
// ordering problem (the generator runs before catalog_generated.go is
// written, so ongc's init() has nothing to iterate and tests would fail).
package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"go/format"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

const (
	ngcCSV     = "ongc/data/NGC.csv"
	addendum   = "ongc/data/addendum.csv"
	versionF   = "ongc/data/VERSION"
	outputPath = "ongc/catalog_generated.go"
)

// normalizeRe mirrors ongc.NormalizeName — kept in sync with that regex.
var normalizeRe = regexp.MustCompile(`^(NGC|IC)\s?([0-9]{1,4})\s?(NED[0-9]{1,2}|[A-Z]{1,2})?$`)

// normalizeName canonicalizes a catalog name. Kept in sync with ongc.NormalizeName.
func normalizeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToUpper(name)
	m := normalizeRe.FindStringSubmatch(name)
	if m == nil {
		return name
	}
	n, _ := strconv.Atoi(m[2])
	suffix := m[3]
	if strings.HasPrefix(suffix, "NED") {
		digits := suffix[3:]
		nn, _ := strconv.Atoi(digits)
		return fmt.Sprintf("%s%04d NED%02d", m[1], n, nn)
	}
	return fmt.Sprintf("%s%04d%s", m[1], n, suffix)
}

// row mirrors ongc.Object at the field level — struct literal emission uses
// the same ordering as the field layout in ongc/ongc.go.
type row struct {
	ID          int
	Name        string
	Type        string
	RA          float64
	Dec         float64
	Const       string
	MajAx       *float64
	MinAx       *float64
	Pa          *float64
	BMag        *float64
	VMag        *float64
	JMag        *float64
	HMag        *float64
	KMag        *float64
	SBrightn    *float64
	Hubble      string
	Parallax    *float64
	PmRA        *float64
	PmDec       *float64
	RadVel      *float64
	Redshift    *float64
	CstarUMag   *float64
	CstarBMag   *float64
	CstarVMag   *float64
	Messier     string
	NGC         string
	IC          string
	CstarNames  string
	Identifiers []string
	CommonNames []string
	NedNotes    string
	OngNotes    string
	Sources     string
}

// parseRARad parses a sexagesimal RA string ("HH:MM:SS.ss") into radians, returning NaN on error.
func parseRARad(s string) float64 {
	if s == "" {
		return math.NaN()
	}
	parts := strings.Split(s, ":")
	if len(parts) != 3 {
		return math.NaN()
	}
	h, err := strconv.Atoi(parts[0])
	if err != nil || h < 0 || h > 23 {
		return math.NaN()
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil || m < 0 || m > 59 {
		return math.NaN()
	}
	sec, err := strconv.ParseFloat(parts[2], 64)
	if err != nil || sec < 0 || sec >= 60 {
		return math.NaN()
	}
	return (float64(h) + float64(m)/60 + sec/3600) * math.Pi / 12
}

// parseDecRad parses a signed sexagesimal Dec string ("±DD:MM:SS.s") into radians, returning NaN on error.
func parseDecRad(s string) float64 {
	if len(s) < 2 {
		return math.NaN()
	}
	sign := 1.0
	switch s[0] {
	case '+':
	case '-':
		sign = -1.0
	default:
		return math.NaN()
	}
	parts := strings.Split(s[1:], ":")
	if len(parts) != 3 {
		return math.NaN()
	}
	if len(parts[0]) == 0 || parts[0][0] < '0' || parts[0][0] > '9' {
		return math.NaN()
	}
	d, err := strconv.Atoi(parts[0])
	if err != nil || d < 0 || d > 90 {
		return math.NaN()
	}
	m, err := strconv.Atoi(parts[1])
	if err != nil || m < 0 || m > 59 {
		return math.NaN()
	}
	sec, err := strconv.ParseFloat(parts[2], 64)
	if err != nil || sec < 0 || sec >= 60 {
		return math.NaN()
	}
	totalDeg := float64(d) + float64(m)/60 + sec/3600
	if totalDeg > 90 {
		return math.NaN()
	}
	return sign * totalDeg * math.Pi / 180
}

// parseFloat64Ptr returns a pointer to the parsed float, or nil if the input is empty or malformed.
func parseFloat64Ptr(s string) *float64 {
	if s == "" {
		return nil
	}
	v, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return nil
	}
	return &v
}

// splitTrim splits s on sep, trims whitespace, and drops empty parts.
func splitTrim(s, sep string) []string {
	parts := strings.Split(s, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// parseRow converts a CSV record into a row, assigning the sequential catalog ID.
func parseRow(rec []string, id int) row {
	r := row{
		ID:          id,
		Name:        normalizeName(rec[0]),
		Type:        rec[1],
		RA:          parseRARad(rec[2]),
		Dec:         parseDecRad(rec[3]),
		Const:       rec[4],
		MajAx:       parseFloat64Ptr(rec[5]),
		MinAx:       parseFloat64Ptr(rec[6]),
		Pa:          parseFloat64Ptr(rec[7]),
		BMag:        parseFloat64Ptr(rec[8]),
		VMag:        parseFloat64Ptr(rec[9]),
		JMag:        parseFloat64Ptr(rec[10]),
		HMag:        parseFloat64Ptr(rec[11]),
		KMag:        parseFloat64Ptr(rec[12]),
		SBrightn:    parseFloat64Ptr(rec[13]),
		Hubble:      rec[14],
		Parallax:    parseFloat64Ptr(rec[15]),
		PmRA:        parseFloat64Ptr(rec[16]),
		PmDec:       parseFloat64Ptr(rec[17]),
		RadVel:      parseFloat64Ptr(rec[18]),
		Redshift:    parseFloat64Ptr(rec[19]),
		CstarUMag:   parseFloat64Ptr(rec[20]),
		CstarBMag:   parseFloat64Ptr(rec[21]),
		CstarVMag:   parseFloat64Ptr(rec[22]),
		NGC:         rec[24],
		IC:          rec[25],
		CstarNames:  rec[26],
		Identifiers: splitTrim(rec[27], ","),
		CommonNames: splitTrim(rec[28], ","),
		NedNotes:    rec[29],
		OngNotes:    rec[30],
		Sources:     rec[31],
	}
	if m := rec[23]; m != "" {
		if n, err := strconv.Atoi(m); err == nil {
			r.Messier = strconv.Itoa(n)
		}
	}
	return r
}

// readCSV reads a semicolon-delimited CSV, skipping the header row.
func readCSV(path string) ([][]string, error) {
	f, err := os.Open(path) //nolint:gosec // path is a constant data file, not user input
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	r := csv.NewReader(f)
	r.Comma = ';'
	r.LazyQuotes = true

	// Skip the header row.
	if _, err := r.Read(); err != nil {
		return nil, fmt.Errorf("read header from %s: %w", path, err)
	}

	var rows [][]string
	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read %s: %w", path, err)
		}
		rows = append(rows, rec)
	}
	return rows, nil
}

// main parses the OpenNGC CSVs and VERSION file, then generates ongc/catalog_generated.go.
func main() {
	// `go generate` runs from the directory containing the //go:generate
	// directive, which is ongc/. Climb to the module root (the directory
	// with go.mod) so the paths at the top of this file work.
	if err := chdirToModuleRoot(); err != nil {
		die(err)
	}

	versionBytes, err := os.ReadFile(versionF)
	if err != nil {
		die(fmt.Errorf("reading version file: %w", err))
	}
	version := strings.TrimSpace(string(versionBytes))

	var all []row
	id := 1
	for _, path := range []string{ngcCSV, addendum} {
		rows, err := readCSV(path)
		if err != nil {
			die(err)
		}
		for _, rec := range rows {
			all = append(all, parseRow(rec, id))
			id++
		}
	}

	var buf bytes.Buffer
	writeHeader(&buf, version, len(all))
	writeObjects(&buf, all)

	src, err := format.Source(buf.Bytes())
	if err != nil {
		// Dump the un-formatted source to help debug generator output.
		_ = os.WriteFile(outputPath+".broken", buf.Bytes(), 0o644) //nolint:gosec
		die(fmt.Errorf("gofmt generated source: %w (raw output in %s.broken)", err, outputPath))
	}

	if err := os.MkdirAll(filepath.Dir(outputPath), 0o755); err != nil {
		die(err)
	}
	if err := os.WriteFile(outputPath, src, 0o644); err != nil { //nolint:gosec
		die(err)
	}

	fmt.Fprintf(os.Stderr, "generated %s (%d objects, version %s)\n", outputPath, len(all), version)
}

// die prints an error to stderr and exits with status 1.
func die(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

// chdirToModuleRoot finds the module root by walking up from the current directory.
func chdirToModuleRoot() error {
	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return os.Chdir(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return fmt.Errorf("no go.mod found walking up from %s", wd)
		}
		dir = parent
	}
}

// pointerField names every *float64 field of Object we need to pool.
var pointerFields = []string{
	"MajAx", "MinAx", "Pa", "BMag", "VMag", "JMag", "HMag", "KMag",
	"SBrightn", "Parallax", "PmRA", "PmDec", "RadVel", "Redshift",
	"CstarUMag", "CstarBMag", "CstarVMag",
}

// writeHeader emits SPDX headers, package clause, and the DatabaseVersion constant.
func writeHeader(w *bytes.Buffer, version string, count int) {
	// Emit the SPDX tags split so the reuse linter doesn't parse this file
	// (internal/gen/main.go) as claiming its own license via the generator's
	// literal strings. The emitted output is still well-formed REUSE metadata.
	fmt.Fprintln(w, "// "+"SPDX-FileCopyrightText"+": 2026 Adam Stevko")
	fmt.Fprintln(w, "//")
	fmt.Fprintln(w, "// "+"SPDX-License-Identifier"+": MIT")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "// Code generated by internal/gen; DO NOT EDIT.")
	fmt.Fprintf(w, "// Source: ongc/data/NGC.csv + ongc/data/addendum.csv at OpenNGC %s (%d objects).\n", version, count)
	fmt.Fprintln(w)
	fmt.Fprintln(w, "package ongc")
	fmt.Fprintln(w)
	fmt.Fprintln(w, `import "math"`)
	fmt.Fprintln(w)
	fmt.Fprintf(w, "const generatedDatabaseVersion = %q\n", version)
	fmt.Fprintln(w)
	fmt.Fprintln(w, `// Pointer-valued fields are held in parallel pools keyed by row index.`)
	fmt.Fprintln(w, `// Rows without a value store math.NaN(); a sibling init (see ongc.go)`)
	fmt.Fprintln(w, `// wires up Object.MajAx etc. to &pool[i] only when the value is finite.`)
	fmt.Fprintln(w, `// This pattern avoids per-row function calls in the slice literal, which`)
	fmt.Fprintln(w, `// the Go race detector's escape-analysis bitmaps can't accommodate at`)
	fmt.Fprintln(w, `// catalog scale (~14k × 17 pointer fields).`)
	fmt.Fprintln(w)
}

// writeObjects emits float64 pools for pointer fields and the generatedObjects slice.
func writeObjects(w *bytes.Buffer, rows []row) {
	// Emit one float pool per pointer field.
	for _, field := range pointerFields {
		fmt.Fprintf(w, "var pool%s = [...]float64{\n", field)
		for _, r := range rows {
			v := pointerFieldValue(r, field)
			if v == nil {
				fmt.Fprint(w, "\tmath.NaN(),\n")
				continue
			}
			fmt.Fprintf(w, "\t%s,\n", floatLit(*v))
		}
		fmt.Fprintln(w, "}")
		fmt.Fprintln(w)
	}

	// Emit the slice literal with only value fields; pointers are wired at
	// init time by wireCatalogPointers (defined in ongc.go).
	fmt.Fprintln(w, "var generatedObjects = []Object{")
	for _, r := range rows {
		writeRow(w, r)
	}
	fmt.Fprintln(w, "}")
}

// pointerFieldValue returns the pointer field value from a row by name.
func pointerFieldValue(r row, name string) *float64 {
	switch name {
	case "MajAx":
		return r.MajAx
	case "MinAx":
		return r.MinAx
	case "Pa":
		return r.Pa
	case "BMag":
		return r.BMag
	case "VMag":
		return r.VMag
	case "JMag":
		return r.JMag
	case "HMag":
		return r.HMag
	case "KMag":
		return r.KMag
	case "SBrightn":
		return r.SBrightn
	case "Parallax":
		return r.Parallax
	case "PmRA":
		return r.PmRA
	case "PmDec":
		return r.PmDec
	case "RadVel":
		return r.RadVel
	case "Redshift":
		return r.Redshift
	case "CstarUMag":
		return r.CstarUMag
	case "CstarBMag":
		return r.CstarBMag
	case "CstarVMag":
		return r.CstarVMag
	}
	panic("unknown pointer field: " + name)
}

// writeRow emits a Go struct literal for one object, excluding pointer fields.
func writeRow(w *bytes.Buffer, r row) {
	fmt.Fprintf(w, "\t{ID: %d, Name: %q, Type: %q", r.ID, r.Name, r.Type)
	writeFloat(w, "RA", r.RA)
	writeFloat(w, "Dec", r.Dec)
	writeStr(w, "Const", r.Const)
	writeStr(w, "Hubble", r.Hubble)
	writeStr(w, "Messier", r.Messier)
	writeStr(w, "NGC", r.NGC)
	writeStr(w, "IC", r.IC)
	writeStr(w, "CstarNames", r.CstarNames)
	writeStringSlice(w, "Identifiers", r.Identifiers)
	writeStringSlice(w, "CommonNames", r.CommonNames)
	writeStr(w, "NedNotes", r.NedNotes)
	writeStr(w, "OngNotes", r.OngNotes)
	writeStr(w, "Sources", r.Sources)
	fmt.Fprintln(w, "},")
}

// writeFloat emits a float field fragment, omitting zero values.
func writeFloat(w *bytes.Buffer, name string, v float64) {
	if math.IsNaN(v) {
		fmt.Fprintf(w, ", %s: math.NaN()", name)
		return
	}
	if v == 0 {
		return
	}
	fmt.Fprintf(w, ", %s: %s", name, floatLit(v))
}

// writeStr emits a string field fragment, omitting empty values.
func writeStr(w *bytes.Buffer, name, v string) {
	if v == "" {
		return
	}
	fmt.Fprintf(w, ", %s: %q", name, v)
}

// writeStringSlice emits a string slice field fragment, omitting empty slices.
func writeStringSlice(w *bytes.Buffer, name string, v []string) {
	if len(v) == 0 {
		return
	}
	fmt.Fprintf(w, ", %s: []string{", name)
	for i, s := range v {
		if i > 0 {
			fmt.Fprint(w, ", ")
		}
		fmt.Fprintf(w, "%q", s)
	}
	fmt.Fprint(w, "}")
}

// floatLit formats v as a valid Go float literal.
func floatLit(v float64) string {
	if math.IsNaN(v) {
		return "math.NaN()"
	}
	if math.IsInf(v, 1) {
		return "math.Inf(1)"
	}
	if math.IsInf(v, -1) {
		return "math.Inf(-1)"
	}
	return strconv.FormatFloat(v, 'g', -1, 64)
}
