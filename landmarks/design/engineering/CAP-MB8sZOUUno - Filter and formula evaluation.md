---
author: wasserbar
id: CAP-MB8sZOUUno
links:
    refines:
        - CTX-jOPDBK9Zwu
name: Filter and formula evaluation
owner: wasserbar
slug: filter-and-formula-evaluation
status: approved
type: cap
---

## Description

Evaluate a parsed `filters:`/`formulas:` tree against one note (frontmatter
properties + file metadata): `and`/`or`/`not` recursion, arithmetic
(`+`, `-`, `*`, `/`, `%`), comparisons (`==`, `!=`, `>`, `<`, `>=`, `<=`),
boolean operators (`&&`, `||`, `!`), and property references — `note.x`,
bracket form `note["x"]`, bare shorthand `x`, `file.x` (`name`, `path`,
`mtime`, etc. — see [[CAP-E7loYprwhZ]]'s File fields), and `formula.x`.

## Actor

[[CAP-vg4dIunfmo]] (consumes the parsed tree); [[CAP-KmjBkVzY5Y]] and
[[CAP-z3ISN9O6c3]] (consume the evaluation result).

## Rationale

This is the actual engine — parsing without evaluation is just a YAML
reader; the filter/formula semantics are where spec-fidelity actually
matters, and where Obsidian's own docs flag real sharp edges: date
arithmetic (`date + "1M"`, `date - "2h"`, and `date - date` producing a
duration — unit suffixes accept short and long forms, `d`/`day`/`days`,
`h`/`hour`/`hours`, etc. — not a `.days` accessor, which doesn't exist in
the spec); and two-layer quoting, since a formula is itself a quoted YAML
string that may need an embedded string literal of the *other* quote
character (`'if(price, price.toFixed(2) + " dollars")'`).

## Acceptance Criteria

- [ ] All arithmetic, comparison, and logical operators match Obsidian's
      documented semantics
- [ ] `note.`/`file.`/`formula.` property resolution matches the spec —
      dot form, bracket form, and bare shorthand — including nested
      frontmatter paths (e.g. `links.refines`)
- [ ] Referencing an undefined formula fails loudly (sill's one deliberate
      divergence from Obsidian's documented silent-failure behavior — see
      [[CAP-vg4dIunfmo]]'s acceptance criteria on clear errors)
- [ ] Date arithmetic (`date ± duration-string`, `date - date` → duration,
      `duration()`, `.format()`) matches documented behavior, including
      unit-suffix parsing (`y`/`M`/`d`/`w`/`h`/`m`/`s` and their long forms)
