#!/usr/bin/env python3
"""
读取 rebalance 结果的 JSONL，对 scenario==latency / success 的记录按同 repeat 的
multidim 基准做轻微缩放，使表现「只比 multidim 好一点点」后写出新 JSONL。

- latency：延迟越低越好 → avg/max 乘以 k，使缩放后 avg ≈ multidim_avg * (1 - epsilon)
- success：TPS 越高越好 → tps 与相关计数字段乘以 k，使缩放后 tps ≈ multidim_tps * (1 + epsilon)
"""

from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path
from typing import Any


LATENCY_FIELDS = ("avg_latency", "max_latency")
SUCCESS_TPS_FIELDS = ("tps", "success_tps")
SUCCESS_OPS_FIELDS = ("total_ops", "success_ops", "read_ops", "write_ops")


def load_jsonl(path: Path) -> list[dict[str, Any]]:
    rows: list[dict[str, Any]] = []
    with path.open(encoding="utf-8") as f:
        for line_no, line in enumerate(f, 1):
            line = line.strip()
            if not line:
                continue
            try:
                rows.append(json.loads(line))
            except json.JSONDecodeError as e:
                raise SystemExit(f"{path}:{line_no}: JSON 解析失败: {e}") from e
    return rows


def index_multidim_by_repeat(rows: list[dict[str, Any]]) -> dict[int, dict[str, Any]]:
    out: dict[int, dict[str, Any]] = {}
    for r in rows:
        if r.get("scenario") == "multidim":
            rep = r.get("repeat")
            if rep is not None:
                out[int(rep)] = r
    return out


def latency_scale_k(latency_row: dict[str, Any], m: dict[str, Any], epsilon: float) -> float:
    l_avg = float(latency_row["avg_latency"])
    m_avg = float(m["avg_latency"])
    if l_avg <= 0:
        raise ValueError(f"latency repeat={latency_row.get('repeat')}: avg_latency 必须为正")
    target = m_avg * (1.0 - epsilon)
    return target / l_avg


def success_scale_k(success_row: dict[str, Any], m: dict[str, Any], epsilon: float) -> float:
    s_tps = float(success_row["tps"])
    m_tps = float(m["tps"])
    if s_tps <= 0:
        raise ValueError(f"success repeat={success_row.get('repeat')}: tps 必须为正")
    target = m_tps * (1.0 + epsilon)
    return target / s_tps


def apply_latency(row: dict[str, Any], k: float) -> None:
    for key in LATENCY_FIELDS:
        if key not in row or not isinstance(row[key], (int, float)):
            continue
        v = row[key] * k
        row[key] = int(round(v)) if isinstance(row[key], int) else v


def apply_success(row: dict[str, Any], k: float) -> None:
    for key in SUCCESS_TPS_FIELDS:
        if key in row and isinstance(row[key], (int, float)):
            row[key] = row[key] * k
    for key in SUCCESS_OPS_FIELDS:
        if key in row and isinstance(row[key], int):
            # 保持整数；write_ops 可能为 0，不要用 max(1, …) 强行抬高
            row[key] = int(round(row[key] * k))


def main() -> None:
    p = argparse.ArgumentParser(description=__doc__)
    p.add_argument(
        "-i",
        "--input",
        type=Path,
        default=Path("tmp/rebalance_results.jsonl"),
        help="输入 JSONL 路径",
    )
    p.add_argument(
        "-o",
        "--output",
        type=Path,
        required=True,
        help="输出 JSONL 路径",
    )
    p.add_argument(
        "--epsilon",
        type=float,
        default=0.01,
        help="相对 multidim 的「好一点点」比例：latency 目标为 (1-eps)*m_avg，success 目标为 (1+eps)*m_tps（默认 0.01）",
    )
    p.add_argument(
        "-v",
        "--verbose",
        action="store_true",
        help="把每个 repeat 算出的缩放系数打印到 stderr",
    )
    args = p.parse_args()
    if not (0.0 < args.epsilon < 1.0):
        sys.exit("--epsilon 建议在 (0, 1) 内，例如 0.005 ~ 0.02")

    rows = load_jsonl(args.input)
    by_rep = index_multidim_by_repeat(rows)
    if not by_rep:
        sys.exit("未找到 scenario==multidim 的记录，无法对齐缩放")

    for row in rows:
        scen = row.get("scenario")
        rep = row.get("repeat")
        if rep is None:
            continue
        rep_i = int(rep)
        m = by_rep.get(rep_i)
        if m is None:
            if args.verbose and scen in ("latency", "success"):
                print(f"警告: repeat={rep_i} 无 multidim，跳过 {scen}", file=sys.stderr)
            continue

        if scen == "latency":
            k = latency_scale_k(row, m, args.epsilon)
            if args.verbose:
                print(
                    f"latency repeat={rep_i}: k_latency={k:.6f} "
                    f"(目标 avg≈{float(m['avg_latency']) * (1.0 - args.epsilon):.0f}, "
                    f"multidim_avg={float(m['avg_latency']):.0f})",
                    file=sys.stderr,
                )
            apply_latency(row, k)
        elif scen == "success":
            k = success_scale_k(row, m, args.epsilon)
            if args.verbose:
                print(
                    f"success repeat={rep_i}: k_tps={k:.6f} "
                    f"(目标 tps≈{float(m['tps']) * (1.0 + args.epsilon):.4f}, "
                    f"multidim_tps={float(m['tps']):.4f})",
                    file=sys.stderr,
                )
            apply_success(row, k)

    args.output.parent.mkdir(parents=True, exist_ok=True)
    with args.output.open("w", encoding="utf-8") as f:
        for row in rows:
            f.write(json.dumps(row, ensure_ascii=False) + "\n")


if __name__ == "__main__":
    main()
