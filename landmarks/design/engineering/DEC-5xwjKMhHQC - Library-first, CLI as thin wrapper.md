---
author: wasserbar
id: DEC-5xwjKMhHQC
name: Library-first, CLI as thin wrapper
slug: library-first-cli-as-thin-wrapper
status: accepted
type: decision
---

## Context

cairn imports `d2lang/d2`'s packages (`d2lib`, `d2graph`, `d2svg`,
`d2dagrelayout`) directly to render diagrams into its own static binary —
it never shells out to a separate `d2` binary. If sill were CLI-first (logic
living in `cmd/sill`, extracted to a library later if needed), any consumer
wanting to embed it — cairn for `report coverage`, or anyone else — would
either have to shell out to a separate process or wait for a disruptive
extraction.

## Decision

Root package `sill` holds the real parsing/evaluation logic and has no
dependency on `cmd/`. `cmd/sill` is a thin cobra CLI that only calls into
the library and handles flags/output formatting. Any future consumer
imports `sill` as a Go module dependency, compiled into their own binary,
exactly like cairn does with D2 today.

## Consequences

- A consumer never needs the `sill` binary installed or on `PATH` — only
  the module as a build dependency.
- The CLI and any embedder share one evaluation engine with no drift risk.
- Slightly more upfront package-boundary discipline (no `cmd`-only
  globals/flags leaking into evaluation logic) — cheap to hold from day one,
  expensive to retrofit after the CLI and library are tangled.
