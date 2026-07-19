#!/usr/bin/env bash
# CLAUDE.md drift guard. Fails CI when CLAUDE.md no longer matches the repo:
#   1. every path listed in the Directory Map must exist on disk
#   2. every internal/ package, .github/workflows/*.yml, and docs/*.md
#      must appear in the Directory Map
#   3. the documented API route list must exactly match internal/api/router.go
#
# Catches structural drift only; semantic drift (stale descriptions,
# conventions) is covered by the periodic accuracy-pass review.
set -euo pipefail
cd "$(dirname "$0")/.."

fail=0
err() {
  echo "::error::CLAUDE.md drift: $*"
  fail=1
}

# ---- extract the fenced Directory Map block --------------------------------
map=$(awk '
  /^## Directory Map$/ { s = 1; next }
  s && /^```/          { if (f) exit; f = 1; next }
  f                    { print }
' CLAUDE.md)

if [ -z "$map" ]; then
  err "could not find the fenced Directory Map block"
  exit 1
fi

# ---- 1. every listed path must exist ---------------------------------------
# Top-level lines are repo-relative; two-space-indented lines are relative to
# the most recent top-level directory (e.g. files under internal/repository/).
parent=""
while IFS= read -r line; do
  [ -z "${line//[[:space:]]/}" ] && continue
  tok=$(awk '{print $1}' <<<"$line")
  if [[ $line == "  "* ]]; then
    path="$parent$tok"
  else
    path="$tok"
    if [[ $tok == */ ]]; then parent="$tok"; else parent=""; fi
  fi
  [ -e "${path%/}" ] || err "Directory Map lists '$path' but it does not exist"
done <<<"$map"

# ---- 2. required paths must be documented ----------------------------------
for d in internal/*/; do
  grep -q "^$d" <<<"$map" || err "package '$d' is not in the Directory Map"
done
for f in .github/workflows/*.yml; do
  grep -q "^  $(basename "$f")" <<<"$map" || err "workflow '$f' is not in the Directory Map"
done
for f in docs/*.md; do
  grep -q "^$f" <<<"$map" || err "doc '$f' is not in the Directory Map"
done

# ---- 3. documented API routes must match router.go -------------------------
# Path params are normalized to {*} so {id} vs {walletID} naming differences
# don't count as drift. The SPA catch-all (r.Get("/*", ...)) is excluded.
code_routes=$(grep -oE 'r\.(Get|Post|Patch|Delete|Put)\("[^"]+"' internal/api/router.go \
  | sed -E 's/^r\.//; s/\("/ /; s/"$//' \
  | awk '{ m = toupper($1); p = $2; gsub(/\{[^}]*\}/, "{*}", p)
           if (p != "/*") print m " /api/v1" p }' \
  | sort -u)

doc_routes=$(grep -oE '`(GET|POST|PATCH|DELETE|PUT)(/(GET|POST|PATCH|DELETE|PUT))* /api/v1[^`]*`' CLAUDE.md \
  | tr -d '`' \
  | awk '{ n = split($1, ms, "/"); p = $2; gsub(/\{[^}]*\}/, "{*}", p)
           for (i = 1; i <= n; i++) print ms[i] " " p }' \
  | sort -u)

missing_in_docs=$(comm -23 <(echo "$code_routes") <(echo "$doc_routes"))
missing_in_code=$(comm -13 <(echo "$code_routes") <(echo "$doc_routes"))
[ -z "$missing_in_docs" ] || err "routes in router.go but not in CLAUDE.md: $(tr '\n' ';' <<<"$missing_in_docs")"
[ -z "$missing_in_code" ] || err "routes in CLAUDE.md but not in router.go: $(tr '\n' ';' <<<"$missing_in_code")"

if [ "$fail" -eq 0 ]; then
  echo "CLAUDE.md is in sync with the repository."
fi
exit "$fail"
