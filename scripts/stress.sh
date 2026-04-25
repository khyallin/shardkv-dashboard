#!/usr/bin/env bash
set -euo pipefail

BASE="${BASE:-http://127.0.0.1:8080}"
OUT="${OUT:-tmp/stress_results.jsonl}"
TARGET_GID="${TARGET_GID:-1}"
DURATION="${DURATION:-30}"
REPEAT="${REPEAT:-3}"
PROFILE="${PROFILE:-full}"
STRESS_KEY_PREFIX="${STRESS_KEY_PREFIX:-stress-g${TARGET_GID}-}"

if [[ "$PROFILE" == "quick" ]]; then
  CONCURRENCY_CASES=(1 4 16)
  READ_RATIO_CASES=(0 50 100)
  VALUE_SIZE_CASES=(64 1024 4096)
  MIX_READ_RATIO_CASES=(0 100)
  MIX_VALUE_SIZE_CASES=(64 4096)
else
  CONCURRENCY_CASES=(1 2 4 8 16 32 64)
  READ_RATIO_CASES=(0 25 50 75 100)
  VALUE_SIZE_CASES=(16 64 256 1024 4096)
  MIX_READ_RATIO_CASES=(0 50 100)
  MIX_VALUE_SIZE_CASES=(64 1024 4096)
fi

CONCURRENCY_CASE_COUNT=${#CONCURRENCY_CASES[@]}
READ_RATIO_CASE_COUNT=${#READ_RATIO_CASES[@]}
VALUE_SIZE_CASE_COUNT=${#VALUE_SIZE_CASES[@]}
MIX_CASE_COUNT=$(( ${#MIX_READ_RATIO_CASES[@]} * ${#MIX_VALUE_SIZE_CASES[@]} ))
TOTAL_CASES=$(( (CONCURRENCY_CASE_COUNT + READ_RATIO_CASE_COUNT + VALUE_SIZE_CASE_COUNT + MIX_CASE_COUNT) * REPEAT ))

: > "$OUT"

CASE_INDEX=0

format_elapsed() {
  local total_seconds="$1"
  printf '%02d:%02d:%02d' $((total_seconds / 3600)) $(((total_seconds % 3600) / 60)) $((total_seconds % 60))
}

run_case() {
  local scenario="$1" rep="$2" c="$3" rr="$4" vs="$5"
  local prefix="$STRESS_KEY_PREFIX"
  local case_label="${scenario} rep=${rep} c=${c} rr=${rr} vs=${vs}"
  local started_at now elapsed

  CASE_INDEX=$((CASE_INDEX + 1))
  started_at=$(date +%s)

  printf '[%d/%d] start %s\n' "$CASE_INDEX" "$TOTAL_CASES" "$case_label" >&2

  payload=$(jq -nc \
    --argjson d "$DURATION" \
    --argjson c "$c" \
    --argjson rr "$rr" \
    --argjson vs "$vs" \
    --argjson gid "$TARGET_GID" \
    --arg p "$prefix" \
    '{duration_sec:$d,concurrency:$c,read_ratio:$rr,value_size:$vs,target_gid:$gid,key_prefix:$p}')

  (
    curl --noproxy '*' -sS -X POST "$BASE/api/v1/stress/run" \
      -H 'Content-Type: application/json' \
      -d "$payload" \
    | jq -c --arg scenario "$scenario" --argjson repeat "$rep" --argjson c "$c" --argjson rr "$rr" --argjson vs "$vs" '
        if .code==0 then
          .result + {scenario:$scenario,repeat:$repeat,concurrency:$c,read_ratio:$rr,value_size:$vs}
        else
          {scenario:$scenario,repeat:$repeat,concurrency:$c,read_ratio:$rr,value_size:$vs,error:.message}
        end' >> "$OUT"
  ) &

  local pid=$!
  while kill -0 "$pid" 2>/dev/null; do
    now=$(date +%s)
    elapsed=$((now - started_at))
    printf '[%d/%d] running %s elapsed=%s\n' "$CASE_INDEX" "$TOTAL_CASES" "$case_label" "$(format_elapsed "$elapsed")" >&2
    sleep 5
  done

  if wait "$pid"; then
    now=$(date +%s)
    elapsed=$((now - started_at))
    printf '[%d/%d] done %s elapsed=%s\n' "$CASE_INDEX" "$TOTAL_CASES" "$case_label" "$(format_elapsed "$elapsed")" >&2
  else
    local status=$?
    now=$(date +%s)
    elapsed=$((now - started_at))
    printf '[%d/%d] failed %s elapsed=%s\n' "$CASE_INDEX" "$TOTAL_CASES" "$case_label" "$(format_elapsed "$elapsed")" >&2
    return "$status"
  fi
}

for c in "${CONCURRENCY_CASES[@]}"; do
  for rep in $(seq 1 "$REPEAT"); do run_case "concurrency" "$rep" "$c" 50 128; done
done

for rr in "${READ_RATIO_CASES[@]}"; do
  for rep in $(seq 1 "$REPEAT"); do run_case "read_ratio" "$rep" 32 "$rr" 128; done
done

for vs in "${VALUE_SIZE_CASES[@]}"; do
  for rep in $(seq 1 "$REPEAT"); do run_case "value_size" "$rep" 32 50 "$vs"; done
done

for rr in "${MIX_READ_RATIO_CASES[@]}"; do
  for vs in "${MIX_VALUE_SIZE_CASES[@]}"; do
    for rep in $(seq 1 "$REPEAT"); do run_case "mix" "$rep" 32 "$rr" "$vs"; done
  done
done