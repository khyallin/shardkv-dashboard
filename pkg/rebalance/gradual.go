package rebalance

import (
	"math"
	"math/rand"

	"github.com/khyallin/shardkv/config"
)

// GradualRebalancer 渐进式均衡器
// 模仿Kubernetes和YARN的设计，采用更保守的策略：
// - 单次只迁移少量分片，避免过度变动
// - 当负载不均衡比例超过阈值时才触发迁移
// - 优先在相邻的两个组之间迁移，减少整体变动
type GradualRebalancer struct {
	maxMovePerRound    float64 // 单轮最多迁移多少比例的分片，默认12.5%（1/8）
	imbalanceThreshold float64 // 不均衡阈值，默认25%
}

// NewGradualRebalancer 创建渐进式均衡器
func NewGradualRebalancer() *GradualRebalancer {
	return &GradualRebalancer{
		maxMovePerRound:    0.125, // 每轮最多迁移12.5%的分片
		imbalanceThreshold: 0.25,  // 25%的负载差异才触发迁移
	}
}

func (g *GradualRebalancer) Rebalance(cfg *config.Config, groups map[int]*GroupRunningStatus) error {
	if len(groups) <= 1 {
		return nil
	}

	// 计算平均QPS
	var totalQPS float64
	for _, status := range groups {
		totalQPS += status.TotalQPS
	}
	avgQPS := totalQPS / float64(len(groups))
	if avgQPS == 0 {
		return nil
	}

	// 找到负载最高和最低的两个组
	var maxQPS, minQPS float64 = -1.0, 1e8
	var maxGid, minGid int

	for gid, status := range groups {
		if status.TotalQPS > maxQPS {
			maxQPS = status.TotalQPS
			maxGid = gid
		}
		if status.TotalQPS < minQPS {
			minQPS = status.TotalQPS
			minGid = gid
		}
	}

	// 检查不均衡比例是否超过阈值
	if maxQPS > 0 && (maxQPS-avgQPS)/avgQPS < g.imbalanceThreshold {
		return nil
	}

	if minGid == maxGid {
		return nil
	}

	// 计算本轮最多可迁移的分片数
	maxMoveCount := int(math.Ceil(float64(config.NShards) * g.maxMovePerRound))
	if maxMoveCount == 0 {
		maxMoveCount = 1
	}

	// 统计max组的分片数，不能全部迁移
	maxGroupShardCount := 0
	var shards []int
	for shard, gid := range cfg.Shards {
		if gid == config.Tgid(maxGid) {
			maxGroupShardCount++
			shards = append(shards, shard)
		}
	}

	if len(shards) == 0 {
		return nil
	}

	// 确保至少保留一个分片在源组，不能全部迁移
	moveCounts := 1
	if maxGroupShardCount > 1 && maxMoveCount > 1 {
		moveCounts = int(math.Min(float64(maxMoveCount), float64(maxGroupShardCount-1)))
	}

	// 迁移选定数量的分片
	for i := 0; i < moveCounts && len(shards) > 0; i++ {
		idx := rand.Intn(len(shards))
		move := shards[idx]
		cfg.Shards[move] = config.Tgid(minGid)
		// 移除已选择的分片
		shards = append(shards[:idx], shards[idx+1:]...)
	}

	return nil
}

var _ Rebalancer = &GradualRebalancer{}
