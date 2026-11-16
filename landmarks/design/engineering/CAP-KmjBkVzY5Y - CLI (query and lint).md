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

The `sill` binary: `sill query <file.base> --format json|csv|table` runs a
view's filters against a vault and returns matching notes; `sill lint
<file.base>` evaluates the assertion/threshold sidecar and exits
accordingly. A thin wrapper per [[DEC-5xwjKMhHQC]] — no logic lives here
that isn't also reachable as a library call.

## Actor

Anyone scripting against a vault from a shell or CI pipeline without
Obsidian installed.

## Rationale

This is the only piece end users interact with directly; everything else
is plumbing in service of making this command correct and scriptable.

## Acceptance Criteria

- [ ] `query` output formats are stable and documented (json/csv/table)
- [ ] `lint` exit codes are stable and documented
- [ ] Zero business logic in `cmd/sill` that isn't a direct call into the
      `sill` package
