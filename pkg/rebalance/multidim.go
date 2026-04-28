package rebalance

import (
	"math"
	"math/rand"

	"github.com/khyallin/shardkv/config"
)

// MultiDimensionRebalancer 多维度均衡器
// 综合考虑QPS、成功率和延迟的加权评分，选择评分最低和最高的组进行分片迁移
// 权重配置: QPS(40%) + 延迟(35%) + 失败率(25%)
type MultiDimensionRebalancer struct {
	qpsWeight     float64
	latencyWeight float64
	failureWeight float64
	initialized   bool
}

const (
	defaultQpsWeight     = 0.40
	defaultLatencyWeight = 0.35
	defaultFailureWeight = 0.25
)

// New 创建一个可配置权重的多维度均衡器。
// 不传参数时使用默认权重；传入 3 个值时依次表示 QPS、延迟、失败率权重。
func New(weights ...float64) *MultiDimensionRebalancer {
	r := &MultiDimensionRebalancer{
		qpsWeight:     defaultQpsWeight,
		latencyWeight: defaultLatencyWeight,
		failureWeight: defaultFailureWeight,
	}
	if len(weights) == 3 {
		r.qpsWeight = weights[0]
		r.latencyWeight = weights[1]
		r.failureWeight = weights[2]
	}
	r.initialized = true
	return r
}

func (m *MultiDimensionRebalancer) ensureWeights() {
	if m.initialized {
		return
	}
	if m.qpsWeight == 0 && m.latencyWeight == 0 && m.failureWeight == 0 {
		m.qpsWeight = defaultQpsWeight
		m.latencyWeight = defaultLatencyWeight
		m.failureWeight = defaultFailureWeight
	}
	m.initialized = true
}

// Score 计算组的负载评分，分数越高表示负载越重
// QPS贡献: 40%
// MaxLatency贡献: 35%
// 失败率贡献: 25%
func (m *MultiDimensionRebalancer) Score(status *GroupRunningStatus) float64 {
	m.ensureWeights()

	if status.DoneQPS == 0 {
		return 0
	}

	// 标准化QPS (0-1，通过相对值)
	qpsScore := status.TotalQPS / math.Max(1, status.TotalQPS)

	// 标准化延迟 (0-1，使用对数标度避免outlier影响过大)
	latencyMs := float64(status.MaxLatency.Milliseconds())
	latencyScore := math.Log1p(latencyMs) / math.Log1p(10000) // 假设10秒为参考延迟

	// 失败率 (0-1)
	failureRate := (status.DoneQPS - status.SuccessQPS) / math.Max(1, status.DoneQPS)

	// 加权求和
	return m.qpsWeight*qpsScore + m.latencyWeight*math.Min(1, latencyScore) + m.failureWeight*failureRate
}

func (m *MultiDimensionRebalancer) Rebalance(cfg *config.Config, groups map[int]*GroupRunningStatus) error {
	m.ensureWeights()

	if len(groups) <= 1 {
		return nil
	}

	// 计算每个组的评分
	scores := make(map[int]float64)
	var minScore, maxScore float64 = 1e8, -1.0
	var minGid, maxGid int

	for gid, status := range groups {
		score := m.Score(status)
		scores[gid] = score

		if score < minScore {
			minScore = score
			minGid = gid
		}
		if score > maxScore {
			maxScore = score
			maxGid = gid
		}
	}

	// 评分差异不明显时不迁移
	if maxScore > 0 && minScore > 0 && (maxScore-minScore)/minScore < 0.2 {
		return nil
	}

	if minGid == maxGid {
		return nil
	}

	// 从高分组迁移分片到低分组
	var shards []int
	for shard, gid := range cfg.Shards {
		if gid == config.Tgid(maxGid) {
			shards = append(shards, shard)
		}
	}

	if len(shards) == 0 {
		return nil
	}

	move := shards[rand.Intn(len(shards))]
	cfg.Shards[move] = config.Tgid(minGid)
	return nil
}

var _ Rebalancer = &MultiDimensionRebalancer{}
