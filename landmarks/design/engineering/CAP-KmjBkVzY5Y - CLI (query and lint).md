---
author: wasserbar
id: CAP-KmjBkVzY5Y
links:
    refines:
        - CTX-jOPDBK9Zwu
name: CLI (query and lint)
owner: wasserbar
slug: cli-query-and-lint
status: approved
type: cap
---

## Description

The `sill` binary: `sill query <file.base> --format table|grid|json|toon|csv`
runs a view's filters against a vault and returns matching notes; `sill
lint <file.base>` evaluates the assertion/threshold sidecar and exits
accordingly. A thin wrapper per [[DEC-5xwjKMhHQC]] — no logic lives here
that isn't also reachable as a library call.

`table` (the TTY default) and `grid` are sill's two human-facing options,
not one: `table` is full detail (every property, nesting shown through
indentation — see hearth's `RenderBlock`), `grid` is a compact
column-set overview (nesting flattened inline) selected by `--fields` or
a view's own `order:`. `json`/`toon`/`csv` are the machine-facing options
— `json` nesting natively, `toon`/`csv` flattening like `grid` does.

## Actor

Anyone scripting against a vault from a shell or CI pipeline without
Obsidian installed.

## Rationale

This is the only piece end users interact with directly; everything else
is plumbing in service of making this command correct and scriptable.

## Acceptance Criteria

- [ ] `query` output formats are stable and documented
      (table/grid/json/toon/csv)
- [ ] A column reference (`--fields` or a view's `order:`) resolves
      through the same evaluator filters use — `note.`/`file.`/
      `formula.`/bare forms all work, not just bare frontmatter
      properties
- [ ] `lint` exit codes are stable and documented
- [ ] Zero business logic in `cmd/sill` that isn't a direct call into the
      `sill` package
