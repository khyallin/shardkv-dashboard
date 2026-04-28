package rebalance

import (
	"math/rand"

	"github.com/khyallin/shardkv/config"
)

// LatencyAwareRebalancer 基于延迟指标的均衡器
// 将分片从高延迟组迁移到低延迟组，考虑最大延迟和平均延迟
type LatencyAwareRebalancer struct{}

func (LatencyAwareRebalancer) Rebalance(cfg *config.Config, groups map[int]*GroupRunningStatus) error {
	if len(groups) <= 1 {
		return nil
	}

	// 找延迟最高和最低的组
	var minLatencyGid, maxLatencyGid int
	var minLatency, maxLatency float64 = 1e8, -1.0

	for gid, status := range groups {
		latency := float64(status.MaxLatency.Milliseconds())
		if latency < minLatency {
			minLatency = latency
			minLatencyGid = gid
		}
		if latency > maxLatency {
			maxLatency = latency
			maxLatencyGid = gid
		}
	}

	// 如果延迟差异不明显，不进行迁移（避免频繁抖动）
	if maxLatency > 0 && minLatency > 0 && (maxLatency-minLatency)/minLatency < 0.5 {
		return nil
	}

	if minLatencyGid == maxLatencyGid {
		return nil
	}

	// 从高延迟组中找一个分片迁移到低延迟组
	var shards []int
	for shard, gid := range cfg.Shards {
		if gid == config.Tgid(maxLatencyGid) {
			shards = append(shards, shard)
		}
	}

	if len(shards) == 0 {
		return nil
	}

	move := shards[rand.Intn(len(shards))]
	cfg.Shards[move] = config.Tgid(minLatencyGid)
	return nil
}

var _ Rebalancer = &LatencyAwareRebalancer{}
