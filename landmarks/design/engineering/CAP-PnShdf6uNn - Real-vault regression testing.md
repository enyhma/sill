---
author: wasserbar
id: CAP-PnShdf6uNn
links:
    refines:
        - CTX-jOPDBK9Zwu
name: Real-vault regression testing
slug: real-vault-regression-testing
status: approved
type: cap
---

## Description

CI runs the built `sill` binary against real third-party Obsidian
vaults — not synthetic fixtures — via `scripts/clone-example-vaults.sh`
(pinned-SHA shallow fetch of each vault in `scripts/vaults.txt`) and
`scripts/validate-vaults.sh` (a trivial query against every vault, plus
every real `.base` file found inside one, failing on any unhandled
`Error:`). `.github/workflows/real-vaults.yml` runs it on evaluator-
relevant code changes, weekly, and on demand.

## Actor

Anyone changing `eval.go`/`functions.go`/`vault.go`/the parser — this is
what catches a regression a synthetic unit test wouldn't, before it ships.

## Rationale

Every bug this caught in practice was invisible to the existing unit
test suite until a real vault exercised it: `list.filter`'s actual
lambda-binding identifier (confirmed from kepano-obsidian's real
Related.base, not guessed), "this" silently matching every single note
in a vault instead of erroring, one note's `nil.contains(...)` aborting
an entire query. Hand-cloning vaults and re-running a CLI by hand doesn't
scale as a standing regression guard — this turns that exploration into
an automated, repeatable one. The vault list itself came from an
Obsidian forum thread of real example vaults, not vaults built to
exercise sill.

## Acceptance Criteria

- [ ] `validate-vaults.sh` exits non-zero if any vault loads zero notes
      out of a non-zero total, or if any real `.base` file inside a
      vault produces an `Error:` line
- [ ] Vaults are pinned to a specific commit (`scripts/vaults.txt`), not
      a moving branch — an upstream vault's own future change must never
      be what breaks this suite
- [ ] Per-note parse warnings (a template file with Templater/Dataview
      placeholder syntax that isn't valid YAML) are expected, not a
      failure — see hearth's DEC-LSh8yPXvpn and DEC-NqI23Hxjzx
- [ ] Runs in CI without either repo being checked out at a specific
      local folder name — the hearth module replace is rewritten at
      CI-build time, not baked into the committed go.mod
