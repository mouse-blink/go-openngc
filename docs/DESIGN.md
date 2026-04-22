# Design & Philosophy

This document explains what go-openngc is, what it is not, and the commitments
it makes to its users. It is written for engineers deciding whether to depend
on the project — including those skeptical of AI-assisted code. Where a claim
can be verified, the relevant file path, commit, or test name is cited so you
can check it yourself.

## Contents

1. [Purpose and non-goals](#1-purpose-and-non-goals)
2. [Upstream relationship](#2-upstream-relationship)
3. [Compatibility posture](#3-compatibility-posture)
4. [Interface stability](#4-interface-stability)
5. [Zero external runtime dependencies](#5-zero-external-runtime-dependencies)
6. [Performance: moving parsing to build time](#6-performance-moving-parsing-to-build-time)
7. [On AI-assisted authorship](#7-on-ai-assisted-authorship)
8. [Quality gates](#8-quality-gates)
9. [Maintenance discipline](#9-maintenance-discipline)

## 1. Purpose and non-goals

go-openngc is a native-Go client for the [OpenNGC](https://github.com/mattiaverga/OpenNGC)
deep-sky object catalog. Its purpose is narrow: give Go applications an
offline, license-clear, dependency-free way to query ~14,000 objects from
the NGC (New General Catalogue), IC (Index Catalogue), and Messier catalogs,
without shelling out to Python or bundling a CPython runtime.

The project is deliberately **not**:

- A replacement for, fork of, or competitor to [OpenNGC](https://github.com/mattiaverga/OpenNGC).
  OpenNGC remains the single source of truth for catalog contents.
- A replacement for, fork of, or competitor to [PyOngc](https://github.com/mattiaverga/PyOngc).
  PyOngc is the reference implementation; go-openngc follows it.
- A framework, toolkit, or platform. It exposes a small, focused API surface
  and a CLI that mirrors PyOngc's.

## 2. Upstream relationship

Work is routed to the project that can actually fix it.

- **Data corrections** — wrong coordinates, misclassified objects, missing
  entries, new additions — belong to
  [OpenNGC](https://github.com/mattiaverga/OpenNGC). When a change is accepted
  upstream, go-openngc ingests the new CSV release and ships a data refresh.
- **Go-side library and CLI bugs** — incorrect parsing, wrong output
  formatting, API regressions, build or install issues — belong here.
- **Semantic deltas versus PyOngc** — any case where the same query produces
  different results — also belong here, and are treated as bugs unless
  explicitly documented as intentional.

This split is already reflected in the [README's contributing section](../README.md#contributing);
this document states the reasoning.

## 3. Compatibility posture

The goal is simple: the `ongc` binary should be a drop-in replacement for
PyOngc's CLI, and library calls should return the same values for the same
inputs.

**CLI byte-exact output parity with PyOngc v1.2.0 — against the bundled
OpenNGC release (currently v20260307) — is a hard invariant.** Any regression in parity is treated as a
bug. Parity is enforced by the tests described in [section 8](#8-quality-gates),
not just aspired to.

**Narrow exception:** there are cases where strict parity would harm Go
consumers — for example, a format that forces callers into fragile
string-parsing when a structured alternative is obvious. In such cases the
project author may approve a specific, documented divergence. Each
divergence requires:

1. A release-notes entry describing the change and its motivation.
2. An updated golden file demonstrating the new expected output.
3. A source comment at the point of divergence pointing back to the release
   notes.

Divergences are exceptions, not the norm. The default answer is always "match
PyOngc."

For library idioms — package layout, error-handling style, API surface shape —
Go conventions win. Observable CLI behavior still falls under the hard
invariant above.

## 4. Interface stability

The library API and the CLI are on different tracks, so they get separate
promises.

### Library (`github.com/xen0l/go-openngc/ongc`)

- **Pre-1.0 (current).** The exported API may change between minor releases
  while it settles. Callers should pin a specific `v0.y.z` in `go.mod` and
  read release notes before upgrading. Breaking changes, when they happen,
  ship with migration guidance.
- **From v1.0 onward.** Strict [Semantic Versioning](https://semver.org).
  Breaking changes to the exported surface require a major-version bump — and,
  per Go module conventions, a new import path. Additive changes (new
  functions, new optional functional-options) ship as minor releases; bug
  fixes ship as patches.
- **What "exported surface" means here.** Function signatures, exported
  struct field sets, documented error types and their `Error()` messages, and
  the semantics of functional options. The `internal/gen` package carries no
  stability guarantee.
- **What is never a breaking change.** Updating the embedded OpenNGC data to
  a newer release, even if individual objects change classification or
  disappear upstream. Data freshness rides with the database version, not the
  Go module version.

### CLI (`cmd/ongc`)

The CLI's observable output is governed by the hard-invariant policy in
[section 3](#3-compatibility-posture). A few points specifically about
*stability over time*:

- Scripts and pipelines parsing CLI output can treat it as stable across
  go-openngc releases within a PyOngc major line.
- If a future PyOngc major release ships incompatible output, go-openngc
  will follow it — that is the whole point of the project — but the change
  will be announced, versioned, and documented up-front, never silently.
- Flag names and subcommand names are part of the invariant. PyOngc's CLI
  surface is the spec.

## 5. Zero external runtime dependencies

go-openngc depends on the Go standard library and nothing else at runtime.

You can verify this directly in [go.mod](../go.mod): there are no runtime
`require` directives. The only entry is a `tool` directive for
`golangci-lint`, which is a development-time tool used during `make lint` —
it is never linked into the binary, and its transitive dependencies never
ship with your build.

Rationale:

- **Reproducible builds.** No transitive version drift between `go build` runs
  separated in time.
- **Minimal supply-chain surface.** Nothing to audit beyond the Go toolchain
  and the OpenNGC CSV data.
- **License clarity.** Code is MIT; data is CC-BY-SA-4.0; there is no third
  category of transitive dependency license to reason about.

The only explicit exception is development tooling — golangci-lint,
govulncheck, REUSE — which is used during `make check` but never shipped.

## 6. Performance: moving parsing to build time

The initial implementation used `go:embed` to bundle the CSV files and parsed
them in `init()`. That `go:embed` approach was simple but had two problems:

1. **~300 ms cold-start penalty** on every CLI invocation and every `go test`
   run. Parsing the full catalog on every start is work the binary can do
   once, at build time.
2. **No recovery path for malformed data.** A bad row would panic `init`
   before user code ever ran. For a library, that is an unrecoverable
   failure mode imposed on callers.

The current approach — introduced in commit `d7648d1` — moves parsing to
build time. [`internal/gen/main.go`](../internal/gen/main.go) reads
`ongc/data/*.csv` and emits `ongc/catalog_generated.go` as ordinary Go source.
Runtime `init()` only builds the `byName`, `aliases`, and declination-band
indexes over the already-compiled slice. No file I/O on the hot path.

Trade-off: the compiled binary is slightly larger because the catalog is
baked in. This is bounded, predictable, and identical in spirit to
`unicode/tables.go` in the standard library and protoc-generated Go code —
there is precedent for this pattern.

The `make gen-check` target fails the build if the committed generated file
has drifted from the CSV sources, so the two cannot silently diverge.

## 7. On AI-assisted authorship

This project was translated from Python (PyOngc v1.2.0) to Go with heavy use
of [Claude](https://www.anthropic.com/claude), Anthropic's coding agent. The
author reviewed, directed, and maintains all changes. This is stated plainly
here because the project does not try to hide its provenance — the
[README's acknowledgments](../README.md#acknowledgments) already name Claude,
and this section explains the reasoning.

**Why it was done this way.** The Go ecosystem lacked a native OpenNGC
client. LLM-assisted translation was the productive path to close that gap
without abandoning the stdlib-only, license-clear design goals. The project
treats coding agents as productivity tools, not as authors of record.

**What it does not mean.** "AI-assisted" is not a synonym for "untested" or
"unreviewed." The next section enumerates the specific, runnable checks that
back the quality claim. Every claim in this document is either derivable from
the code or cited to a file or commit you can read.

**On the "AI slop" objection.** Some readers will decline AI-assisted code on
principle. This document will not try to argue them out of that choice. The
project's answer is to earn trust through verifiable quality — not through
rhetoric — and to make the evidence easy to find. If, after reading
[section 8](#8-quality-gates), the objection persists, that call is yours to
make — but the evidence will be there for you to judge on its own terms.

## 8. Quality gates

Each row below is a specific, runnable check — not a general claim. There is
no `.github/workflows` directory committed at present; the `Makefile` is the
single source of truth, and `make check` runs all gates together.

| Gate                  | Tool / method                                                                                           | Invocation        | Reference                                                                 |
| --------------------- | ------------------------------------------------------------------------------------------------------- | ----------------- | ------------------------------------------------------------------------- |
| Static analysis       | `go vet`                                                                                                | `go vet ./...`    | Go stdlib                                                                 |
| Linting               | golangci-lint (errcheck, govet, staticcheck, unused, ineffassign, gocritic, revive, exhaustive, goconst, misspell, unparam, nestif, gofumpt) | `make lint`       | [`.golangci.yml`](../.golangci.yml)                                       |
| Vulnerability scan    | govulncheck                                                                                             | `make vuln`       | [`Makefile`](../Makefile)                                                 |
| License compliance    | REUSE                                                                                                   | `make reuse`      | [`REUSE.toml`](../REUSE.toml), [`Makefile`](../Makefile)                  |
| Race detector         | Go race flag                                                                                            | `make test`       | [`Makefile`](../Makefile) (`go test -race ./...`)                         |
| Codegen drift guard   | Regenerate + diff                                                                                       | `make gen-check`  | [`Makefile`](../Makefile)                                                 |
| Composite             | All of the above                                                                                        | `make check`      | [`Makefile`](../Makefile)                                                 |

### Testing depth

- **Byte-exact golden-file CLI tests.** The `ongc` binary is executed under
  `exec.Command` and its stdout is diffed byte-for-byte against reference
  output in `testdata/golden/*.txt`. The golden files are regenerated from
  PyOngc itself via [`scripts/generate_testdata.py`](../scripts/generate_testdata.py).
  Commands are manifest-driven (`testdata/golden/commands.json`). See
  `TestGolden` in [`ongc/golden_test.go`](../ongc/golden_test.go).
- **Fixture-driven property parity.** `TestFixtureDriven` in
  [`ongc/ongc_test.go`](../ongc/ongc_test.go) validates every test object's
  RA, Dec, type, and Messier number against `testdata/objects.json`, which
  is generated from PyOngc v1.2.0.
- **Error-message byte parity.** `TestErrorMessageParity` in
  [`ongc/lookup_test.go`](../ongc/lookup_test.go) confirms that error
  strings match PyOngc character-for-character (for example, `Object named X
  not found in the database.`).
- **Full catalog.** The embedded catalog holds ~14,000 objects. Spot-check
  and fixture tests cover representative entries across the full database;
  there are 19 `_test.go` files spread across `ongc/` and `cmd/ongc/`.

If you want the shortest way to exercise all of this yourself:

```sh
make check
```

## 9. Maintenance discipline

- Go stable releases are adopted promptly; the module tracks the latest
  stable toolchain.
- The linter set, test harness, and Go version are revisited on each
  meaningful change. Best practices in the Go ecosystem evolve; the project
  tries to evolve with them rather than freeze at the state it shipped in.
- Project conventions, gotchas, and non-obvious constraints that compound
  across sessions are documented in [`CLAUDE.md`](../CLAUDE.md) at the repo
  root. That file is intended for both human contributors and AI coding
  agents working on the project.
