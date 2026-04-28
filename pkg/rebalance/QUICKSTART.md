# ShardKV Rebalancer 快速开始

## 文件清单

| 文件 | 说明 |
|------|------|
| `api.go` | Rebalancer 接口定义和 GroupRunningStatus 结构 |
| `null.go` | 空均衡器（无操作） |
| `num.go` | 数量均衡器（基于分片数） |
| `qps.go` | QPS 均衡器（基于吞吐量）【原有实现】 |
| `latency.go` | 延迟感知均衡器（基于最大延迟） |
| `multidim.go` | 多维度均衡器（QPS + 延迟 + 成功率） |
| `gradual.go` | 渐进式均衡器（温和迁移，避免系统抖动） |
| `success_aware.go` | 可用性感知均衡器（基于成功率，用于故障隔离） |
| `rebalancer_test.go` | 单元测试和示例 |
| `README.md` | 详细设计文档 |

## 最小化集成

### 1. 修改 ConfigService 初始化

在 `internal/service/config.go` 中修改 `NewConfigService()` 函数：

**原始代码：**
```go
func NewConfigService() *ConfigService {
	skv := shardkv.New()
	s := &ConfigService{
		skv:        skv,
		ctrler:     shardkv.MakeCtrler(),
		client:     shardkv.MakeClient(),
		groups:     make([]*shardkv.Group, 0),
		rebalancer: &rebalance.QpsRebalancer{},  // ← 原有
	}
	s.setup()
	return s
}
```

**修改为（选择一种）：**

#### 选项A: 延迟感知（延迟敏感应用）
```go
rebalancer: &rebalance.LatencyAwareRebalancer{},
```

#### 选项B: 多维度均衡（综合性能优化）
```go
rebalancer: &rebalance.MultiDimensionRebalancer{},
```

#### 选项C: 渐进式均衡（生产环境推荐）
```go
rebalancer: rebalance.NewGradualRebalancer(),
```

#### 选项D: 可用性感知（故障隔离）
```go
rebalancer: &rebalance.SuccessAwareRebalancer{},
```

### 2. 运行测试验证

```bash
cd /Users/kai/code/shardkv-dashboard
go test -v ./pkg/rebalance/
```

**预期输出：**
```
=== RUN   TestQpsRebalancer
    rebalancer_test.go:52: After QpsRebalancer: Group2 has 5 shards (expected 5)
--- PASS: TestQpsRebalancer (0.00s)
=== RUN   TestLatencyAwareRebalancer
    rebalancer_test.go:93: After LatencyAwareRebalancer: Group2 has 5 shards (expected 5)
--- PASS: TestLatencyAwareRebalancer (0.00s)
...
PASS
ok      github.com/khyallin/shardkv-dashboard/pkg/rebalance     0.989s
```

## 性能参数调优

### GradualRebalancer 参数调整

```go
import "github.com/khyallin/shardkv-dashboard/pkg/rebalance"

// 保守策略（生产环境）
r := rebalance.NewGradualRebalancer()
r.maxMovePerRound = 0.125        // 每轮最多迁移 12.5%
r.imbalanceThreshold = 0.25      // 负载差异 > 25% 才迁移

// 平衡策略（测试环境）
r := rebalance.NewGradualRebalancer()
r.maxMovePerRound = 0.25         // 每轮最多迁移 25%
r.imbalanceThreshold = 0.15      // 负载差异 > 15% 才迁移

// 激进策略（快速优化）
r := rebalance.NewGradualRebalancer()
r.maxMovePerRound = 0.5          // 每轮最多迁移 50%
r.imbalanceThreshold = 0.1       // 负载差异 > 10% 就迁移
```

## 实际使用对比

### 场景1: 简单 Web 应用（推荐 QpsRebalancer）
- 负载相对均匀
- 不关心延迟细节
- 当前实现已足够

### 场景2: 实时分析（推荐 LatencyAwareRebalancer）
- 关注 P99 延迟
- 需要保证响应速度
- 用户体验敏感

```go
rebalancer: &rebalance.LatencyAwareRebalancer{},
```

### 场景3: 金融交易系统（推荐 MultiDimensionRebalancer）
- 吞吐、延迟、成功率都重要
- 对系统稳定性要求高
- 需要综合平衡

```go
rebalancer: &rebalance.MultiDimensionRebalancer{},
```

### 场景4: 关键服务生产环境（推荐 GradualRebalancer）
- 尽可能减少系统抖动
- 避免级联迁移
- 平稳优化

```go
rebalancer: rebalance.NewGradualRebalancer()
```

### 场景5: 服务故障恢复（推荐 SuccessAwareRebalancer）
- 某个后端服务出现故障
- 需要快速隔离故障节点
- 保证整体可用性

```go
rebalancer: &rebalance.SuccessAwareRebalancer{},
```

## 监控指标建议

在集成新的 Rebalancer 后，建议监控以下指标：

```
1. 每轮迁移的分片数（分析迁移频率）
2. 迁移前后的负载标准差（评估均衡效果）
3. Rebalance() 方法耗时（性能开销）
4. 各组的 QPS、延迟、成功率变化趋势
5. 系统的整体吞吐和延迟变化
```

## 故障排查

### 问题1: 迁移频繁导致系统抖动
**原因**: 使用 QpsRebalancer 或 LatencyAwareRebalancer，且阈值设置过低
**解决**: 
- 切换到 GradualRebalancer
- 增加阈值判断

### 问题2: 无法均衡成功率低的节点
**原因**: 未使用 SuccessAwareRebalancer
**解决**:
```go
rebalancer: &rebalance.SuccessAwareRebalancer{},
```

### 问题3: 负载指标污染
**原因**: 迁移过程中分片指标被统计进去（已知问题）
**解决**: 
- 在迁移期间过滤掉迁移流量
- 使用更长的时间窗口平滑指标

## 演进路径

1. **第一阶段**: 使用 QpsRebalancer（当前）
2. **第二阶段**: 根据场景选择合适的 Rebalancer
3. **第三阶段**: 自定义 Rebalancer 满足特定需求
4. **第四阶段**: 多策略混合（不同时段用不同策略）

## 扩展示例

如果需要实现自定义的 Rebalancer，参考以下模板：

```go
package rebalance

import "github.com/khyallin/shardkv/config"

// CustomRebalancer 自定义均衡器
type CustomRebalancer struct {
	// 添加你的参数
}

func (r *CustomRebalancer) Rebalance(cfg *config.Config, groups map[int]*GroupRunningStatus) error {
	// 1. 验证输入
	if len(groups) <= 1 {
		return nil
	}

	// 2. 计算指标（使用 GroupRunningStatus 的字段）
	// - TotalQPS: 总吞吐
	// - SuccessQPS: 成功吞吐
	// - DoneQPS: 完成吞吐
	// - MaxLatency: 最大延迟
	// - AvgLatency: 平均延迟

	// 3. 决定迁移目标
	// - minGid: 目标组（负载最低）
	// - maxGid: 源组（负载最高）

	// 4. 选择分片迁移
	// var shards []int
	// for shard, gid := range cfg.Shards {
	//     if gid == config.Tgid(maxGid) {
	//         shards = append(shards, shard)
	//     }
	// }
	// if len(shards) > 0 {
	//     move := shards[你的选择策略]
	//     cfg.Shards[move] = config.Tgid(minGid)
	// }

	return nil
}

var _ Rebalancer = &CustomRebalancer{}
```

---

## 总结

| 特性 | Qps | Latency | MultiDim | Gradual | SuccessAware |
|------|-----|---------|----------|---------|--------------|
| 关注维度 | 吞吐 | 延迟 | 全面 | 吞吐(渐进) | 成功率 |
| 均衡速度 | 快 | 中 | 中 | 慢 | 快 |
| 系统抖动 | 中 | 中 | 低 | 最低 | 中 |
| 推荐场景 | 通用 | 延迟敏感 | 综合性能 | 生产稳定 | 故障隔离 |
| 实现复杂度 | 低 | 低 | 高 | 中 | 低 |
