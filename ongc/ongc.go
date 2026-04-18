// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

package ongc

//go:generate go run ../internal/gen

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

var (
	objects         []*Object
	byName          map[string]*Object
	aliases         map[string]*Object
	DatabaseVersion = generatedDatabaseVersion
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

	// searchKeys is a pre-uppercased, pre-flattened corpus used by Search:
	// Name + every CommonName + every Identifier, all uppercase. Built once
	// in init so Search doesn't have to call strings.ToUpper ~50K times
	// per invocation. Unexported because it's an implementation detail.
	searchKeys []string
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

// init builds the lookup indexes over the compiled-in catalog and wires up
// the pointer-valued fields (MajAx, MinAx, BMag, etc.) from pool arrays in
// catalog_generated.go. No CSV parsing at runtime — generatedObjects and
// the parallel pools come from catalog_generated.go, produced by
// `go generate ./...`.
//
// Pointer fields live in parallel [14033]float64 pools rather than inline
// fPtr(...) calls so race-detector builds stay within the compiler's
// escape-analysis bitmap budget.
func init() {
	assertPoolLengths()

	n := len(generatedObjects)
	objects = make([]*Object, 0, n)
	byName = make(map[string]*Object, n)
	// ~3.2 aliases per object empirically (Messier + NGC/IC cross + common +
	// identifiers); overshoot slightly to avoid map growth.
	aliases = make(map[string]*Object, n*4)

	// Single pass: wire pointer-valued fields from the parallel pools and
	// build the indexes in the same loop. One cache sweep over the 5 MB
	// object slice instead of two (previously wireCatalogPointers then a
	// separate index-build loop).
	for i := range generatedObjects {
		obj := &generatedObjects[i]
		wireObjectPointers(obj, i)
		buildSearchKeys(obj)
		objects = append(objects, obj)
		byName[obj.Name] = obj
		buildAliases(obj)
		addToDecBand(obj)
	}
}

// buildSearchKeys populates obj.searchKeys with an uppercased corpus of
// every name Search should match against — Name, CommonNames, Identifiers.
// Done once in init to avoid per-call strings.ToUpper allocations on the
// Search hot path (~51K identifier allocations per query otherwise).
func buildSearchKeys(obj *Object) {
	keys := make([]string, 0, 1+len(obj.CommonNames)+len(obj.Identifiers))
	// obj.Name is already uppercase after NormalizeName.
	keys = append(keys, obj.Name)
	for _, n := range obj.CommonNames {
		keys = append(keys, strings.ToUpper(n))
	}
	for _, id := range obj.Identifiers {
		keys = append(keys, strings.ToUpper(id))
	}
	obj.searchKeys = keys
}

// wireObjectPointers sets the pointer-valued fields of a single Object to
// addresses in the package-level pools, one per row index. Entries with
// NaN in their pool represent "not present" and leave the pointer nil.
//
// Split out of init for test targeting — see pool_guard_test.go.
func wireObjectPointers(obj *Object, i int) {
	if v := &poolMajAx[i]; !math.IsNaN(*v) {
		obj.MajAx = v
	}
	if v := &poolMinAx[i]; !math.IsNaN(*v) {
		obj.MinAx = v
	}
	if v := &poolPa[i]; !math.IsNaN(*v) {
		obj.Pa = v
	}
	if v := &poolBMag[i]; !math.IsNaN(*v) {
		obj.BMag = v
	}
	if v := &poolVMag[i]; !math.IsNaN(*v) {
		obj.VMag = v
	}
	if v := &poolJMag[i]; !math.IsNaN(*v) {
		obj.JMag = v
	}
	if v := &poolHMag[i]; !math.IsNaN(*v) {
		obj.HMag = v
	}
	if v := &poolKMag[i]; !math.IsNaN(*v) {
		obj.KMag = v
	}
	if v := &poolSBrightn[i]; !math.IsNaN(*v) {
		obj.SBrightn = v
	}
	if v := &poolParallax[i]; !math.IsNaN(*v) {
		obj.Parallax = v
	}
	if v := &poolPmRA[i]; !math.IsNaN(*v) {
		obj.PmRA = v
	}
	if v := &poolPmDec[i]; !math.IsNaN(*v) {
		obj.PmDec = v
	}
	if v := &poolRadVel[i]; !math.IsNaN(*v) {
		obj.RadVel = v
	}
	if v := &poolRedshift[i]; !math.IsNaN(*v) {
		obj.Redshift = v
	}
	if v := &poolCstarUMag[i]; !math.IsNaN(*v) {
		obj.CstarUMag = v
	}
	if v := &poolCstarBMag[i]; !math.IsNaN(*v) {
		obj.CstarBMag = v
	}
	if v := &poolCstarVMag[i]; !math.IsNaN(*v) {
		obj.CstarVMag = v
	}
}

// assertPoolLengths panics if the generator ever emits pool arrays with a
// length different from len(generatedObjects). wireObjectPointers indexes
// every pool by the object's row index; a mismatch would otherwise surface
// as an obscure out-of-range panic later. Fails fast at init with a clear
// message naming the offending pool.
func assertPoolLengths() {
	n := len(generatedObjects)
	expectPoolLen("MajAx", len(poolMajAx), n)
	expectPoolLen("MinAx", len(poolMinAx), n)
	expectPoolLen("Pa", len(poolPa), n)
	expectPoolLen("BMag", len(poolBMag), n)
	expectPoolLen("VMag", len(poolVMag), n)
	expectPoolLen("JMag", len(poolJMag), n)
	expectPoolLen("HMag", len(poolHMag), n)
	expectPoolLen("KMag", len(poolKMag), n)
	expectPoolLen("SBrightn", len(poolSBrightn), n)
	expectPoolLen("Parallax", len(poolParallax), n)
	expectPoolLen("PmRA", len(poolPmRA), n)
	expectPoolLen("PmDec", len(poolPmDec), n)
	expectPoolLen("RadVel", len(poolRadVel), n)
	expectPoolLen("Redshift", len(poolRedshift), n)
	expectPoolLen("CstarUMag", len(poolCstarUMag), n)
	expectPoolLen("CstarBMag", len(poolCstarBMag), n)
	expectPoolLen("CstarVMag", len(poolCstarVMag), n)
}

// expectPoolLen is the single-pool guard used by assertPoolLengths. Factored
// out so it's unit-testable without mocking package globals.
func expectPoolLen(name string, got, want int) {
	if got != want {
		panic(fmt.Sprintf("ongc: generated pool %q has length %d, want %d (generator bug)", name, got, want))
	}
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
