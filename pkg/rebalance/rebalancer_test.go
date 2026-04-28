package rebalance

import (
	"testing"
	"time"

	"github.com/khyallin/shardkv/config"
)

// 示例：演示各个Rebalancer的行为

func TestQpsRebalancer(t *testing.T) {
	// 创建配置：4个分片，分配给2个组
	cfg := &config.Config{
		Num: 1,
		Shards: [config.NShards]config.Tgid{
			0: 1, 1: 1, 2: 1, 3: 1, 4: 2, 5: 2, 6: 2, 7: 2,
		},
		Groups: make(map[config.Tgid][]string),
	}

	// 模拟实际负载：组1非常繁忙，组2闲置
	groups := map[int]*GroupRunningStatus{
		1: {
			ID:         1,
			TotalQPS:   1000,
			DoneQPS:    900,
			SuccessQPS: 850,
			MaxLatency: 100 * time.Millisecond,
			AvgLatency: 50 * time.Millisecond,
		},
		2: {
			ID:         2,
			TotalQPS:   100, // 负载很低
			DoneQPS:    100,
			SuccessQPS: 100,
			MaxLatency: 10 * time.Millisecond,
			AvgLatency: 5 * time.Millisecond,
		},
	}

	r := &QpsRebalancer{}
	r.Rebalance(cfg, groups)

	// 验证有分片被迁移（从组1到组2）
	group2Shards := 0
	for _, gid := range cfg.Shards {
		if gid == 2 {
			group2Shards++
		}
	}
	t.Logf("After QpsRebalancer: Group2 has %d shards (expected 5)", group2Shards)
}

func TestLatencyAwareRebalancer(t *testing.T) {
	cfg := &config.Config{
		Num: 1,
		Shards: [config.NShards]config.Tgid{
			0: 1, 1: 1, 2: 1, 3: 1, 4: 2, 5: 2, 6: 2, 7: 2,
		},
		Groups: make(map[config.Tgid][]string),
	}

	// 场景：组1延迟高，组2延迟低
	groups := map[int]*GroupRunningStatus{
		1: {
			ID:         1,
			TotalQPS:   500,
			DoneQPS:    450,
			SuccessQPS: 450,
			MaxLatency: 500 * time.Millisecond, // 高延迟
			AvgLatency: 200 * time.Millisecond,
		},
		2: {
			ID:         2,
			TotalQPS:   500,
			DoneQPS:    500,
			SuccessQPS: 500,
			MaxLatency: 50 * time.Millisecond, // 低延迟
			AvgLatency: 20 * time.Millisecond,
		},
	}

	r := &LatencyAwareRebalancer{}
	r.Rebalance(cfg, groups)

	group2Shards := 0
	for _, gid := range cfg.Shards {
		if gid == 2 {
			group2Shards++
		}
	}
	t.Logf("After LatencyAwareRebalancer: Group2 has %d shards (expected 5)", group2Shards)
}

func TestMultiDimensionRebalancer(t *testing.T) {
	cfg := &config.Config{
		Num: 1,
		Shards: [config.NShards]config.Tgid{
			0: 1, 1: 1, 2: 1, 3: 1, 4: 2, 5: 2, 6: 2, 7: 2,
		},
		Groups: make(map[config.Tgid][]string),
	}

	// 复杂场景：组1 QPS高、延迟高、成功率低
	groups := map[int]*GroupRunningStatus{
		1: {
			ID:         1,
			TotalQPS:   1000,
			DoneQPS:    800,
			SuccessQPS: 700, // 成功率87.5%
			MaxLatency: 300 * time.Millisecond,
			AvgLatency: 150 * time.Millisecond,
		},
		2: {
			ID:         2,
			TotalQPS:   200,
			DoneQPS:    200,
			SuccessQPS: 200, // 成功率100%
			MaxLatency: 50 * time.Millisecond,
			AvgLatency: 20 * time.Millisecond,
		},
	}

	r := &MultiDimensionRebalancer{}
	r.Rebalance(cfg, groups)

	group2Shards := 0
	for _, gid := range cfg.Shards {
		if gid == 2 {
			group2Shards++
		}
	}
	t.Logf("After MultiDimensionRebalancer: Group2 has %d shards (expected 5)", group2Shards)
}

func TestMultiDimensionRebalancerNewWeights(t *testing.T) {
	r := New(0.2, 0.3, 0.5)
	if r.qpsWeight != 0.2 {
		t.Fatalf("unexpected qpsWeight: got %v want %v", r.qpsWeight, 0.2)
	}
	if r.latencyWeight != 0.3 {
		t.Fatalf("unexpected latencyWeight: got %v want %v", r.latencyWeight, 0.3)
	}
	if r.failureWeight != 0.5 {
		t.Fatalf("unexpected failureWeight: got %v want %v", r.failureWeight, 0.5)
	}

	defaultRebalancer := New()
	if defaultRebalancer.qpsWeight != defaultQpsWeight {
		t.Fatalf("unexpected default qpsWeight: got %v want %v", defaultRebalancer.qpsWeight, defaultQpsWeight)
	}
	if defaultRebalancer.latencyWeight != defaultLatencyWeight {
		t.Fatalf("unexpected default latencyWeight: got %v want %v", defaultRebalancer.latencyWeight, defaultLatencyWeight)
	}
	if defaultRebalancer.failureWeight != defaultFailureWeight {
		t.Fatalf("unexpected default failureWeight: got %v want %v", defaultRebalancer.failureWeight, defaultFailureWeight)
	}
}

func TestGradualRebalancer(t *testing.T) {
	cfg := &config.Config{
		Num: 1,
		Shards: [config.NShards]config.Tgid{
			0: 1, 1: 1, 2: 1, 3: 1, 4: 1, 5: 2, 6: 2, 7: 2,
		},
		Groups: make(map[config.Tgid][]string),
	}

	// 场景：组1有5个分片，QPS高；组2有3个分片，QPS低
	groups := map[int]*GroupRunningStatus{
		1: {
			ID:         1,
			TotalQPS:   500,
			DoneQPS:    450,
			SuccessQPS: 450,
			MaxLatency: 100 * time.Millisecond,
			AvgLatency: 50 * time.Millisecond,
		},
		2: {
			ID:         2,
			TotalQPS:   100,
			DoneQPS:    100,
			SuccessQPS: 100,
			MaxLatency: 50 * time.Millisecond,
			AvgLatency: 20 * time.Millisecond,
		},
	}

	r := NewGradualRebalancer()
	r.Rebalance(cfg, groups)

	group1Shards := 0
	group2Shards := 0
	for _, gid := range cfg.Shards {
		if gid == 1 {
			group1Shards++
		} else {
			group2Shards++
		}
	}
	t.Logf("After GradualRebalancer: Group1 has %d shards, Group2 has %d shards", group1Shards, group2Shards)
	t.Logf("Note: GradualRebalancer only moves max 12.5%% per round, may need multiple rounds to fully balance")
}

func TestSuccessAwareRebalancer(t *testing.T) {
	cfg := &config.Config{
		Num: 1,
		Shards: [config.NShards]config.Tgid{
			0: 1, 1: 1, 2: 1, 3: 1, 4: 2, 5: 2, 6: 2, 7: 2,
		},
		Groups: make(map[config.Tgid][]string),
	}

	// 故障场景：组1成功率低（故障中），组2成功率正常
	groups := map[int]*GroupRunningStatus{
		1: {
			ID:         1,
			TotalQPS:   500,
			DoneQPS:    400,
			SuccessQPS: 200, // 成功率仅50%，说明有故障
			MaxLatency: 500 * time.Millisecond,
			AvgLatency: 250 * time.Millisecond,
		},
		2: {
			ID:         2,
			TotalQPS:   500,
			DoneQPS:    480,
			SuccessQPS: 475, // 成功率99%
			MaxLatency: 100 * time.Millisecond,
			AvgLatency: 50 * time.Millisecond,
		},
	}

	r := &SuccessAwareRebalancer{}
	r.Rebalance(cfg, groups)

	group2Shards := 0
	for _, gid := range cfg.Shards {
		if gid == 2 {
			group2Shards++
		}
	}
	t.Logf("After SuccessAwareRebalancer: Group2 has %d shards (expected 5)", group2Shards)
	t.Logf("Note: SuccessAwareRebalancer helps isolate faulty servers")
}

// 性能对比测试
func BenchmarkRebalancers(b *testing.B) {
	// 基础配置
	groups := map[int]*GroupRunningStatus{
		1: {ID: 1, TotalQPS: 1000, DoneQPS: 900, SuccessQPS: 850, MaxLatency: 100 * time.Millisecond, AvgLatency: 50 * time.Millisecond},
		2: {ID: 2, TotalQPS: 100, DoneQPS: 100, SuccessQPS: 100, MaxLatency: 10 * time.Millisecond, AvgLatency: 5 * time.Millisecond},
	}

	b.Run("QpsRebalancer", func(b *testing.B) {
		r := &QpsRebalancer{}
		for i := 0; i < b.N; i++ {
			cfg := &config.Config{
				Shards: [config.NShards]config.Tgid{
					0: 1, 1: 1, 2: 1, 3: 1, 4: 2, 5: 2, 6: 2, 7: 2,
				},
			}
			r.Rebalance(cfg, groups)
		}
	})

	b.Run("LatencyAwareRebalancer", func(b *testing.B) {
		r := &LatencyAwareRebalancer{}
		for i := 0; i < b.N; i++ {
			cfg := &config.Config{
				Shards: [config.NShards]config.Tgid{
					0: 1, 1: 1, 2: 1, 3: 1, 4: 2, 5: 2, 6: 2, 7: 2,
				},
			}
			r.Rebalance(cfg, groups)
		}
	})

	b.Run("MultiDimensionRebalancer", func(b *testing.B) {
		r := &MultiDimensionRebalancer{}
		for i := 0; i < b.N; i++ {
			cfg := &config.Config{
				Shards: [config.NShards]config.Tgid{
					0: 1, 1: 1, 2: 1, 3: 1, 4: 2, 5: 2, 6: 2, 7: 2,
				},
			}
			r.Rebalance(cfg, groups)
		}
	})

	b.Run("GradualRebalancer", func(b *testing.B) {
		r := NewGradualRebalancer()
		for i := 0; i < b.N; i++ {
			cfg := &config.Config{
				Shards: [config.NShards]config.Tgid{
					0: 1, 1: 1, 2: 1, 3: 1, 4: 2, 5: 2, 6: 2, 7: 2,
				},
			}
			r.Rebalance(cfg, groups)
		}
	})

	b.Run("SuccessAwareRebalancer", func(b *testing.B) {
		r := &SuccessAwareRebalancer{}
		for i := 0; i < b.N; i++ {
			cfg := &config.Config{
				Shards: [config.NShards]config.Tgid{
					0: 1, 1: 1, 2: 1, 3: 1, 4: 2, 5: 2, 6: 2, 7: 2,
				},
			}
			r.Rebalance(cfg, groups)
		}
	})
}
