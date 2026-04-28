package rebalance

import (
	"math/rand"

	"github.com/khyallin/shardkv/config"
)

// SuccessAwareRebalancer 可用性感知的均衡器
// 优先关注请求成功率，而不是吞吐量
// 将分片从成功率低的组迁移到成功率高的组，有助于提升系统整体的可用性
type SuccessAwareRebalancer struct{}

func (SuccessAwareRebalancer) Rebalance(cfg *config.Config, groups map[int]*GroupRunningStatus) error {
	if len(groups) <= 1 {
		return nil
	}

	// 计算每个组的成功率
	successRates := make(map[int]float64)
	var minSuccessRate float64 = 2.0
	var maxSuccessRate float64 = -1.0
	var minGid, maxGid int

	for gid, status := range groups {
		var rate float64
		if status.DoneQPS > 0 {
			rate = status.SuccessQPS / status.DoneQPS
		}
		successRates[gid] = rate

		if rate < minSuccessRate {
			minSuccessRate = rate
			minGid = gid
		}
		if rate > maxSuccessRate {
			maxSuccessRate = rate
			maxGid = gid
		}
	}

	// 如果成功率差异很小，不需要迁移
	if minSuccessRate > 0 && maxSuccessRate > 0 {
		diff := maxSuccessRate - minSuccessRate
		if diff < 0.01 { // 少于1%的差异
			return nil
		}
	}

	if minGid == maxGid {
		return nil
	}

	// 从低成功率的组迁移分片到高成功率的组
	var shards []int
	for shard, gid := range cfg.Shards {
		if gid == config.Tgid(minGid) {
			shards = append(shards, shard)
		}
	}

	if len(shards) == 0 {
		return nil
	}

	move := shards[rand.Intn(len(shards))]
	cfg.Shards[move] = config.Tgid(maxGid)
	return nil
}

var _ Rebalancer = &SuccessAwareRebalancer{}
