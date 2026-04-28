# ShardKV Rebalancer 实现指南

本文档介绍了 ShardKV Dashboard 中的多种分片均衡器实现，参考了 Kubernetes、Elastic Search、Kafka 等产品的设计理念。

## 现有实现

### 1. NullRebalancer（空均衡器）
- **策略**: 不执行任何操作
- **场景**: 用于禁用自动均衡或调试
- **优点**: 无任何开销，方便人工控制
- **风险**: 系统负载可能严重不均

### 2. NumRebalancer（数量均衡器）
- **策略**: 基于分片数量均匀分配，调用 `cfg.Rebalance()`
- **场景**: 简单场景下的快速均衡
- **优点**: 实现简洁，分片数均等分配
- **限制**: 不考虑实际负载差异，可能均衡不合理

### 3. QpsRebalancer（吞吐量均衡器）【现有实现】
- **策略**: 找到最忙（最高QPS）和最闲（最低QPS）的两个组，随机迁移最忙组的一个分片
- **场景**: 通用负载均衡，大多数在线业务
- **优点**: 简单有效，能快速响应负载变化
- **限制**: 
  - 单次只迁移一个分片，均衡速度慢
  - 不考虑分片所属组是否为空（可能panic）
  - 不区分分片成本或业务优先级

---

## 新增实现

### 4. LatencyAwareRebalancer（延迟感知均衡器）

**设计参考**: Elastic Search 的 shard allocation awareness

```go
type LatencyAwareRebalancer struct{}
```

**策略**:
- 关注 `MaxLatency`，找延迟最高和最低的两个组
- 延迟差异 < 50% 时不迁移（避免频繁抖动）
- 将分片从高延迟组迁移到低延迟组

**权重计算**:
- 完全基于 `MaxLatency`，权重 100%

**适用场景**:
- 对延迟敏感的应用（金融、实时分析）
- 希望保证 P99 延迟的业务
- 某些服务器硬件性能差异大的场景

**优点**:
- 直观反映用户体验
- 有助于SLA达成

**缺点**:
- 忽视吞吐量，可能负载不均
- 延迟可能由其他因素引起（网络、GC），单纯迁移分片可能无效

---

### 5. MultiDimensionRebalancer（多维度均衡器）

**设计参考**: Kubernetes 的调度器、YARN 的资源调度

```go
type MultiDimensionRebalancer struct{}

Score = 0.40 * QpsScore + 0.35 * LatencyScore + 0.25 * FailureRateScore
```

**策略**:
- 综合考虑三个维度：
  - **吞吐量 (40%)**: `TotalQPS / maxQPS`
  - **延迟 (35%)**: `log1p(MaxLatency) / log1p(referenceLatency)` 
  - **失败率 (25%)**: `(DoneQPS - SuccessQPS) / DoneQPS`
- 计算每个组的负载评分
- 评分差异 < 20% 时不迁移
- 将分片从高分组迁移到低分组

**权重选择原因**:
- 吞吐量占比最高(40%)，是基础指标
- 延迟(35%)可反映系统压力和用户体验
- 失败率(25%)关乎可用性

**适用场景**:
- 综合性能要求的应用
- 负载特征复杂的系统（既关心吞吐，也关心延迟和稳定性）
- 希望获得较好平衡点的业务

**优点**:
- 全面考虑多维度指标
- 均衡效果最佳，稳定性好
- 权重可调整以适应不同业务

**缺点**:
- 权重设置需要调优
- 计算成本最高

---

### 6. GradualRebalancer（渐进式均衡器）

**设计参考**: Kubernetes 的控制器、Ceph 的 crush 算法

```go
type GradualRebalancer struct {
    maxMovePerRound    float64 // 默认 12.5% (1/8)
    imbalanceThreshold float64 // 默认 25%
}
```

**策略**:
- 单次只迁移 12.5% 的分片（假设共有8个分片，每次最多迁移1个）
- 负载不均衡比例 > 25% 时才触发迁移
- 确保源组至少保留1个分片
- 温和的迁移方式，避免过度变动

**参数说明**:
- `maxMovePerRound`: 控制迁移激进度
  - 0.125 (12.5%): 保守策略，适合生产环境
  - 0.25 (25%): 平衡策略
  - 0.5 (50%): 激进策略，快速收敛
- `imbalanceThreshold`: 触发迁移的条件
  - 0.25 (25%): 容忍较大偏差，减少迁移次数
  - 0.1 (10%): 更频繁的微调

**适用场景**:
- 生产环境的稳定性优先场景
- 分片迁移成本高的系统（数据庞大、迁移耗时）
- 希望平滑优化而不剧烈变动的业务

**优点**:
- 改变缓和，系统抖动少
- 降低迁移成本
- 可预测性好，便于容量规划

**缺点**:
- 均衡速度慢，可能需要多轮才能收敛
- 暂时性的不均衡可能存在较长时间

---

### 7. SuccessAwareRebalancer（可用性感知均衡器）

**设计参考**: Netflix 的混沌工程

```go
type SuccessAwareRebalancer struct{}
```

**策略**:
- 关注 `SuccessRate = SuccessQPS / DoneQPS`
- 成功率差异 > 1% 时触发迁移
- 将分片从低成功率组迁移到高成功率组

**权重计算**:
- 完全基于 `SuccessRate`，权重 100%

**适用场景**:
- 高可用性要求的系统（金融、医疗）
- 某些后端服务可靠性差异大
- 故障诊断和恢复阶段

**优点**:
- 直接关注可用性
- 能自动隔离故障节点
- 提升整体系统可用性

**缺点**:
- 忽视吞吐和延迟
- 可能反复迁移（成功率低可能是暂时的）
- 不适合作为常规均衡策略

---

## 使用对比表

| 均衡器 | 关注维度 | 均衡速度 | 系统抖动 | 实现复杂度 | 推荐场景 |
|--------|---------|---------|---------|-----------|---------|
| Null | - | - | - | 最低 | 调试、人工控制 |
| Num | 分片数 | 很快 | 高 | 低 | 简单均衡 |
| Qps | 吞吐量 | 快 | 中 | 低 | 通用场景 |
| LatencyAware | 延迟 | 中 | 中 | 低 | 延迟敏感 |
| MultiDimension | QPS+延迟+成功率 | 中 | 低 | 高 | 综合性能 |
| Gradual | 吞吐量(渐进) | 慢 | 最低 | 中 | 生产稳定性 |
| SuccessAware | 成功率 | 快 | 中 | 低 | 故障隔离 |

---

## 配置推荐

### 开发环境
```go
rebalancer: &rebalance.QpsRebalancer{}
```

### 测试环境
```go
rebalancer: rebalance.NewGradualRebalancer()
// 调整参数
gradual := rebalance.NewGradualRebalancer()
gradual.maxMovePerRound = 0.25      // 更激进
gradual.imbalanceThreshold = 0.1    // 更敏感
```

### 生产环境 - 通用
```go
rebalancer: rebalance.NewGradualRebalancer()
```

### 生产环境 - 高性能
```go
rebalancer: &rebalance.MultiDimensionRebalancer{}
```

### 生产环境 - 高可用
```go
rebalancer: &rebalance.SuccessAwareRebalancer{}
```

---

## 集成方式

在 `internal/service/config.go` 中修改：

```go
func NewConfigService() *ConfigService {
	skv := shardkv.New()
	s := &ConfigService{
		skv:        skv,
		ctrler:     shardkv.MakeCtrler(),
		client:     shardkv.MakeClient(),
		groups:     make([]*shardkv.Group, 0),
		rebalancer: rebalance.NewGradualRebalancer(), // 选择需要的实现
	}
	s.setup()
	return s
}
```

---

## 注意事项

1. **性能监测**: 观察 `Rebalance()` 的执行时间和频率，避免过度迁移

2. **指标污染**: 如现有注记所示，迁移中的分片流量会污染指标，下一轮决策可能基于错误数据

3. **收敛保证**: 
   - `Qps` 和 `LatencyAware`: 最坏情况下需要 N-1 轮（N为组数）
   - `Gradual`: 可能需要更多轮，但更稳定
   - `MultiDimension`: 接近最优但不保证全局最优

4. **幂等性**: 所有实现都是幂等的，可安全重复执行

5. **自定义扩展**: 参考现有实现，可轻松创建新的 Rebalancer

---

## 性能特性

根据现有代码分析：

```
Ticker 周期: 10秒
Rebalance 操作: O(N + S)，其中 N=组数, S=分片数
迁移单位: 1个分片 (Qps, LatencyAware, SuccessAware)
         多个分片 (Gradual, MultiDimension)
```

---

## 扩展建议

可根据需要实现的其他均衡器：

1. **CostAwareRebalancer**: 考虑分片大小和迁移成本
2. **TimeWindowRebalancer**: 根据时间窗口的历史负载预测
3. **ConstraintBasedRebalancer**: 支持亲和性约束
4. **AdaptiveRebalancer**: 自动调整权重的自适应均衡器
