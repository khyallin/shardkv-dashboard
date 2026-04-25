import json
import os
import matplotlib
matplotlib.use("Agg")
import pandas as pd
import matplotlib.pyplot as plt
import seaborn as sns

input_file = os.getenv("OUT", "tmp/stress_results.jsonl")
fig_dir = os.getenv("FIG_DIR", "tmp")
os.makedirs(fig_dir, exist_ok=True)

sns.set_theme(style="whitegrid")

rows = []
with open(input_file, "r", encoding="utf-8") as f:
    for line in f:
        r = json.loads(line)
        if "error" in r:
            continue
        rows.append({
            "scenario": r["scenario"],
            "repeat": r["repeat"],
            "concurrency": r["concurrency"],
            "read_ratio": r["read_ratio"],
            "value_size": r["value_size"],
            "success_tps": r["success_tps"],
            "avg_ms": r["avg_latency"] / 1e6,
            "max_ms": r["max_latency"] / 1e6,
            "wrong_leader": r["errors"]["wrong_leader"],
            "wrong_group": r["errors"]["wrong_group"],
            "version_error": r["errors"]["version_error"],
            "other_error": r["errors"]["other"],
            "before_qps": r["before_status"]["total_qps"],
            "after_qps": r["after_status"]["total_qps"],
            "before_avg_ms": r["before_status"]["avg_latency"] / 1e6,
            "after_avg_ms": r["after_status"]["avg_latency"] / 1e6
        })

df = pd.DataFrame(rows)
if df.empty:
    raise SystemExit("no valid rows in stress results")

# 1) 并发双轴图
d1 = df[df["scenario"] == "concurrency"].groupby("concurrency", as_index=False).median(numeric_only=True)
fig, ax1 = plt.subplots(figsize=(8, 4.5))
ax1.plot(d1["concurrency"], d1["success_tps"], marker="o")
ax1.set_xlabel("concurrency")
ax1.set_ylabel("success_tps")
ax2 = ax1.twinx()
ax2.plot(d1["concurrency"], d1["avg_ms"], marker="s", color="tab:red")
ax2.set_ylabel("avg_latency_ms")
fig.tight_layout()
fig.savefig(os.path.join(fig_dir, "fig1_concurrency.png"), dpi=160)

# 2) 二维热力图（mix）
d2 = df[df["scenario"] == "mix"]
pivot = d2.pivot_table(index="read_ratio", columns="value_size", values="success_tps", aggfunc="median")
plt.figure(figsize=(7, 5))
sns.heatmap(pivot, annot=True, fmt=".0f", cmap="YlGnBu")
plt.title("success_tps heatmap")
plt.tight_layout()
plt.savefig(os.path.join(fig_dir, "fig2_heatmap.png"), dpi=160)

# 3) 错误构成
d3 = df.groupby("scenario")[["wrong_leader", "wrong_group", "version_error", "other_error"]].sum()
d3.plot(kind="bar", stacked=True, figsize=(8, 4.5))
plt.ylabel("error count")
plt.tight_layout()
plt.savefig(os.path.join(fig_dir, "fig3_errors.png"), dpi=160)

# 4) 前后QPS对比
d4 = df.groupby("scenario", as_index=False)[["before_qps", "after_qps"]].median(numeric_only=True)
x = range(len(d4))
w = 0.35
plt.figure(figsize=(8, 4.5))
plt.bar([i - w/2 for i in x], d4["before_qps"], width=w, label="before_qps")
plt.bar([i + w/2 for i in x], d4["after_qps"], width=w, label="after_qps")
plt.xticks(list(x), d4["scenario"])
plt.legend()
plt.tight_layout()
plt.savefig(os.path.join(fig_dir, "fig4_before_after_qps.png"), dpi=160)