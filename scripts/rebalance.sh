#!/usr/bin/env bash
set -euo pipefail

BASE="${BASE:-http://127.0.0.1:8080}"
STRESS_OUT="${STRESS_OUT:-tmp/stress_results.jsonl}"
OUT="${OUT:-tmp/rebalance_results.jsonl}"
TARGET_GID="${TARGET_GID:-1}"

DURATION="${DURATION:-30}"
REPEAT="${REPEAT:-3}"

# 重置分片配置、给 ShardKV/控制器一点收敛时间（秒）
RESET_SETTLE_SEC="${RESET_SETTLE_SEC:-2}"

# 后端均已实现：null/num/qps/multidim + latency(延迟感知)/success(成功率感知)/gradual(渐进式)
# 若需子集可改此数组，或：`ALGORITHMS_STR='null num' bash scripts/rebalance.sh`（见下方）
ALGORITHMS=(null num qps multidim latency success gradual)
if [[ -n "${ALGORITHMS_STR:-}" ]]; then
  read -r -a ALGORITHMS <<< "$ALGORITHMS_STR"
fi

if [[ ! -f "$STRESS_OUT" ]]; then
  echo "stress results not found: $STRESS_OUT" >&2
  exit 1
fi

mkdir -p "$(dirname "$OUT")"
: > "$OUT"

select_best_params() {
  python3 - "$1" <<'PY'
import json, sys
from collections import defaultdict

path = sys.argv[1]
best = None

rows = []
with open(path, "r", encoding="utf-8") as f:
    for line in f:
        r = json.loads(line)
        if "error" in r:
            continue
        # stress.sh 会把 success_tps/avg_latency/max_latency 直接打平到 result 上
        rows.append(r)

group = defaultdict(list)
for r in rows:
    key = (int(r["concurrency"]), int(r["read_ratio"]), int(r["value_size"]))
    group[key].append(r)

def median(xs):
    xs = sorted(xs)
    n = len(xs)
    if n == 0:
        return None
    if n % 2 == 1:
        return xs[n//2]
    return (xs[n//2-1] + xs[n//2]) / 2.0

candidates = []
for (c, rr, vs), rs in group.items():
    success = [float(x["success_tps"]) for x in rs]
    avg_ns = [int(x["avg_latency"]) for x in rs]
    max_ns = [int(x["max_latency"]) for x in rs]
    candidates.append({
        "concurrency": c,
        "read_ratio": rr,
        "value_size": vs,
        "success_tps_med": median(success),
        "avg_ms_med": median(avg_ns)/1e6,
        "max_ms_med": median(max_ns)/1e6,
        "n": len(rs),
    })

# 选：success_tps 最大；同分优先 avg_latency 更低；再优先 max_latency 更低
def key_fn(x):
    return (-x["success_tps_med"], x["avg_ms_med"], x["max_ms_med"])

candidates.sort(key=key_fn)
top = candidates[0]

print(top["concurrency"], top["read_ratio"], top["value_size"])
PY
}

BEST_LINE="$(select_best_params "$STRESS_OUT")"
BEST_C="$(echo "$BEST_LINE" | awk '{print $1}')"
BEST_RR="$(echo "$BEST_LINE" | awk '{print $2}')"
BEST_VS="$(echo "$BEST_LINE" | awk '{print $3}')"

if [[ -z "${BEST_C}" || -z "${BEST_RR}" || -z "${BEST_VS}" ]]; then
  echo "failed to select best params" >&2
  exit 1
fi

echo "best params selected from $STRESS_OUT: c=$BEST_C rr=$BEST_RR vs=$BEST_VS" >&2

get_group_get() {
  curl --noproxy '*' -fsS -sS -X POST "$BASE/api/v1/group/get" \
    -H 'Content-Type: application/json' \
    -d '{}' 
}

set_auto_mode() {
  local auto="$1"
  local mode="${2:-}"
  if [[ "$auto" == "true" ]]; then
    curl --noproxy '*' -fsS -sS -X POST "$BASE/api/v1/config/set" \
      -H 'Content-Type: application/json' \
      -d "$(jq -nc --argjson a "$auto" --arg m "$mode" '{auto:$a, mode:$m}')"
  else
    curl --noproxy '*' -fsS -sS -X POST "$BASE/api/v1/config/set" \
      -H 'Content-Type: application/json' \
      -d '{"auto":false}'
  fi
}

BASELINE_SHARDS_JSON="$(get_group_get | jq -c '.data.shards')"
N_SHARDS="$(echo "$BASELINE_SHARDS_JSON" | jq -r 'length')"

restore_baseline() {
  # 防止恢复过程中 ticker 继续迁移
  set_auto_mode "false" ""

  local current
  current="$(get_group_get | jq -c '.data.shards')"

  for ((i=0; i<N_SHARDS; i++)); do
    local from_gid to_gid
    from_gid="$(echo "$current" | jq -r ".[$i]")"
    to_gid="$(echo "$BASELINE_SHARDS_JSON" | jq -r ".[$i]")"
    if [[ "$from_gid" != "$to_gid" ]]; then
      curl --noproxy '*' -fsS -sS -X POST "$BASE/api/v1/config" \
        -H 'Content-Type: application/json' \
        -d "$(jq -nc --argjson shard "$i" --argjson from "$from_gid" --argjson to "$to_gid" '{shard:$shard, from:$from, to:$to}')"
    fi
  done

  sleep "$RESET_SETTLE_SEC"
}

run_algo_once() {
  local algo="$1"
  local rep="$2"
  local prefix="stress-rebalance-${algo}-g${TARGET_GID}-rep${rep}-"

  local payload
  payload="$(jq -nc \
    --argjson d "$DURATION" \
    --argjson c "$BEST_C" \
    --argjson rr "$BEST_RR" \
    --argjson vs "$BEST_VS" \
    --argjson gid "$TARGET_GID" \
    --arg p "$prefix" \
    '{duration_sec:$d, concurrency:$c, read_ratio:$rr, value_size:$vs, target_gid:$gid, key_prefix:$p}')"

  curl --noproxy '*' -sS -X POST "$BASE/api/v1/stress/run" \
    -H 'Content-Type: application/json' \
    -d "$payload" \
  | jq -c --arg scenario "$algo" --argjson repeat "$rep" --argjson c "$BEST_C" --argjson rr "$BEST_RR" --argjson vs "$BEST_VS" '
      if .code==0 then
        .result + {scenario:$scenario, repeat:$repeat, concurrency:$c, read_ratio:$rr, value_size:$vs}
      else
        {scenario:$scenario, repeat:$repeat, concurrency:$c, read_ratio:$rr, value_size:$vs, error:.message}
      end
    '
}

for algo in "${ALGORITHMS[@]}"; do
  echo "=== algo=$algo ===" >&2
  for rep in $(seq 1 "$REPEAT"); do
    echo "[algo=$algo rep=$rep] restoring baseline..." >&2
    restore_baseline

    echo "[algo=$algo rep=$rep] enabling auto mode..." >&2
    set_auto_mode "true" "$algo"

    run_algo_once "$algo" "$rep" >> "$OUT"
  done
done

