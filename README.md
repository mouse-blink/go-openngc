# go-openngc

[![Go](https://img.shields.io/badge/go-1.26-blue)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/code-MIT-green)](LICENSES/MIT.txt)
[![Data: CC-BY-SA-4.0](https://img.shields.io/badge/data-CC--BY--SA--4.0-lightgrey)](LICENSES/CC-BY-SA-4.0.txt)

Go port of [PyOngc](https://github.com/mattiaverga/PyOngc) — a library and CLI for querying the
[OpenNGC](https://github.com/mattiaverga/OpenNGC) deep-sky object catalog. Provides offline lookup,
search, filtering, and spatial queries for ~14K NGC/IC/Messier objects with zero external dependencies.
CLI output is byte-compatible with PyOngc v1.2.1.

## Installation

```bash
# Library
go get github.com/xen0l/go-openngc/ongc

# CLI
go install github.com/xen0l/go-openngc/cmd/ongc@latest
```

## Usage

### Library

Query the catalog by NGC, IC, or Messier number; common names; or cross-catalog identifiers.
Names are normalized automatically—`"NGC1"`, `"ngc1"`, and `"ngc 1"` all resolve to the same object.

```go
import "github.com/xen0l/go-openngc/ongc"

obj, err := ongc.Get("NGC1")      // NGC, IC, common name, or any identifier
obj, err := ongc.Messier(31)      // Andromeda Galaxy → NGC0224
results  := ongc.Search("Crab")   // substring search across all names
```

Compose filters to find objects of interest:

```go
galaxies, err := ongc.GetList(
    ongc.WithType("G"),
    ongc.WithConstellation("Vir"),
    ongc.WithUpToVMag(12),
)
```

Spatial queries:

```go
// Angular separation between two objects (degrees)
sep, err := ongc.GetSeparation("NGC0001", "NGC0002")

// Find objects within 30 arcminutes of M31 (NGC catalog only)
objs, dists, err := ongc.GetNeighbors("M31", 30, "NGC")

// Same, from arbitrary sky coordinates
objs, dists, err := ongc.GetNearby("00:42:44.33", "+41:16:09.4", 30, "NGC")
```

See the [full API reference on pkg.go.dev](https://pkg.go.dev/github.com/xen0l/go-openngc/ongc).

## Data

The embedded database comes from the [OpenNGC](https://github.com/mattiaverga/OpenNGC) project by
Mattia Verga, released under [CC-BY-SA-4.0](https://creativecommons.org/licenses/by-sa/4.0/).
It contains ~13,969 NGC objects and ~64 addendum entries with coordinates, magnitudes, sizes,
Hubble classifications, radial velocities, and cross-identifications (Messier, PGC, UGC, MCG, etc.).

## Contributing

**Data corrections**: Please contribute upstream to the
[OpenNGC project](https://github.com/mattiaverga/OpenNGC) — that's the authoritative source.

**Library bugs or CLI issues**: Open an issue in this repository.

## Acknowledgments

- [OpenNGC](https://github.com/mattiaverga/OpenNGC) database by Mattia Verga (CC-BY-SA-4.0)
- [PyOngc](https://github.com/mattiaverga/PyOngc) by Mattia Verga (MIT) — reference implementation
- Built with assistance from [Claude](https://anthropic.com) by Anthropic

## License

Code is released under the [MIT License](LICENSES/MIT.txt).
Embedded OpenNGC data is released under [CC-BY-SA-4.0](LICENSES/CC-BY-SA-4.0.txt).
