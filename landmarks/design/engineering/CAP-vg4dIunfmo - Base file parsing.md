---
author: wasserbar
id: CAP-vg4dIunfmo
links:
    refines:
        - CTX-jOPDBK9Zwu
name: Base file parsing
owner: wasserbar
slug: base-file-parsing
status: approved
type: cap
---

## Description

Parse a `.base` file's YAML into typed Go structs covering the documented
top-level keys: `filters`, `formulas`, `properties`, `summaries`, `views`.
Preserve unrecognized top-level keys (e.g. a future `sillAssert:`) rather
than erroring or dropping them, so round-tripping a file never loses data.

## Actor

Anything embedding sill as a library (the `sill` CLI itself, cairn's
`report coverage`, or any other Go program).

## Rationale

Every other capability depends on having a correctly parsed, typed
representation of the file first.

## Acceptance Criteria

- [ ] Parses every example in Obsidian's own Bases syntax docs without error
- [ ] Round-trips: parse → re-marshal produces semantically equivalent YAML
- [ ] Unknown top-level keys are preserved, not dropped
- [ ] Malformed YAML and unknown *required* structure produce a clear error,
      not a silent partial parse
