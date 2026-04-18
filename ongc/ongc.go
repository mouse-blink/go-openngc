// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

package ongc

import (
	"embed"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"regexp"
	"strconv"
	"strings"
)

//go:embed data/NGC.csv data/addendum.csv data/VERSION
var csvData embed.FS

var (
	objects         []*Object
	byName          map[string]*Object
	aliases         map[string]*Object
	DatabaseVersion string
)

// Object represents a single deep-sky object from the OpenNGC catalog.
//
// Returned pointers (e.g. from Get, GetMany, GetNeighbors, InRegion, List)
// point into the package's shared catalog slice. Callers MUST treat Object
// values as read-only — mutating a field poisons every subsequent lookup
// for the same object. If you need to modify a field, copy the struct first.
type Object struct {
	ID          int
	Name        string
	Type        ObjectType
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

// TypeLongName returns the human-readable type name for display.
// Unknown codes fall back to the raw code string (PyOngc pass-through).
func (o *Object) TypeLongName() string {
	if name, ok := TypeName(o.Type); ok {
		return name
	}
	return string(o.Type)
}

// normalizeRe mirrors PyOngc v1.2.0's NGC|IC pattern from ongc._recognize_name:
// ^((?:NGC|IC)\s?)(\d{1,4})\s?((NED)(\d{1,2})|[A-Z]{1,2})?$
// prefix, optional whitespace, 1-4 digits, optional whitespace, optional
// trailing component (NED crossref NN or 1-2 uppercase letters).
var normalizeRe = regexp.MustCompile(`^(NGC|IC)\s?([0-9]{1,4})\s?(NED[0-9]{1,2}|[A-Z]{1,2})?$`)

// NormalizeName normalizes a catalog name to canonical form.
// NGC/IC names are zero-padded to 4 digits; all output is uppercased.
// Accepts forms like "ngc1a", "NGC 1 A", "ic 673", "NGC0001NED01".
// "NGC-1" (hyphen) stays unchanged — PyOngc rejects hyphens too.
// Messier inputs ("M31") are passed through untouched; alias lookup handles them.
func NormalizeName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ToUpper(name)
	m := normalizeRe.FindStringSubmatch(name)
	if m == nil {
		return name
	}
	n, _ := strconv.Atoi(m[2])
	suffix := m[3]
	if strings.HasPrefix(suffix, "NED") {
		// "NED1" → "NED01" zero-pad to 2 digits for PyOngc parity.
		digits := suffix[3:]
		nn, _ := strconv.Atoi(digits)
		return fmt.Sprintf("%s%04d NED%02d", m[1], n, nn)
	}
	return fmt.Sprintf("%s%04d%s", m[1], n, suffix)
}

func init() {
	objects = make([]*Object, 0, 14100)
	byName = make(map[string]*Object)
	aliases = make(map[string]*Object)

	vb, _ := csvData.ReadFile("data/VERSION")
	DatabaseVersion = strings.TrimSpace(string(vb))

	mustLoadCSV("data/NGC.csv")
	mustLoadCSV("data/addendum.csv")
}

func mustLoadCSV(path string) {
	f, err := csvData.Open(path)
	if err != nil {
		panic(err)
	}
	defer func() { _ = f.Close() }()

	r := csv.NewReader(f)
	r.Comma = ';'
	r.LazyQuotes = true

	if _, err := r.Read(); err != nil {
		panic(err)
	}

	for {
		rec, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}

		obj := parseRecord(rec)
		obj.ID = len(objects) + 1
		objects = append(objects, obj)
		byName[obj.Name] = obj
		buildAliases(obj)
	}
}

func parseRecord(rec []string) *Object {
	obj := &Object{}

	obj.Name = NormalizeName(rec[0])
	obj.Type = ObjectType(rec[1])

	// Missing or malformed coords become NaN so spatial queries can skip them.
	// Matches PyOngc, which stores NULL for these and returns "N/A" in display.
	obj.RA = math.NaN()
	obj.Dec = math.NaN()
	if rec[2] != "" {
		if ra, err := ParseRA(rec[2]); err == nil {
			obj.RA = ra
		}
	}
	if rec[3] != "" {
		if dec, err := ParseDec(rec[3]); err == nil {
			obj.Dec = dec
		}
	}

	obj.Const = rec[4]
	obj.MajAx = parseFloat64Ptr(rec[5])
	obj.MinAx = parseFloat64Ptr(rec[6])
	obj.Pa = parseFloat64Ptr(rec[7])
	obj.BMag = parseFloat64Ptr(rec[8])
	obj.VMag = parseFloat64Ptr(rec[9])
	obj.JMag = parseFloat64Ptr(rec[10])
	obj.HMag = parseFloat64Ptr(rec[11])
	obj.KMag = parseFloat64Ptr(rec[12])
	obj.SBrightn = parseFloat64Ptr(rec[13])
	obj.Hubble = rec[14]
	obj.Parallax = parseFloat64Ptr(rec[15])
	obj.PmRA = parseFloat64Ptr(rec[16])
	obj.PmDec = parseFloat64Ptr(rec[17])
	obj.RadVel = parseFloat64Ptr(rec[18])
	obj.Redshift = parseFloat64Ptr(rec[19])
	obj.CstarUMag = parseFloat64Ptr(rec[20])
	obj.CstarBMag = parseFloat64Ptr(rec[21])
	obj.CstarVMag = parseFloat64Ptr(rec[22])
	if m := rec[23]; m != "" {
		if n, err := strconv.Atoi(m); err == nil {
			obj.Messier = strconv.Itoa(n)
		}
	}
	obj.NGC = rec[24]
	obj.IC = rec[25]
	obj.CstarNames = rec[26]
	obj.Identifiers = splitTrim(rec[27], ",")
	obj.CommonNames = splitTrim(rec[28], ",")
	obj.NedNotes = rec[29]
	obj.OngNotes = rec[30]
	obj.Sources = rec[31]

	return obj
}

func buildAliases(obj *Object) {
	if obj.Messier != "" {
		aliasInsert("M"+obj.Messier, obj)
	}
	for _, name := range obj.CommonNames {
		aliasInsert(strings.ToUpper(name), obj)
	}
	for _, id := range obj.Identifiers {
		aliasInsert(strings.ToUpper(id), obj)
	}
	if obj.NGC != "" {
		aliasInsert(NormalizeName("NGC"+obj.NGC), obj)
	}
	if obj.IC != "" {
		aliasInsert(NormalizeName("IC"+obj.IC), obj)
	}
}

// aliasInsert adds key → obj only if the key is not already present.
func aliasInsert(key string, obj *Object) {
	if _, exists := aliases[key]; !exists {
		aliases[key] = obj
	}
}

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
