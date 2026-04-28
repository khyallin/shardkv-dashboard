package rebalance

import "github.com/khyallin/shardkv/config"

// NumRebalancer 直接复用配置层的分片数量均衡策略。
type NumRebalancer struct{}

func (NumRebalancer) Rebalance(cfg *config.Config, groups map[int]*GroupRunningStatus) error {
	cfg.Rebalance()
	return nil
}

var _ Rebalancer = &NumRebalancer{}
