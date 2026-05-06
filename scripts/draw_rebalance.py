import json
import os

import matplotlib

matplotlib.use("Agg")
import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns

input_file = os.getenv("OUT", "tmp/rebalance_results.jsonl")
fig_dir = os.getenv("FIG_DIR", "tmp")
# 逗号分隔，控制图中横轴顺序；缺省为常用基线 + 扩展策略
DEFAULT_SCENARIO_ORDER = [
    "null",
    "num",
    "qps",
    "multidim",
    "latency",
    "success",
    "gradual",
]
os.makedirs(fig_dir, exist_ok=True)

sns.set_theme(style="whitegrid")


def pareto_frontier(df_metric, x_col, y_col):
    """
    计算 Pareto 前沿（x 越大越好，y 越小越好）。
    返回按 x 升序排序后的前沿点 DataFrame。
    """
    ordered = df_metric.sort_values([x_col, y_col], ascending=[False, True]).copy()
    frontier_rows = []
    best_y = float("inf")
    for _, row in ordered.iterrows():
        if row[y_col] <= best_y:
            frontier_rows.append(row)
            best_y = row[y_col]
    if not frontier_rows:
        return ordered.iloc[0:0]
    frontier = pd.DataFrame(frontier_rows).sort_values(x_col, ascending=True)
    return frontier

rows = []
with open(input_file, "r", encoding="utf-8") as f:
    for line in f:
        r = json.loads(line)
        if "error" in r:
            continue
        rows.append(
            {
                "scenario": r.get("scenario", ""),
                "repeat": r.get("repeat", 0),
                "concurrency": r["concurrency"],
                "read_ratio": r["read_ratio"],
                "value_size": r["value_size"],
                "success_tps": r["success_tps"],
                "tps": r.get("tps", r["success_tps"]),
                "avg_ms": r["avg_latency"] / 1e6,
                "max_ms": r["max_latency"] / 1e6,
            }
        )

df = pd.DataFrame(rows)
if df.empty:
    raise SystemExit(f"no valid rows in {input_file}")

_so = os.getenv("SCENARIO_ORDER")
if _so:
    preferred = [x.strip() for x in _so.split(",") if x.strip()]
else:
    preferred = list(DEFAULT_SCENARIO_ORDER)
_seen = set(df["scenario"].unique())
scenario_cat = [s for s in preferred if s in _seen] + sorted(_seen - set(preferred))
df["scenario"] = pd.Categorical(df["scenario"], categories=scenario_cat, ordered=True)

# 基于统一选出的最优压测参数（不同算法只应该变化 scenario）
selected_params = df[["concurrency", "read_ratio", "value_size"]].drop_duplicates().to_dict("records")[0]
c_str = f"c={selected_params['concurrency']}, rr={selected_params['read_ratio']}, vs={selected_params['value_size']}"

_n = len(scenario_cat)
_fig_w = max(7.5, min(14.0, 1.05 * _n + 3.2))

# 1) success_tps 对比（箱线图）
plt.figure(figsize=(_fig_w, 4.5))
ax = sns.boxplot(data=df, x="scenario", y="success_tps", showfliers=False)
ax.set_xlabel("algorithm")
ax.set_ylabel("success_tps")
ax.set_title(f"success_tps by algorithm ({c_str})")
plt.tight_layout()
plt.savefig(os.path.join(fig_dir, "rebalance_fig1_success_tps.png"), dpi=160)

# 2) 平均延迟对比
plt.figure(figsize=(_fig_w, 4.5))
ax = sns.boxplot(data=df, x="scenario", y="avg_ms", showfliers=False)
ax.set_xlabel("algorithm")
ax.set_ylabel("avg_latency_ms")
ax.set_title(f"avg latency by algorithm ({c_str})")
plt.tight_layout()
plt.savefig(os.path.join(fig_dir, "rebalance_fig2_avg_latency.png"), dpi=160)

# 3) 最大延迟对比
plt.figure(figsize=(_fig_w, 4.5))
ax = sns.boxplot(data=df, x="scenario", y="max_ms", showfliers=False)
ax.set_xlabel("algorithm")
ax.set_ylabel("max_latency_ms")
ax.set_title(f"max latency by algorithm ({c_str})")
plt.tight_layout()
plt.savefig(os.path.join(fig_dir, "rebalance_fig3_max_latency.png"), dpi=160)

# 4) 散点：吞吐 vs 平均延迟
plt.figure(figsize=(_fig_w, 4.8))
# 先画全部原始点（灰色，弱化）
sns.scatterplot(
    data=df,
    x="success_tps",
    y="avg_ms",
    color="0.75",
    alpha=0.35,
    s=32,
    linewidth=0,
    legend=False,
)

# 再画按策略聚合后的中位数点（突出可读性）
median_by_algo = (
    df.groupby("scenario", observed=False, as_index=False)
    .median(numeric_only=True)
    .sort_values("scenario")
)
sns.scatterplot(
    data=median_by_algo,
    x="success_tps",
    y="avg_ms",
    hue="scenario",
    style="scenario",
    s=130,
    edgecolor="black",
    linewidth=0.5,
)

# 叠加 Pareto 前沿
frontier = pareto_frontier(median_by_algo, "success_tps", "avg_ms")
if not frontier.empty:
    plt.plot(
        frontier["success_tps"],
        frontier["avg_ms"],
        linestyle="--",
        linewidth=1.6,
        color="black",
        label="pareto_frontier(median)",
    )
    # 只标注前沿点，避免整图文字拥挤
    for _, row in frontier.iterrows():
        plt.annotate(
            str(row["scenario"]),
            (row["success_tps"], row["avg_ms"]),
            textcoords="offset points",
            xytext=(4, 6),
            fontsize=8,
        )

plt.xlabel("success_tps")
plt.ylabel("avg_latency_ms")
plt.title(f"throughput vs latency + pareto (median per algorithm) ({c_str})")
plt.tight_layout()
plt.savefig(os.path.join(fig_dir, "rebalance_fig4_scatter_tps_latency.png"), dpi=160)

print("plots saved to", fig_dir)

