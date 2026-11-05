#!/usr/bin/env bash
# Merge unit-test and operator e2e-test coverage profiles into a single
# library-only coverage report, and generate the text, HTML, and badge outputs.
#
# Usage: merge-coverage.sh [unit-profile] [e2e-profile] [outdir]
# Defaults match the layout produced by the CI test job:
#   unit-profile: coverage/unit_coverage.out   (root module, -covermode=set)
#   e2e-profile:  tests/operator/coverage.out  (operator e2e, -covermode=set)
#   outdir:       coverage
set -euo pipefail

UNIT_PROFILE=${1:-coverage/unit_coverage.out}
E2E_PROFILE=${2:-tests/operator/coverage.out}
OUT_DIR=${3:-coverage}

for profile in "$UNIT_PROFILE" "$E2E_PROFILE"; do
  if [[ ! -s "$profile" ]]; then
    echo "empty or missing coverage profile: $profile" >&2
    exit 1
  fi
done

mkdir -p "$OUT_DIR"

awk '
  BEGIN { mode = "" }
  FNR == 1 {
    if ($1 != "mode:") {
      printf "missing coverage mode header in %s\n", FILENAME > "/dev/stderr"
      exit 1
    }
    if (mode == "") {
      mode = $2
    } else if ($2 != mode) {
      printf "coverage mode mismatch: %s vs %s in %s\n", mode, $2, FILENAME > "/dev/stderr"
      exit 1
    }
    next
  }
  {
    key = $1 " " $2
    count = $3 + 0
    if (!(key in seen)) {
      seen[key] = 1
      order[++n] = key
      counts[key] = count
    } else if (mode == "set") {
      if (count > counts[key]) {
        counts[key] = count
      }
    } else {
      counts[key] += count
    }
  }
  END {
    print "mode: " mode
    for (i = 1; i <= n; i++) {
      print order[i] " " counts[order[i]]
    }
  }
' "$UNIT_PROFILE" "$E2E_PROFILE" > "$OUT_DIR/merged_coverage.out"

# Keep only the header and statements belonging to this library.
# A missing match is an error: it means the profiles cover nothing from the library.
{
  head -n 1 "$OUT_DIR/merged_coverage.out"
  grep "github.com/u-ctf/controller-fwk" "$OUT_DIR/merged_coverage.out"
} > "$OUT_DIR/library_coverage.out"

go tool cover -func="$OUT_DIR/library_coverage.out" -o "$OUT_DIR/coverage.txt"
go tool cover -html="$OUT_DIR/library_coverage.out" -o "$OUT_DIR/coverage.html"

PERCENTAGE=$(awk '/^total:/ {gsub(/%/, "", $3); print $3}' "$OUT_DIR/coverage.txt")
COLOR="red"
if awk -v p="$PERCENTAGE" 'BEGIN { exit !(p >= 90) }'; then
  COLOR="green"
elif awk -v p="$PERCENTAGE" 'BEGIN { exit !(p >= 75) }'; then
  COLOR="yellow"
elif awk -v p="$PERCENTAGE" 'BEGIN { exit !(p >= 50) }'; then
  COLOR="orange"
fi

printf '{"schemaVersion": 1, "label": "coverage", "message": "%s%%", "color": "%s"}\n' \
  "$PERCENTAGE" "$COLOR" > "$OUT_DIR/coverage_badge.json"
