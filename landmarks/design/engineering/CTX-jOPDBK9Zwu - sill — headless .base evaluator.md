---
author: wasserbar
id: CTX-jOPDBK9Zwu
name: sill — headless .base evaluator
owner: wasserbar
slug: sill-headless-base-evaluator
status: approved
type: cntxt
---

## Context

Obsidian Bases (`.base` files) let a vault define filtered/sorted views over
notes' frontmatter, but the only thing that evaluates them today is the
Obsidian desktop app itself — its CLI (`base:query` etc.) requires the GUI
app running, and "Obsidian Headless" is an unrelated Sync/Publish product.
There is no standalone, scriptable, exit-code-driven way to evaluate a
`.base` file. sill fills that gap: a Go library (importable the way
`d2lang/d2` is) plus a thin CLI on top.

sill was spun out of [[cairn]] while exploring whether cairn's own `report`
command (coverage/duplicates/matrix) could be replaced by generated `.base`
views. It can't — two of the three reports (matrix's type×type cross-tab,
duplicates' similarity scoring) have no per-row-filter shape at all, and the
third (coverage) is already simpler as direct Go than it would be re-expressed
through a generic filter DSL. sill is justified on its own merits — the
ecosystem gap — not as a dependency cairn needs.

## Goals

- Parse `.base` YAML into the documented spec: filters (`and`/`or`/`not`,
  comparisons, `&&`/`||`/`!`), formulas, properties, summaries, views.
- Evaluate filters/formulas against a set of notes (frontmatter + file
  metadata) with the complete documented function library, across every
  type namespace Obsidian defines (global, any, date, string, number,
  list, link, file, object, regexp) — see [[CAP-E7loYprwhZ]] for the full
  catalog.
- A generic assertion/threshold layer, additive to the spec: "count/percentage
  of notes matching filter X that also satisfy condition Y, exit non-zero
  below a threshold." Lives under its own top-level key so a file using it
  stays valid and renders correctly if opened in Obsidian (see
  [[DEC-eNLURTFi6w]]).
- Machine-readable output (`json`/`csv`/`table`) and real exit codes, so it's
  usable from CI and scripts without Obsidian installed.
- Stay importable as a library first (see [[DEC-5xwjKMhHQC]]), CLI second.

## Non Goals

- View rendering (table/cards/kanban/map) — that's Obsidian's job; sill only
  needs the filter/formula result set, not presentation.
- Any cairn-specific (or any single consumer's) functions or schema
  awareness — see [[DEC-RPUG8095er]]. sill stays a general-purpose vault
  tool; cairn is, at most, one consumer of it.
- Full fidelity with every Obsidian plugin-dependent feature (e.g. the Map
  view's lat/long requirements) — out of scope until a real need appears.
