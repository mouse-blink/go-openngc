# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

Note: the v0.x line may ship breaking API changes. Strict SemVer begins at v1.0.

## [v0.1.0] - 2026-04-22

### Added

- Initial public release of `github.com/xen0l/go-openngc`, a Go port of PyOngc.
- Library package `github.com/xen0l/go-openngc/ongc`:
  - Lookup API: `Get`, `Messier`, `GetMany`, `Search`.
  - Filter API: `GetList` with functional options (`WithType`, `WithConstellation`, `WithUpToVMag`, and more).
  - Spatial API: `GetSeparation`, `GetNeighbors`, `GetNearby`, `InRegion`.
  - Typed errors: `ObjectNotFound`, `InvalidCoordinates`, `UnknownIdentifier`.
- CLI `cmd/ongc` with six subcommands matching PyOngc (`list`, `show`, `stats`, `search`, `neighbors`, `nearby`).
- Byte-exact CLI output parity with PyOngc v1.2.0 against OpenNGC v20260307, enforced by golden-file tests.
- Fixture-driven property parity against PyOngc-generated reference values.
- Byte-for-byte error-message parity with PyOngc.
- Zero third-party runtime dependencies (Go standard library only).
- Embedded OpenNGC catalog compiled into the binary at build time (~14,000 objects).
- Build-time version injection via `-ldflags="-X main.AppVersion=..."`; current version surfaced by the `stats` subcommand.

### Data

- Embedded catalog: OpenNGC v20260307, by Mattia Verga, licensed under CC-BY-SA-4.0.

### Licensing

- Go code: MIT.
- Embedded OpenNGC data: CC-BY-SA-4.0 (upstream license preserved).
- REUSE spec compliance: every file is SPDX-annotated via `REUSE.toml`; `make reuse` is a CI gate.
