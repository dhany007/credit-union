#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."
version="${VERSION:-}"
if [[ ! "$version" =~ ^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
  echo 'Usage: make release VERSION=v0.1.0 (new stable vMAJOR.MINOR.PATCH tag)' >&2
  exit 2
fi
command -v gh >/dev/null || { echo 'Install GitHub CLI (gh) and run gh auth login first.' >&2; exit 1; }
gh auth status >/dev/null 2>&1 || { echo 'Run gh auth login with repository write access first.' >&2; exit 1; }
repo="$(gh repo view --json nameWithOwner --jq .nameWithOwner)"
main_sha="$(gh api "repos/$repo/commits/main" --jq .sha)"
[[ "$main_sha" =~ ^[0-9a-f]{40}$ ]] || { echo 'Could not resolve remote main.' >&2; exit 1; }
refs="$(gh api "repos/$repo/git/matching-refs/tags/$version" --jq '.[].ref')"
if grep -Fxq "refs/tags/$version" <<< "$refs"; then
  echo "Tag $version already exists; choose a new VERSION." >&2
  exit 1
fi
printf 'Releasing %s at remote main %s in %s\n' "$version" "$main_sha" "$repo"
gh release create "$version" --repo "$repo" --target "$main_sha" \
  --title "$version" --generate-notes
