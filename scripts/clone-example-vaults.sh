#!/usr/bin/env bash
# Shallow-clones the real-world vaults listed in vaults.txt, each at its
# pinned commit, into one subdirectory per repo under $1.
#
# A plain `git clone --depth 1` only ever gets the CURRENT tip of the
# default branch — it can't fetch an older pinned commit once the
# upstream repo has moved on. Instead this inits an empty repo and fetches
# exactly the pinned SHA at depth 1, which GitHub supports for arbitrary
# reachable commits on public repos.
set -euo pipefail

dest="${1:?usage: clone-example-vaults.sh <dest-dir>}"
manifest="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/vaults.txt"
mkdir -p "$dest"

while read -r repo sha; do
  case "$repo" in '#'*|'') continue ;; esac
  name="${repo##*/}"
  target="$dest/$name"
  if [ -d "$target/.git" ]; then
    echo "skip $repo @ $sha (already present)"
    continue
  fi
  echo "fetching $repo @ $sha"
  mkdir -p "$target"
  git -C "$target" init --quiet
  git -C "$target" remote add origin "https://github.com/$repo.git"
  git -C "$target" fetch --quiet --depth 1 origin "$sha"
  git -C "$target" checkout --quiet "$sha"
done <"$manifest"
