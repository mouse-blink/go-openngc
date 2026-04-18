# go-openngc

Go port of [PyOngc](https://github.com/mattiaverga/PyOngc) — library and CLI for querying the
[OpenNGC](https://github.com/mattiaverga/OpenNGC) deep-sky object catalog (~14K objects).
Output-compatible with PyOngc v1.2.0 against OpenNGC v20260307. **Zero external dependencies** (stdlib only).

## Quick Start

- `rtk go build ./...` — compile
- `rtk go test ./...` — run all tests
- `rtk go vet ./...` — static analysis
- `rtk go install ./cmd/ongc` — install CLI
- `rtk go mod tidy` — tidy module
- Targeted tests: `rtk go test ./ongc/... -run TestGet -v`
- Module: `github.com/xen0l/go-openngc`, Go 1.26
- Data setup (one-time): download `NGC.csv` + `addendum.csv` from the [OpenNGC repo](https://github.com/mattiaverga/OpenNGC) into `ongc/data/`
- Generate test fixtures: `pip install pyongc && python scripts/generate_testdata.py`

## Source Layout

> Use `Glob`/`Grep`/`ls` to discover files.

```
ongc/types.go    ongc/ongc.go     ongc/lookup.go   ongc/filter.go
ongc/spatial.go  ongc/format.go   ongc/doc.go      ongc/*_test.go
ongc/data/       cmd/ongc/        testdata/golden/ testdata/objects.json
scripts/generate_testdata.py
```

## Architecture

`init() → parse NGC.csv + addendum.csv → []Object → byName + aliases indexes`

| File | Responsibility |
|------|---------------|
| `ongc/types.go` | ObjectType enum (21 codes), type code maps, error types |
| `ongc/ongc.go` | Object struct, CSV parsing, `init()`, `go:embed`, index building |
| `ongc/lookup.go` | `Get`, `Messier`, `GetMany`, `Search` |
| `ongc/filter.go` | `List` with functional options pattern |
| `ongc/spatial.go` | `Neighbors`, `Separation`, `InRegion`, `Nearby` — brute-force spherical at 14K objects |
| `ongc/format.go` | `String()`, `PrintDetails()` 79-col box, coord formatters |
| `cmd/ongc/main.go` | CLI with 6 subcommands, stdlib `flag` only — no cobra |

**CSV format**: `;` delimiter, 32 columns, `HH:MM:SS.ss` RA, `±DD:MM:SS.s` Dec, empty field = nil/NaN.
**Coordinates**: stored as float64 radians internally; `RADeg()`, `DecDeg()`, `RAString()`, `DecString()` for callers.
**Object ID**: row index + 1 (matches PyOngc SQLite AUTOINCREMENT). NGC.csv rows first, addendum.csv appended.

### Object type codes

| Code | Long form | Code | Long form |
|------|-----------|------|-----------|
| `*` | `Star` | `PN` | `Planetary Nebula` |
| `**` | `Double star` | `HII` | `HII Ionized region` |
| `*Ass` | `Association of stars` | `DrkN` | `Dark Nebula` |
| `OCl` | `Open Cluster` | `EmN` | `Emission Nebula` |
| `GCl` | `Globular Cluster` | `Neb` | `Nebula` |
| `Cl+N` | `Star cluster + Nebula` | `RfN` | `Reflection Nebula` |
| `G` | `Galaxy` | `SNR` | `Supernova remnant` |
| `GPair` | `Galaxy Pair` | `Nova` | `Nova star` |
| `GTrpl` | `Galaxy Triplet` | `NonEx` | `Nonexistent object` |
| `GGroup` | `Group of galaxies` | `Dup` | `Duplicated record` |
| `Other` | `Object of other/unknown type` | | |

## Build Order

Build files in this order; after each: `rtk go build ./...` + `rtk go vet ./...` + `rtk go test ./...`

1. `ongc/types.go` — ObjectType enum, type code maps, error types
2. `ongc/ongc.go` — Object struct, CSV parsing, `init()`, embed, indexes
3. `ongc/lookup.go` — `Get`, `Messier`, `GetMany`, `Search`
4. `ongc/filter.go` — `List` with functional options
5. `ongc/spatial.go` — `Neighbors`, `Separation`, `InRegion`, `Nearby`
6. `ongc/format.go` — `String()`, `PrintDetails()`, coord formatters
7. `cmd/ongc/main.go` — CLI

## Conventions

### Library

- **Error types**: `ObjectNotFound`, `InvalidCoordinates`, `UnknownIdentifier` — all satisfy `error`. `ObjectNotFound` message: `"Object named {NORMALIZED_NAME} not found in the database."`
- **Error wrapping**: `fmt.Errorf("context: %w", err)` — never bare `return err`, never swallow
- **Functional options**: `List()` uses `func(*listOptions)` — add filters without breaking the API
- **Missing CSV values**: empty field → nil for pointer types, `math.NaN()` for float64; render as literal `"N/A"` in all output
- **Name normalization**: trim → uppercase → zero-pad NGC/IC numeric to 4 digits → uppercase trailing letter. `ngc1a` → `NGC0001A`, `ic 673` → `IC0673`, `M31` → alias map → `NGC0224`
- **go:embed**: `ongc/data/NGC.csv` and `ongc/data/addendum.csv` embedded at compile time — no runtime file access
- **Byte-exact output** (required for PyOngc v1.2.0 compatibility against OpenNGC v20260307):
  - `String()`: `"{name}, {type_long} in {const}"` e.g. `"NGC0001, Galaxy in Peg"`
  - RA: `fmt.Sprintf("%02d:%02d:%05.2f", h, m, s)` → `"00:07:15.84"`
  - Dec: sign always shown, 1 decimal → `"+27:42:29.1"`, `"-00:09:01.3"`
  - Radial velocity: `fmt.Sprintf("%dkm/s", int(radvel))` — **no space** before unit
  - Separation: `fmt.Sprintf("%d° %dm %.2fs", deg, arcmin, arcsec)` — int deg, int arcmin, .2f arcsec
  - PrintDetails box: 79 chars wide, border `"+" + 77×"-" + "+"`, conditional sections for galaxy/PN types; wrap identifiers/notes at ~74 cols, indent continuation 4 spaces

### Testing

- **Golden files**: run CLI via `exec.Command`, compare stdout byte-for-byte with `testdata/golden/*.txt`
- **Property tests**: load `testdata/objects.json`, unmarshal, assert struct fields match PyOngc reference values
- **Error tests**: bad name → `ObjectNotFound`; bad coords → `InvalidCoordinates`
- **Test-first for bugs**: write failing test, verify it fails, then fix
- **Generate fixtures**: `pip install pyongc && python scripts/generate_testdata.py` (requires PyOngc installed)
- **Catalog update gotcha**: updating `ongc/data/*.csv` to a newer OpenNGC version may cause `TestFullCatalogParity` failures for objects whose fields changed between versions — fix affected entries in `testdata/catalog_fields.json` individually

### Workflow

- **Shell commands**: prefix all with `rtk`. `rtk go <cmd>` works natively; non-go binaries need `rtk proxy <bin> <subcmd>` (e.g. `rtk proxy golangci-lint run`, `rtk proxy reuse lint`)
- **Scope discipline**: only modify requested files; stop and ask on scope creep; pre-flight for 3+ files
- **Commit style**: conventional format (`feat:`, `fix:`, `refactor:`), one logical change per commit

## Gotchas

- **REUSE compliance**: project uses `REUSE.toml` (converted from `.reuse/dep5`); when adding new unannotated files, add them to the relevant `[[annotations]]` path list in `REUSE.toml` — `REUSE.toml` itself must be listed in its own annotations
- **CLI flag order**: Go stdlib `flag` stops parsing at first non-flag arg — for boolean flags that must work in any position, scan `args` manually rather than relying on `flag.Parse`
