package rebalance

import (
	"math/rand"

	"github.com/khyallin/shardkv/config"
)

// QpsRebalancer 按总 QPS 做最小粒度迁移，适合简单场景。
type QpsRebalancer struct{}

func (QpsRebalancer) Rebalance(cfg *config.Config, groups map[int]*GroupRunningStatus) error {
	minqps, mingid, maxqps, maxgid := 1e8, -1, -1.0, -1
	for gid, status := range groups {
		if minqps < 0 || status.TotalQPS < minqps {
			minqps = status.TotalQPS
			mingid = gid
		}
		if maxqps < 0 || status.TotalQPS > maxqps {
			maxqps = status.TotalQPS
			maxgid = gid
		}
	}
	if mingid == maxgid {
		return nil
	}
	var shards []int
	for shard, gid := range cfg.Shards {
		if gid == config.Tgid(maxgid) {
			shards = append(shards, shard)
		}
	}

	// 防御：当 maxgid 组没有任何分片时，避免 rand.Intn(0) panic。
	if len(shards) == 0 {
		return nil
	}

	move := shards[rand.Intn(len(shards))]
	cfg.Shards[move] = config.Tgid(mingid)
	return nil
}

var _ Rebalancer = &QpsRebalancer{}
