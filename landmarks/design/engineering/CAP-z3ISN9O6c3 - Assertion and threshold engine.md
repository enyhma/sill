---
author: wasserbar
id: CAP-z3ISN9O6c3
links:
    refines:
        - CTX-jOPDBK9Zwu
name: Assertion and threshold engine
owner: wasserbar
slug: assertion-and-threshold-engine
status: approved
type: cap
---

## Description

sill's one real spec extension: given a filter (the population) and a
condition (what must hold), compute how many/what percentage of matching
notes satisfy it, and compare against a threshold. Lives under its own
top-level key per [[DEC-eNLURTFi6w]], never inside `filters:`/`views:`.

## Actor

[[CAP-KmjBkVzY5Y]] (`sill lint` reads this to produce pass/fail + exit code).

## Rationale

This is the piece vanilla Bases has no primitive for, and the actual reason
sill is more than a read-only query tool — it's what would make a
coverage-style check ("every note matching X also satisfies Y") expressible
without Obsidian's GUI, for any vault, not just cairn's.

## Acceptance Criteria

- [ ] A file using this extension still parses and renders correctly as a
      plain Base when opened in Obsidian (the sidecar key is genuinely
      ignorable)
- [ ] Threshold evaluation produces a documented, stable exit code (0 =
      passed, non-zero = below threshold) — unlike Obsidian's own CLI, whose
      docs don't specify exit-code behavior
- [ ] Count and percentage modes both supported
