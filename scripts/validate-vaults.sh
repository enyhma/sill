#!/usr/bin/env bash
# Validates the sill binary against every vault cloned by
# clone-example-vaults.sh: a trivial query must load at least one note
# per non-empty vault, and every real .base file found inside a vault
# must query without an "Error:" line. Per-note parse warnings (e.g. an
# Obsidian template file using Templater/Dataview placeholder syntax
# that isn't valid YAML) are expected and not a failure — see
# hearth's DEC-LSh8yPXvpn and sill's DEC-NqI23Hxjzx.
set -uo pipefail

vaults_dir="${1:?usage: validate-vaults.sh <vaults-dir> <sill-binary>}"
sill_bin="${2:?usage: validate-vaults.sh <vaults-dir> <sill-binary>}"

fail=0
smoke_base="$(mktemp)"
printf 'filters: '"'"'file.ext == "md"'"'"'\n' >"$smoke_base"

for vault in "$vaults_dir"/*/; do
  name="$(basename "$vault")"
  total="$(find "$vault" -name '*.md' | wc -l | tr -d ' ')"

  matched="$("$sill_bin" query "$smoke_base" --vault "$vault" --format json 2>/dev/null | grep -c '"' || true)"
  echo "== $name: $matched/$total notes loaded via a trivial query =="
  if [ "$total" -gt 0 ] && [ "$matched" -eq 0 ]; then
    echo "FAIL: $name loaded zero notes out of $total"
    fail=1
  fi

  while IFS= read -r -d '' base; do
    result="$("$sill_bin" query "$base" --vault "$vault" --format table 2>&1)"
    if echo "$result" | grep -q '^Error:'; then
      echo "FAIL: $name/$(basename "$base"): $(echo "$result" | grep '^Error:')"
      fail=1
    else
      echo "ok   $name/$(basename "$base")"
    fi
  done < <(find "$vault" -name '*.base' -print0)
done

rm -f "$smoke_base"
exit "$fail"
