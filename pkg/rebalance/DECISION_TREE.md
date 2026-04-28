# Rebalancer 选择决策树

基于你的具体业务场景，按照以下决策树快速选择合适的 Rebalancer。

## 场景识别

### 问题 1: 你最关心什么？

```
       ┌─────────────────────────────┐
       │   你的系统最关心什么？      │
       └──────────────┬──────────────┘
                      │
        ┌─────────────┼─────────────┐
        │             │             │
        ▼             ▼             ▼
    吞吐量(QPS)   响应延迟      可用性/成功率
        │             │             │
        │             │             │
    ┌───┴──┐      ┌──┴──┐      ┌──┴──────┐
    │      │      │     │      │         │
    ▼      ▼      ▼     ▼      ▼         ▼
   简单   复杂   高    中等   故障  大故障
   系统   业务   要求  要求   隔离  隔离
    │      │      │     │      │     │
    ▼      ▼      ▼     ▼      ▼     ▼
  Qps   MultiDim Lat  Gradual Succ  Succ
  Rebal  Rebal    Aware Rebal Aware Aware
```

## 详细决策流程

### 流程 A: 吞吐量优先

**特征**:
- 关注 TotalQPS
- 希望最大化系统吞吐
- 用户体验对延迟不敏感

**判断点**:

```
吞吐量优先?
  ├─ YES
  │   ├─ 简单系统(1-2台服务器)?
  │   │   └─ YES → QpsRebalancer ✓
  │   │
  │   └─ 复杂系统(多维度约束)?
  │       └─ YES → MultiDimensionRebalancer(权重调整)
  │
  └─ NO → 继续到流程 B
```

**推荐**: `&rebalance.QpsRebalancer{}`

**示例行业**:
- 日志处理系统
- 批量数据处理
- 简单缓存层

---

### 流程 B: 延迟优先

**特征**:
- P99 延迟是关键指标
- SLA 对延迟有要求
- 用户体验对速度敏感

**判断点**:

```
延迟是否为关键指标?
  ├─ YES
  │   ├─ 成功率也很重要?
  │   │   ├─ YES → MultiDimensionRebalancer ✓✓
  │   │   └─ NO → LatencyAwareRebalancer ✓
  │   │
  │   └─ 需要稳定性优先(生产环)?
  │       └─ YES → GradualRebalancer(调参)
  │
  └─ NO → 继续到流程 C
```

**推荐**:
- 快速响应: `&rebalance.LatencyAwareRebalancer{}`
- 综合性能: `&rebalance.MultiDimensionRebalancer{}`
- 稳定优先: `rebalance.NewGradualRebalancer()`

**示例行业**:
- 实时分析系统（BI/报表）
- 金融交易查询
- 搜索引擎

---

### 流程 C: 可用性优先

**特征**:
- 成功率/可用性是首要指标
- 故障隔离能力重要
- 系统需要 99.9% 以上可用性

**判断点**:

```
可用性/成功率是否首要指标?
  ├─ YES
  │   ├─ 某些服务故障概率高?
  │   │   ├─ YES → SuccessAwareRebalancer ✓✓
  │   │   └─ NO → MultiDimensionRebalancer(提高失败率权重)
  │   │
  │   └─ 需要保持稳定(避免级联)?
  │       └─ YES → GradualRebalancer
  │
  └─ NO → 需要平衡多个维度
```

**推荐**:
- 故障隔离: `&rebalance.SuccessAwareRebalancer{}`
- 全面保障: `&rebalance.MultiDimensionRebalancer{}`
- 平稳优化: `rebalance.NewGradualRebalancer()`

**示例行业**:
- 金融交易系统
- 医疗信息系统
- 关键业务系统

---

## 场景-方案映射表

| # | 场景描述 | 系统类型 | 推荐方案 | 替代方案 | 调参建议 |
|---|--------|--------|--------|--------|--------|
| 1 | 简单Web应用，不关心延迟 | 中小型 | QpsRebalancer | MultiDim | - |
| 2 | 电商平台，全面性能 | 大型 | MultiDimensionRebalancer | GradualRebalancer | 权重可调 |
| 3 | BI/报表系统，延迟敏感 | 中型 | LatencyAwareRebalancer | MultiDim | 阈值0.3 |
| 4 | 金融交易，高可用 | 大型 | SuccessAwareRebalancer | MultiDim | 阈值0.005 |
| 5 | 生产环境，稳定优先 | 任意 | GradualRebalancer | MultiDim | move=0.125, threshold=0.25 |
| 6 | 测试/开发环境 | 任意 | QpsRebalancer | GradualRebalancer(move=0.25) | - |
| 7 | 故障恢复中 | 任意 | SuccessAwareRebalancer | MultiDim | 临时使用 |
| 8 | 迁移成本高 | 任意 | GradualRebalancer | MultiDim | move=0.0625 |
| 9 | 多cloud环境 | 大型 | MultiDimensionRebalancer | GradualRebalancer | 区域感知 |
| 10 | 容器编排集群 | 云原生 | GradualRebalancer | MultiDim | 参考K8s参数 |

---

## 按环境选择

### 开发环境 (Dev)
```go
rebalancer: &rebalance.QpsRebalancer{}
// 或用NullRebalancer手工测试
```
**原因**: 快速反馈，简单场景

### 预发布环境 (Staging)
```go
rebalancer: rebalance.NewGradualRebalancer()
// 参数: maxMovePerRound=0.25, imbalanceThreshold=0.1
```
**原因**: 测试不同策略的效果

### 生产环境 (Prod) - 标准配置
```go
rebalancer: rebalance.NewGradualRebalancer()
// 参数: maxMovePerRound=0.125, imbalanceThreshold=0.25 (默认)
```
**原因**: 最稳定，系统抖动最小

### 生产环境 (Prod) - 性能优化
```go
rebalancer: &rebalance.MultiDimensionRebalancer{}
// 权重: Qps(40%) + Latency(35%) + FailureRate(25%)
```
**原因**: 综合性能最优

### 生产环境 (Prod) - 故障恢复
```go
rebalancer: &rebalance.SuccessAwareRebalancer{}
// 临时用于故障期间
```
**原因**: 快速隔离故障节点

---

## 性能和稳定性 vs 响应速度

```
响应速度 ▲
        │     SuccessAware (51.67ns)
        │         ↓
        │     LatencyAware (38.97ns) ─────────────┐
        │         ↓                                │
        │     QpsRebalancer (38.57ns)            │
        │                                         │
        │     MultiDim (74.64ns)                 │
        │         ↓                              │
        │     Gradual (79.81ns) ────────────────────
        │
        └─────────────────────────────────────────► 
          稳定性/成本
                低 ───────────────────────────► 高

四象限分析:
┌─ 快速响应 + 高稳定性 (理想)
│  ├─ MultiDimensionRebalancer
│  └─ LatencyAwareRebalancer
│
├─ 快速响应 + 低成本
│  ├─ QpsRebalancer
│  └─ SuccessAwareRebalancer
│
├─ 缓慢但高稳定性
│  ├─ GradualRebalancer (生产推荐)
│  └─ NullRebalancer (人工控制)
│
└─ 其他位置 (权衡场景)
   └─ 根据权重调整
```

---

## 迁移指南

### 从 QpsRebalancer 迁移到新方案

#### 场景 1: 想要降低系统抖动

```go
// 旧
rebalancer: &rebalance.QpsRebalancer{}

// 新
rebalancer: rebalance.NewGradualRebalancer()
// 单轮只迁移12.5%，系统更稳定
```

#### 场景 2: 想要改善用户体验（延迟）

```go
// 旧
rebalancer: &rebalance.QpsRebalancer{}

// 新 - 选项A (单纯看延迟)
rebalancer: &rebalance.LatencyAwareRebalancer{}

// 新 - 选项B (综合考虑，推荐)
rebalancer: &rebalance.MultiDimensionRebalancer{}
```

#### 场景 3: 处理故障导致的成功率下降

```go
// 旧
rebalancer: &rebalance.QpsRebalancer{}

// 新 - 临时
rebalancer: &rebalance.SuccessAwareRebalancer{}
// 待故障恢复后改回

// 新 - 永久
rebalancer: &rebalance.MultiDimensionRebalancer{}
// 综合考虑成功率
```

#### 场景 4: 迁移成本高，需要缓慢调整

```go
// 旧
rebalancer: &rebalance.QpsRebalancer{}

// 新
rebalancer := rebalance.NewGradualRebalancer()
rebalancer.maxMovePerRound = 0.0625  // 6.25%, 更保守
```

---

## 监控告警建议

集成新 Rebalancer 后，建议建立以下监控:

```
Rebalancer 迁移频率
  ├─ 正常: 10-50% 分片每天迁移
  ├─ 告警(过高): > 70% 分片每天迁移 → 考虑提高threshold
  └─ 告警(过低): < 5% 分片每天迁移 → 可能策略太保守

各组负载标准差
  ├─ 好: σ < avg * 0.2 (20% 以内)
  ├─ 可接受: σ < avg * 0.3
  └─ 告警: σ > avg * 0.5 → 均衡失效

单次迁移时间
  ├─ 良好: < 1秒
  ├─ 可接受: < 5秒
  └─ 告警: > 10秒 → 迁移性能问题

Rebalance() 调用耗时
  ├─ 良好: < 100ms
  ├─ 可接受: < 500ms
  └─ 告警: > 1s → 影响服务响应
```

---

## 常见问题

**Q: 我应该使用哪一个?**
A: 从 QpsRebalancer 开始，根据实际监控数据选择：
- 延迟问题 → LatencyAwareRebalancer
- 全面优化 → MultiDimensionRebalancer
- 太频繁迁移 → GradualRebalancer
- 故障期间 → SuccessAwareRebalancer

**Q: 可以动态切换 Rebalancer 吗?**
A: 当前实现不支持动态切换。建议在运维界面添加配置选项，然后重启服务应用新策略。

**Q: 多久评估一次效果?**
A: 建议运行 7 天观察各项指标趋势，然后根据 SLA 目标调整。迁移成本高的场景可延长到 14-30 天。

**Q: 能组合使用吗?**
A: 可以实现 HybridRebalancer，根据时间/条件切换策略（高峰期用激进，低峰期用保守）。

**Q: 为什么 GradualRebalancer 比 QpsRebalancer 慢?**
A: GradualRebalancer 引入了额外的阈值检查和多轮迭代逻辑，但性能差异很小(79.81ns vs 38.57ns，仍是纳秒级)。收益是稳定性大幅提升。

---

## 总结

| 优先级 | 场景 | 首选 | 备选 |
|--------|------|------|------|
| 🥇 首要 | 生产环境稳定性 | GradualRebalancer | NullRebalancer(人工) |
| 🥈 次要 | 综合性能优化 | MultiDimensionRebalancer | LatencyAwareRebalancer |
| 🥉 三级 | 快速故障隔离 | SuccessAwareRebalancer | MultiDimensionRebalancer |
| 特殊 | 简单系统/测试 | QpsRebalancer | NullRebalancer |

**推荐路径**: QpsRebalancer → GradualRebalancer → MultiDimensionRebalancer
