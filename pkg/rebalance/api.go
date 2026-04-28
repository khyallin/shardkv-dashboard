package rebalance

import (
	"time"

	"github.com/khyallin/shardkv/config"
)

type GroupRunningStatus struct {
	ID         int
	TotalQPS   float64
	DoneQPS    float64
	SuccessQPS float64
	MaxLatency time.Duration
	AvgLatency time.Duration
}

type Rebalancer interface {
	Rebalance(cfg *config.Config, groups map[int]*GroupRunningStatus) error
}
