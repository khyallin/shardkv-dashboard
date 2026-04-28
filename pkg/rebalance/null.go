package rebalance

import "github.com/khyallin/shardkv/config"

// NullRebalancer 保持当前分片分配不变，适合关闭自动均衡。
type NullRebalancer struct{}

func (r *NullRebalancer) Rebalance(cfg *config.Config, groups map[int]*GroupRunningStatus) error {
	return nil
}

var _ Rebalancer = &NullRebalancer{}
