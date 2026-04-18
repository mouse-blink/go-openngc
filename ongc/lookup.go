// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

package ongc

import (
	"fmt"
	"strings"
)

// LookupResult holds one result from GetMany.
type LookupResult struct {
	Object *Object
	Err    error
}

// Get resolves any name (NGC, IC, Messier alias, common name, cross-ID) to an Object.
// Name is normalized before lookup. Checks byName first, then aliases.
func Get(name string) (*Object, error) {
	norm := NormalizeName(name)
	if obj, ok := byName[norm]; ok {
		return obj, nil
	}
	if obj, ok := aliases[norm]; ok {
		return obj, nil
	}
	return nil, &ObjectNotFound{Name: norm}
}

// GetMany resolves multiple names; each entry carries either Object (non-nil) or Err (non-nil).
func GetMany(names []string) []LookupResult {
	results := make([]LookupResult, len(names))
	for i, name := range names {
		obj, err := Get(name)
		results[i] = LookupResult{Object: obj, Err: err}
	}
	return results
}

// Messier returns the object for Messier number n (1–110).
// Delegates to Get so historical edge cases follow OpenNGC's resolution.
func Messier(n int) (*Object, error) {
	return Get(fmt.Sprintf("M%d", n))
}

// Search does case-insensitive substring match against Name, CommonNames,
// and Identifiers. Returns all matches in catalog order. Empty or
// whitespace-only query returns an empty slice (avoids a silent full
// catalog scan for stray input).
func Search(query string) []*Object {
	query = strings.TrimSpace(query)
	if query == "" {
		return []*Object{}
	}
	upper := strings.ToUpper(query)
	var results []*Object
	for _, obj := range objects {
		if strings.Contains(strings.ToUpper(obj.Name), upper) {
			results = append(results, obj)
			continue
		}
		matched := false
		for _, cn := range obj.CommonNames {
			if strings.Contains(strings.ToUpper(cn), upper) {
				matched = true
				break
			}
		}
		if matched {
			results = append(results, obj)
			continue
		}
		for _, id := range obj.Identifiers {
			if strings.Contains(strings.ToUpper(id), upper) {
				results = append(results, obj)
				break
			}
		}
	}
	return results
}
