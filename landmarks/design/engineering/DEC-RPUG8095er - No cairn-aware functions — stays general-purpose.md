---
author: wasserbar
id: DEC-RPUG8095er
name: No cairn-aware functions — stays general-purpose
slug: no-cairn-aware-functions-stays-general-purpose
status: accepted
type: decision
---

## Context

Of cairn's three `report` subcommands, only `coverage` is even shaped like
something a generic assertion engine could express ("does some other note's
list-property reference this note" — already native via
`list.contains`). `matrix` needs a typed relation-schema model (cairn's
`relations.yaml`) with no vault-level equivalent; `duplicates` needs
similarity/embedding scoring. Giving sill cairn-specific functions to reach
either of those would make it cairn's library wearing a "standalone tool"
costume, not an actually general one.

## Decision

sill's function library only ever grows generic, vault-shaped primitives
(property/tag/folder/date/list checks). No function may require knowledge
of a specific tool's schema (cairn's types, relations, or coverage rules) to
be meaningful. If a consumer needs tool-specific logic, that logic stays in
the consumer, built on sill's generic primitives — not pushed into sill.

## Consequences

- cairn's `report matrix` and `report duplicates` stay in cairn permanently
  — they are not migration candidates, regardless of how sill evolves.
- `report coverage` is a *possible* future migration once sill exists and
  is proven, not a reason to build sill — see [[CTX-jOPDBK9Zwu]]'s Context.
- sill's roadmap is driven by what's generically useful to any vault, which
  is the actual value proposition over depending on the live Obsidian app.
