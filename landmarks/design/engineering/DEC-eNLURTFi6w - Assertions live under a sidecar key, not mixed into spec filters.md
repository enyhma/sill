---
author: wasserbar
id: DEC-eNLURTFi6w
name: Assertions live under a sidecar key, not mixed into spec filters
slug: assertions-live-under-a-sidecar-key-not-mixed
status: accepted
type: decision
---

## Context

Vanilla Bases has no primitive for "count/percentage of notes matching
filter X also satisfying Y, fail below a threshold" — that's sill's one
real addition to the spec. Obsidian's YAML parser only reads the top-level
keys it knows about (`filters`, `formulas`, `properties`, `summaries`,
`views`); an unrecognized sibling key is ignored, not rejected.

## Decision

sill's assertion/threshold extension is written under its own top-level
key (e.g. `sillAssert:`), never inside `filters:`/`formulas:`/`views:`. A
`.base` file using it stays fully spec-compliant and renders correctly if
opened in the Obsidian app — the app simply never looks at the extra key.

## Consequences

- One file can serve two audiences: Obsidian renders the GUI view from the
  spec-compliant portion; `sill lint` reads the sidecar key for CI/scripted
  checks. Neither consumer needs to know about the other's section.
- sill's extension vocabulary can evolve without ever risking an existing
  `.base` file's Obsidian-rendered view.
- If Obsidian's own spec ever adds a conflicting key with the same name,
  sill's key needs renaming — low-probability, cheap to fix, not worth
  pre-emptively namespacing further today.
