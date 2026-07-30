#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
ROUNDS="${ROUNDS:-1}"
MODES="${MODES:-pcg,crypto,nist,gm}"

resolve_profile() {
  local profile="$1"

  PROFILE_NOTE=""
  case "$profile" in
    poweron)
      SAMPLES=20
      BITS=1000000
      PROFILE_NOTE="工程级上电复核档"
      ;;
    factory)
      SAMPLES=50
      BITS=1000000
      PROFILE_NOTE="工程级出厂复核档"
      ;;
    quick)
      SAMPLES=20
      BITS=20000
      PROFILE_NOTE="快速流程验证档"
      ;;
    strict-0005|strict-0005-1m|gmt0005|gmt0005-1m)
      SAMPLES=1000
      BITS=1000000
      PROFILE_NOTE="严格 GM/T 0005-2021 样本集检测档（1000 x 1,000,000 bit）"
      ;;
    strict-0005-20k|gmt0005-20k)
      SAMPLES=1000
      BITS=20000
      PROFILE_NOTE="严格 GM/T 0005-2021 样本集检测档（1000 x 20,000 bit）"
      ;;
    strict-0005-100m|gmt0005-100m)
      SAMPLES=1000
      BITS=100000000
      PROFILE_NOTE="严格 GM/T 0005-2021 样本集检测档（1000 x 100,000,000 bit）"
      ;;
    *)
      echo "unsupported profile: $profile" >&2
      echo "supported profiles: quick, poweron, factory, strict-0005, strict-0005-20k, strict-0005-1m, strict-0005-100m" >&2
      return 1
      ;;
  esac

  if [[ -n "${SAMPLES_OVERRIDE:-}" ]]; then
    SAMPLES="$SAMPLES_OVERRIDE"
  fi
  if [[ -n "${BITS_OVERRIDE:-}" ]]; then
    BITS="$BITS_OVERRIDE"
  fi

  return 0
}

main() {
  local profile="${1:-poweron}"
  local tarball_path="${2:-$ROOT_DIR/docs/rddetector_linux_amd64.tar.gz}"
  local out_dir="${3:-$ROOT_DIR/docs/randomness-reports/$profile}"

  resolve_profile "$profile"

  if [[ ! -f "$tarball_path" ]]; then
    echo "rddetector archive not found: $tarball_path" >&2
    return 1
  fi

  local work_dir="$ROOT_DIR/temp/randomness/$profile"
  local tool_dir="$work_dir/tool"
  local sample_root="$work_dir/samples"

  mkdir -p "$tool_dir" "$sample_root" "$out_dir"
  tar -xzf "$tarball_path" -C "$tool_dir"

  local rddetector_bin
  rddetector_bin="$(find "$tool_dir" -maxdepth 2 -type f -name 'rddetector*' | head -n 1)"
  if [[ -z "$rddetector_bin" ]]; then
    echo "failed to locate rddetector binary after extracting $tarball_path" >&2
    return 1
  fi
  chmod +x "$rddetector_bin"

  echo "profile=$profile rounds=$ROUNDS samples=$SAMPLES bits=$BITS modes=$MODES"
  if [[ -n "$PROFILE_NOTE" ]]; then
    echo "profile_note=$PROFILE_NOTE"
  fi
  echo "using rddetector: $rddetector_bin"

  local round raw_mode mode round_sample_dir round_out_dir mode_sample_dir mode_report mode_analysis
  for round in $(seq 1 "$ROUNDS"); do
    round_sample_dir="$sample_root/round-$round"
    round_out_dir="$out_dir/round-$round"
    mkdir -p "$round_sample_dir" "$round_out_dir"

    echo "== generating samples for round $round =="
    SEALDICE_RANDOMNESS_GENERATE=1 \
    SEALDICE_RANDOMNESS_OUT_DIR="$round_sample_dir" \
    SEALDICE_RANDOMNESS_MODES="$MODES" \
    SEALDICE_RANDOMNESS_SAMPLES="$SAMPLES" \
    SEALDICE_RANDOMNESS_BITS="$BITS" \
    go test ./dice -run TestGenerateRandomnessSamples -count=1

    IFS=',' read -ra MODE_ARR <<< "$MODES"
    for raw_mode in "${MODE_ARR[@]}"; do
      mode="$(echo "$raw_mode" | xargs)"
      [[ -z "$mode" ]] && continue

      mode_sample_dir="$round_sample_dir/$mode"
      mode_report="$round_out_dir/${mode}-report.csv"
      mode_analysis="$round_out_dir/${mode}-analysis.csv"

      echo "== rddetector mode=$mode round=$round =="
      "$rddetector_bin" \
        -i "$mode_sample_dir" \
        -o "$mode_report" \
        -a "$mode_analysis" \
        -f csv
    done
  done

  go run ./scripts/randomness/summarize_reports.go \
    -in "$out_dir" \
    -out "$out_dir/summary.md" \
    -profile "$profile" \
    -rounds "$ROUNDS" \
    -samples "$SAMPLES" \
    -bits "$BITS"

  echo "reports written to: $out_dir"
}

if [[ "${BASH_SOURCE[0]}" == "$0" ]]; then
  main "$@"
fi
