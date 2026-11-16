# sill

A headless, spec-compliant evaluator for Obsidian [Bases](https://obsidian.md/help/bases) (`.base` files) — a Go library first, a thin CLI second.

Obsidian's own CLI requires the desktop app running; there's no standalone way to evaluate a `.base` file from a script or CI pipeline. sill fills that gap, and adds one thing the spec doesn't have: a generic assertion/threshold layer, stored under its own sidecar key so a file using it stays valid and renders correctly in Obsidian.

Design docs live in `landmarks/` (managed with [cairn](https://github.com/enyhma/cairn)) — start at `landmarks/design/engineering/CTX-jOPDBK9Zwu - sill — headless .base evaluator.md`.
