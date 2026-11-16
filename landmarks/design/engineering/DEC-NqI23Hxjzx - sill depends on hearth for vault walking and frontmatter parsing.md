---
author: wasserbar
id: DEC-NqI23Hxjzx
links:
    related_to:
        - CAP-KmjBkVzY5Y
name: sill depends on hearth for vault walking and frontmatter parsing
slug: sill-depends-on-hearth-for-vault-walking-and
status: accepted
type: decision
---

## Context

[[CAP-KmjBkVzY5Y]]'s CLI needs to turn a directory of `.md` files into
the `EvalContext` values the evaluator ([[CAP-MB8sZOUUno]]) runs against
— frontmatter split from body, read concurrently across a vault. hearth
already built exactly this ([[hearth]]'s `Parse`/`ParseDir`), and
hearth's own `CAP-siFpsbC68G` (Backlink graph) names sill's
`link.linksTo`/`file.hasLink` as its actor — the two were designed to
slot together, but nothing had committed sill to actually depending on
hearth, and [[DEC-RPUG8095er]] (no cairn-aware functions) could be
misread as "sill depends on nothing," which it doesn't claim.

The alternative was sill re-implementing its own directory walk and
frontmatter split, duplicating code hearth already has tested.

## Decision

`cmd/sill`'s vault-loading layer imports hearth (`Parse`/`ParseDir`) for
frontmatter and body, and adapts a `hearth.Note` into an `EvalContext` —
deriving `file.*` fields (name, path, folder, ext, size, mtime) from the
OS and the note's path. hearth has no concept of Obsidian's `#tag`/
`tags:` property, so sill does its own tag extraction on top of a
`hearth.Note`'s frontmatter and body.

This is not a contradiction of [[DEC-RPUG8095er]]: that decision is about
sill never growing a function that requires knowledge of a *specific
tool's* schema (cairn's types, relations). hearth is schema-less by
design — depending on it adds no such coupling.

## Consequences

- sill's go.mod gains a `replace` pointing at hearth's local path (no
  published hearth module yet) — fine for two sibling repos developed
  together, revisit once either publishes independently.
- cairn, hearth, and sill now form a real dependency chain (sill → hearth)
  for the first time outside of the design docs' cross-references.
- If hearth's `Note`/`ParseDir` signatures change, sill's vault-loading
  adapter is the one call site that needs updating — contained, since
  nothing else in sill touches hearth directly.
