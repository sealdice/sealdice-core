#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
# shellcheck source=/dev/null
source "$ROOT_DIR/scripts/randomness/run_rddetector_reports.sh"

assert_eq() {
  local got="$1"
  local want="$2"
  local label="$3"
  if [[ "$got" != "$want" ]]; then
    echo "assertion failed: $label got=$got want=$want" >&2
    exit 1
  fi
}

resolve_profile strict-0005
assert_eq "$SAMPLES" "1000" "strict-0005 samples"
assert_eq "$BITS" "1000000" "strict-0005 bits"

resolve_profile strict-0005-20k
assert_eq "$SAMPLES" "1000" "strict-0005-20k samples"
assert_eq "$BITS" "20000" "strict-0005-20k bits"

resolve_profile strict-0005-100m
assert_eq "$SAMPLES" "1000" "strict-0005-100m samples"
assert_eq "$BITS" "100000000" "strict-0005-100m bits"

resolve_profile poweron
assert_eq "$SAMPLES" "20" "poweron samples"
assert_eq "$BITS" "1000000" "poweron bits"

echo "run_rddetector_reports.sh profile resolution ok"
