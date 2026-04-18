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
- `make gen` — regenerate `ongc/catalog_generated.go` from `ongc/data/*.csv`
- `make gen-check` — CI gate: fails if generated file is out of sync
- Targeted tests: `rtk go test ./ongc/... -run TestGet -v`
- Module: `github.com/xen0l/go-openngc`, Go 1.26
- Data setup (one-time): download `NGC.csv` + `addendum.csv` from the [OpenNGC repo](https://github.com/mattiaverga/OpenNGC) into `ongc/data/`, then `make gen`
- Generate test fixtures: `pip install pyongc==1.2.0 && python scripts/generate_testdata.py` (see catalog-update gotcha below for the rebuild flow when upstream OpenNGC ≠ PyOngc's bundled version)

## Source Layout

> Use `Glob`/`Grep`/`ls` to discover files.

```
ongc/types.go    ongc/ongc.go                ongc/lookup.go   ongc/filter.go
ongc/spatial.go  ongc/format.go              ongc/doc.go      ongc/*_test.go
ongc/data/       ongc/catalog_generated.go   cmd/ongc/        testdata/golden/
internal/gen/    scripts/generate_testdata.py
```

## Architecture

`go generate (internal/gen) → ongc/catalog_generated.go → init() builds byName + aliases indexes`

| File | Responsibility |
|------|---------------|
| `ongc/types.go` | ObjectType enum (21 codes), `TypeName`/`TypeCodes`/`IsKnownType` accessors, error types |
| `ongc/ongc.go` | `Object` struct, `init()` building indexes, `//go:generate` directive |
| `ongc/catalog_generated.go` | Generated: compiled-in `[]Object` slice + `generatedDatabaseVersion` const |
| `internal/gen/main.go` | Generator: reads `ongc/data/*.csv`, emits `catalog_generated.go` |
| `ongc/lookup.go` | `Get`, `Messier`, `GetMany`, `Search` |
| `ongc/filter.go` | `List` with functional options pattern |
| `ongc/spatial.go` | `Neighbors`, `Separation`, `InRegion`, `Nearby`, `validateCatalog` |
| `ongc/format.go` | `String()`, `PrintDetails()` 79-col box, coord formatters, `pyFloat` |
| `cmd/ongc/main.go` | CLI with 6 subcommands, stdlib `flag` only — no cobra |
| `cmd/ongc/flags.go` | `reorderFlags` helper that hoists flags past positionals |

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

1. `ongc/types.go` — ObjectType enum, `TypeName`/`TypeCodes`/`IsKnownType`, error types
2. `ongc/ongc.go` — `Object` struct, `init()` (iterates `generatedObjects`), `//go:generate` directive
3. `internal/gen/main.go` — generator that reads `ongc/data/*.csv` and emits `ongc/catalog_generated.go`
4. `ongc/lookup.go` — `Get`, `Messier`, `GetMany`, `Search`
5. `ongc/filter.go` — `List` with functional options
6. `ongc/spatial.go` — `Neighbors`, `Separation`, `InRegion`, `Nearby`, `validateCatalog`
7. `ongc/format.go` — `String()`, `PrintDetails()`, coord formatters, `pyFloat`
8. `cmd/ongc/main.go` + `cmd/ongc/flags.go` — CLI with stdlib `flag` + `reorderFlags` helper

## Conventions

### Library

- **Error types**: `ObjectNotFound`, `InvalidCoordinates`, `UnknownIdentifier` — all satisfy `error`. `ObjectNotFound` message: `"Object named {NORMALIZED_NAME} not found in the database."`
- **Error wrapping**: `fmt.Errorf("context: %w", err)` — never bare `return err`, never swallow
- **Functional options**: `List()` uses `func(*listOptions)` — add filters without breaking the API
- **Missing CSV values**: empty field → nil for pointer types, `math.NaN()` for float64; render as literal `"N/A"` in all output
- **Name normalization**: trim → uppercase → zero-pad NGC/IC numeric to 4 digits → uppercase trailing letter. `ngc1a` → `NGC0001A`, `ic 673` → `IC0673`, `M31` → alias map → `NGC0224`
- **Catalog data**: compiled into `ongc/catalog_generated.go` at build time by `internal/gen` (run via `go generate ./...` or `make gen`). CSVs in `ongc/data/` are the source of truth; commit both the CSV and the regenerated Go file. Runtime init only builds `byName` and `aliases` indexes over the compiled-in slice — no CSV reader on the hot path.
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
- **Catalog update flow**: updating `ongc/data/*.csv` to a newer OpenNGC version requires: (1) `make gen` to regenerate `ongc/catalog_generated.go`, and (2) regenerate test fixtures against PyOngc 1.2.0 code + new data — `pip install pyongc==1.2.0`, overwrite its bundled `ongc.db` via upstream `createdb.py` pointed at the new CSVs, patch `DBDATE` in its `__init__.py`, then `python scripts/generate_testdata.py`.

### Workflow

- **Shell commands**: prefix all with `rtk`. `rtk go <cmd>` works natively; non-go binaries need `rtk proxy <bin> <subcmd>` (e.g. `rtk proxy golangci-lint run`, `rtk proxy reuse lint`)
- **Scope discipline**: only modify requested files; stop and ask on scope creep; pre-flight for 3+ files
- **Commit style**: conventional format (`feat:`, `fix:`, `refactor:`), one logical change per commit

## Gotchas

- **REUSE compliance**: project uses `REUSE.toml` (converted from `.reuse/dep5`); when adding new unannotated files, add them to the relevant `[[annotations]]` path list in `REUSE.toml` — `REUSE.toml` itself must be listed in its own annotations
- **CLI flag order**: Go stdlib `flag` stops parsing at first non-flag arg — for boolean flags that must work in any position, scan `args` manually rather than relying on `flag.Parse`
