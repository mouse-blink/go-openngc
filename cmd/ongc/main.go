// SPDX-FileCopyrightText: 2026 Adam Stevko
//
// SPDX-License-Identifier: MIT

package main

import (
	"flag"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/xen0l/go-openngc/ongc"
)

// AppVersion is set via: go build -ldflags="-X main.AppVersion=v0.1.0"
var AppVersion = "dev"

const usageText = `Usage: ongc <subcommand> [flags] [args]

Flags must come before positional arguments.

Subcommands:
  view [--details] NAME
  nearby [--radius N] [--catalog all|NGC|IC] RA DEC
  neighbors [--radius N] [--catalog all|NGC|IC] NAME
  search [--catalog all|NGC|IC] [--type CODE,...] [--constellation CODE,...]
         [--minsize N] [--maxsize N] [--uptobmag N] [--uptovmag N]
         [--minra N] [--maxra N] [--mindec N] [--maxdec N]
         [--named SUBSTR] [--withname]
  separation OBJ1 OBJ2
  stats
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usageText)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "view":
		cmdView(os.Args[2:])
	case "nearby":
		cmdNearby(os.Args[2:])
	case "neighbors":
		cmdNeighbors(os.Args[2:])
	case "search":
		cmdSearch(os.Args[2:])
	case "separation":
		cmdSeparation(os.Args[2:])
	case "stats":
		cmdStats(os.Args[2:])
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n\n%s", os.Args[1], usageText)
		os.Exit(2)
	}
}

func cmdView(args []string) {
	fs := flag.NewFlagSet("view", flag.ExitOnError)
	details := fs.Bool("details", false, "show detailed 79-column box output")
	fs.Parse(reorderFlags(fs, args)) //nolint:errcheck // ExitOnError handles errors

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "view: NAME required")
		os.Exit(2)
	}

	name := fs.Arg(0)
	obj, err := ongc.Get(name)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	if *details {
		fmt.Print(obj.PrintDetails())
		fmt.Println()
	} else {
		fmt.Println(obj.String())
	}
}

func cmdNearby(args []string) {
	fs := flag.NewFlagSet("nearby", flag.ExitOnError)
	radius := fs.Float64("radius", 60, "search radius in arcmin")
	catalog := fs.String("catalog", "all", "NGC|IC|all")
	fs.Parse(reorderFlags(fs, args)) //nolint:errcheck // ExitOnError handles errors

	if fs.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "nearby: RA and DEC required")
		os.Exit(2)
	}

	raStr, decStr := fs.Arg(0), fs.Arg(1)
	objs, dists, err := ongc.GetNearby(raStr, decStr, *radius, *catalog)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Printf("\nObjects in proximity of %s %s from nearest to farthest:\n", raStr, decStr)
	for i, obj := range objs {
		fmt.Printf("%.2f° --> %s\n", dists[i], obj.String())
	}
	fmt.Printf("(using a search radius of %.0f arcmin)\n\n", *radius)
}

func cmdNeighbors(args []string) {
	fs := flag.NewFlagSet("neighbors", flag.ExitOnError)
	radius := fs.Float64("radius", 30, "search radius in arcmin")
	catalog := fs.String("catalog", "all", "NGC|IC|all")
	fs.Parse(reorderFlags(fs, args)) //nolint:errcheck // ExitOnError handles errors

	if fs.NArg() < 1 {
		fmt.Fprintln(os.Stderr, "neighbors: NAME required")
		os.Exit(2)
	}

	name := fs.Arg(0)
	objs, dists, err := ongc.GetNeighbors(name, *radius, *catalog)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Look the ref up once more to display its canonical Name (the user
	// may have typed "m31" or "ngc 224"). GetNeighbors already validated
	// resolution, so fall back to the user-supplied string if the second
	// lookup ever fails — avoids a nil-dereference if the invariant breaks.
	refName := name
	if ref, rerr := ongc.Get(name); rerr == nil && ref != nil {
		refName = ref.Name
	}
	fmt.Printf("\n%s neighbors from nearest to farthest:\n", refName)
	for i, obj := range objs {
		fmt.Printf("%.2f° --> %s\n", dists[i], obj.String())
	}
	fmt.Printf("(using a search radius of %.0f arcmin)\n\n", *radius)
}

// splitCSV splits a comma-separated string, trims whitespace, and drops empty elements.
func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	result := parts[:0]
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func cmdSearch(args []string) {
	fs := flag.NewFlagSet("search", flag.ExitOnError)
	catalog := fs.String("catalog", "all", "NGC|IC|all")
	typeStr := fs.String("type", "", "comma-separated type codes")
	constStr := fs.String("constellation", "", "comma-separated constellation codes")
	minsize := fs.Float64("minsize", 0, "minimum size in arcmin")
	maxsize := fs.Float64("maxsize", 0, "maximum size in arcmin")
	uptobmag := fs.Float64("uptobmag", 0, "up to B magnitude")
	uptovmag := fs.Float64("uptovmag", 0, "up to V magnitude")
	minra := fs.Float64("minra", 0, "minimum RA in degrees")
	maxra := fs.Float64("maxra", 0, "maximum RA in degrees")
	mindec := fs.Float64("mindec", 0, "minimum Dec in degrees")
	maxdec := fs.Float64("maxdec", 0, "maximum Dec in degrees")
	named := fs.String("named", "", "only objects whose name contains SUBSTR")
	withname := fs.Bool("withname", false, "only named objects")
	fs.Parse(reorderFlags(fs, args)) //nolint:errcheck // ExitOnError handles errors

	// Track which numeric flags the user explicitly set, so "--minra 0"
	// (filter to RA=0) is distinguishable from "--minra not set at all".
	// Matches PyOngc/Click, which uses None as the unset sentinel.
	isSet := make(map[string]bool)
	fs.Visit(func(f *flag.Flag) { isSet[f.Name] = true })

	var opts []ongc.ListOption //nolint:prealloc
	opts = append(opts, ongc.WithCatalog(*catalog))

	if *typeStr != "" {
		opts = append(opts, ongc.WithType(splitCSV(*typeStr)...))
	}
	if *constStr != "" {
		opts = append(opts, ongc.WithConstellation(splitCSV(*constStr)...))
	}
	if isSet["minsize"] {
		opts = append(opts, ongc.WithMinSize(*minsize))
	}
	if isSet["maxsize"] {
		opts = append(opts, ongc.WithMaxSize(*maxsize))
	}
	if isSet["uptobmag"] {
		opts = append(opts, ongc.WithUpToBMag(*uptobmag))
	}
	if isSet["uptovmag"] {
		opts = append(opts, ongc.WithUpToVMag(*uptovmag))
	}
	if isSet["minra"] {
		opts = append(opts, ongc.WithMinRA(*minra))
	}
	if isSet["maxra"] {
		opts = append(opts, ongc.WithMaxRA(*maxra))
	}
	if isSet["mindec"] {
		opts = append(opts, ongc.WithMinDec(*mindec))
	}
	if isSet["maxdec"] {
		opts = append(opts, ongc.WithMaxDec(*maxdec))
	}
	if *named != "" {
		opts = append(opts, ongc.WithNamed(*named))
	}
	if *withname {
		opts = append(opts, ongc.WithName())
	}

	results, err := ongc.GetList(opts...)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	for _, obj := range results {
		fmt.Println(obj.String())
	}
}

func cmdSeparation(args []string) {
	fs := flag.NewFlagSet("separation", flag.ExitOnError)
	fs.Parse(reorderFlags(fs, args)) //nolint:errcheck // ExitOnError handles errors

	if fs.NArg() < 2 {
		fmt.Fprintln(os.Stderr, "separation: OBJ1 OBJ2 required")
		os.Exit(2)
	}

	name1, name2 := fs.Arg(0), fs.Arg(1)
	sep, err := ongc.GetSeparation(name1, name2)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Both names were validated by GetSeparation; ignore errors on lookup.
	obj1, _ := ongc.Get(name1)
	obj2, _ := ongc.Get(name2)

	deg := int(sep)
	arcmin := int((sep - float64(deg)) * 60)
	arcsec := ((sep-float64(deg))*60 - float64(arcmin)) * 60
	fmt.Printf("Apparent angular separation between %s and %s is:\n", obj1.Name, obj2.Name)
	fmt.Printf("%d° %dm %.2fs\n", deg, arcmin, arcsec)
}

func cmdStats(args []string) {
	// args intentionally ignored; no flags for this subcommand.
	_ = args

	all, _ := ongc.GetList() // no options → all objects
	total := len(all)

	counts := make(map[ongc.ObjectType]int)
	for _, obj := range all {
		counts[obj.Type]++
	}

	fmt.Printf("\ngo-openngc v%s (OpenNGC %s)\n", AppVersion, ongc.DatabaseVersion)
	fmt.Printf("Total objects in database: %d\n", total)
	fmt.Println("Object types statistics:")

	type pair struct {
		code  ongc.ObjectType
		name  string
		count int
	}
	codes := ongc.TypeCodes()
	pairs := make([]pair, 0, len(codes))
	for _, code := range codes {
		displayName, _ := ongc.TypeName(code)
		pairs = append(pairs, pair{code, displayName, counts[code]})
	}
	sort.Slice(pairs, func(i, j int) bool { return string(pairs[i].code) < string(pairs[j].code) })
	for _, p := range pairs {
		fmt.Printf("\t%-29s-> %d\n", p.name, p.count)
	}
}
