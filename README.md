# ShardKV Dashboard

<div align="center">

[![Go Version](https://img.shields.io/badge/Go-1.25+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/Docker-Ready-brightgreen.svg)](Dockerfile)

**🎯 分布式键值存储的可视化管理平台** | **实时监控** | **性能压测** | **一键部署**

[**English**](README.en.md) | [**快速开始**](#快速开始) | [**功能演示**](#核心功能) | [**API文档**](#api-文档)

</div>

---

## ✨ 概述

**ShardKV Dashboard** 是一个**企业级分布式系统管理工具**，为 ShardKV（分片键值存储）集群提供完整的**可视化管理、实时监控、数据操作和性能测试**能力。

无论你是系统管理员、性能测试工程师还是分布式系统研究者，ShardKV Dashboard 都能让复杂的集群管理变得简单直观。

### 💡 为什么选择 ShardKV Dashboard？

| 特点 | 说明 |
|------|------|
| 🎨 **零学习曲线** | 直观的Web界面，拖拽式操作，无需命令行 |
| 📊 **实时洞察** | 毫秒级监控集群QPS、延迟、成功率 |
| 🔧 **灵活管理** | 动态创建/停止分片组、智能分片迁移 |
| ⚡ **性能评估** | 内置多维度压测工具，支持自定义场景 |
| 🐳 **开箱即用** | Docker/Docker Compose 一键启动 |
| 📈 **专业分析** | JSON Lines 输出 + Python 绘图脚本 |

---

## 🚀 快速开始

### 前置要求

- **Go 1.25+** 
- **Docker 29.2+**
- **Python 3.8+**（可选，用于分析压测结果）

### 使用 MakeFile（推荐）

最简单的方式，一行命令启动完整环境：

```bash
# 1. 克隆仓库
git clone https://github.com/khyallin/shardkv-dashboard.git
cd shardkv-dashboard

# 2. 启动所有服务
make all

# 3. 打开浏览器
open http://localhost:8080
```

### 使用 Docker

```bash
# 1. 拉取镜像
docker pull khyal/shardkv-dashboard

# 2. 启动服务
docker run -d --name shardkv-dashboard --network shardkv-net -v /var/run/docker.sock:/var/run/docker.sock -p 8080:8080 khyal/shardkv-dashboard
```
就这样！Dashboard 已经在 `http://localhost:8080` 运行了。

## 📚 核心功能

### 1. 🗂️ 集群管理

**直观的分片组管理** - 可视化编辑集群拓扑

```
集群概览 → 创建分片组 → 拖拽迁移分片 → 性能监控
   ↓
实时显示：
  • 分片分布
  • 每组的 QPS、成功率、延迟
  • 自动负载均衡状态
```

**支持操作：**
- ✅ 动态创建新的分片组
- ✅ 拖拽分片到目标分片组（即时生效）
- ✅ 优雅停止分片组（无数据丢失）
- ✅ 自动负载均衡（启用后每10秒检查一次）

### 2. 📊 性能监控

**实时指标看板** - 一目了然的集群状态

```json
{
  "total_qps": 15420,        // 总吞吐量
  "done_qps": 14890,          // 已完成吞吐量
  "success_qps": 14850,       // 成功吞吐量
  "avg_latency": 2.3,         // 平均延迟（ms）
  "max_latency": 45.8         // 最大延迟（ms）
}
```

**监控维度：**
- 集群整体QPS和成功率
- 每个分片组的性能指标
- 实时延迟统计（最小/平均/最大）
- 错误分类统计

### 3. 🔑 键值操作

**Web 表单式数据管理** - 无需编写代码操作数据

**Get 操作：**
```
输入 Key → 立即查询 → 显示：
  • Value 值
  • Value 类型（string/int/float/bool）
  • Version 版本号
```

**Put 操作：**
```
填写表单：
  • Key：数据键
  • Type：选择类型（string/int/float/bool）
  • Value：输入值（自动类型校验）
  • Version：指定版本（可选）
```

**支持的数据类型：**
| 类型 | 示例 | 说明 |
|------|------|------|
| `string` | `"hello world"` | 字符串 |
| `int` | `42` | 整数（-9223372036854775808 ~ 9223372036854775807） |
| `float` | `3.14` | 浮点数（无 Inf/NaN） |
| `bool` | `true/false` | 布尔值 |

### 4. ⚡ 性能压测

**企业级压力测试** - 多维度评估系统性能

**压测维度：**
```
并发数: 1, 4, 16, 32, 64 ...
读写比: 0% (纯写), 50% (混合), 100% (纯读)
数据大小: 16B, 64B, 256B, 1KB, 4KB ...
```

**测试结果包含：**
```json
{
  "duration_sec": 60.0,
  "total_ops": 923520,
  "success_ops": 920145,
  "failed_ops": 3375,
  "tps": 15392.0,
  "success_tps": 15335.75,
  "avg_latency_ms": 3.2,
  "max_latency_ms": 156.8,
  "errors": {
    "wrong_leader": 1200,
    "wrong_group": 1850,
    "version_error": 325,
    "other": 0
  }
}
```

**内置脚本分析：**
```bash
# 运行压测并导出结果
scripts/stress.sh

# 使用 Python 绘制性能曲线
python3 scripts/draw.py tmp/stress_results.jsonl
```

---

## 🏗️ 架构设计

### 系统架构

```
┌───────────────────────────────────────────────────────────────────────┐
│                           ShardKV Dashboard                           │
├───────────────────────────────────────────────────────────────────────┤
│ Frontend: web/index.html (HTML/CSS/JS)                               │
│  - 拖拽分片、集群管理、KV 读写、压测配置                                   │
└───────────────────────────────┬───────────────────────────────────────┘
                │ HTTP / JSON
                ▼
┌───────────────────────────────────────────────────────────────────────┐
│ API 层: Gin Router + Handler                                          │
│  - 路由注册: internal/app/router.go                                   │
│  - 处理器: internal/handler/*.go                                      │
│  - 端点: /api/v1/group/*, /api/v1/config*, /api/v1/kv/*, /stress/run │
└───────────────────────────────┬───────────────────────────────────────┘
                │ 调用业务服务
                ▼
┌───────────────────────────────────────────────────────────────────────┐
│ Service 层                                                            │
│  ┌───────────────────────┐  ┌─────────────────────┐  ┌──────────────┐ │
│  │ ConfigService         │  │ KVService           │  │ StressService│ │
│  │ - 创建/停止分片组        │  │ - Get/Put 类型校验      │  │ - 并发压测      │ │
│  │ - 手动迁移与自动均衡      │  │ - 版本控制写入          │  │ - 指标与错误分解 │ │
│  │ - 查询 group status     │  │                       │  │              │ │
│  └──────────────┬────────┘  └───────────┬─────────┘  └──────┬───────┘ │
└─────────────────┼────────────────────────┼─────────────────────┼─────────┘
          │                        │                     │
          │ 统一通过 pkg/shardkv 适配层创建 client/ctrler/group
          ▼                        ▼                     ▼
┌───────────────────────────────────────────────────────────────────────┐
│ Adapter 层: pkg/shardkv + pkg/docker                                  │
│  - shardkv.New(): Docker SDK 拉取镜像并管理容器                         │
│  - MakeCtrler()/MakeClient(): 连接 shardkv controller/client           │
│  - MakeGroup()/RunGroup()/StopGroup(): 编排分片组容器生命周期              │
└───────────────────────────────┬───────────────────────────────────────┘
                │ RPC + 容器网络(shardkv-net)
                ▼
┌───────────────────────────────────────────────────────────────────────┐
│ 被管理集群: ShardKV                                                    │
│  - Controller (GID 0) 维护配置与迁移                                     │
│  - Shard Groups (GID >= 1) 提供 KV 服务                                 │
│  - 内部基于 Raft + RSM + StateMachine                                   │
└───────────────────────────────────────────────────────────────────────┘
```

### API 端点概览

| 端点 | 方法 | 功能 | 身份 |
|------|------|------|------|
| `/ping` | GET | 健康检查 | ✅ Public |
| `/` | GET | Web 页面 | ✅ Public |
| **集群管理** | | | |
| `/api/v1/group/get` | POST | 获取集群配置 | 🔒 App |
| `/api/v1/group/status` | POST | 查询分片组状态 | 🔒 App |
| `/api/v1/group/create` | POST | 创建分片组 | 🔒 App |
| `/api/v1/group/stop` | POST | 停止分片组 | 🔒 App |
| **配置操作** | | | |
| `/api/v1/config` | POST | 手动迁移分片 | 🔒 App |
| `/api/v1/config/auto` | POST | 自动均衡开关 | 🔒 App |
| **数据操作** | | | |
| `/api/v1/kv/get` | POST | 查询键值 | 🔒 App |
| `/api/v1/kv/put` | POST | 设置键值 | 🔒 App |
| **压力测试** | | | |
| `/api/v1/stress/run` | POST | 运行压测 | 🔒 App |

### 响应格式

所有 API 响应遵循统一格式：

```json
{
  "code": 0,                  // 0=成功, 非0=失败
  "message": "success",       // 操作信息
  "data": { ... }             // 业务数据（可选）
}
```

**错误响应示例：**
```json
{
  "code": -1,
  "message": "failed to create group: timeout",
  "data": null
}
```

---

## 📖 API 文档

### 获取集群配置

**请求**
```bash
curl -X POST http://localhost:8080/api/v1/group/get \
  -H "Content-Type: application/json" \
  -d '{}'
```

**响应**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "num": 4,
    "shards": [0, 1, 2, 3],
    "groups": {
      "1": [0, 1],
      "2": [2, 3]
    }
  }
}
```

---

### 查询分片组状态

**请求**
```bash
curl -X POST http://localhost:8080/api/v1/group/status \
  -H "Content-Type: application/json" \
  -d '{"gid": 1}'
```

**响应**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "total_qps": 15420,
    "done_qps": 14890,
    "success_qps": 14850,
    "max_latency": 45.8,
    "avg_latency": 2.3
  }
}
```

---

### 键值查询 (Get)

**请求**
```bash
curl -X POST http://localhost:8080/api/v1/kv/get \
  -H "Content-Type: application/json" \
  -d '{"key": "user_123"}'
```

**响应**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "type": "string",
    "value": "John Doe",
    "version": 5
  }
}
```

---

### 键值设置 (Put)

**请求**
```bash
curl -X POST http://localhost:8080/api/v1/kv/put \
  -H "Content-Type: application/json" \
  -d '{
    "key": "user_123",
    "type": "string",
    "value": "Jane Smith",
    "version": 5
  }'
```

**响应**
```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

---

### 运行压力测试

**请求**
```bash
curl -X POST http://localhost:8080/api/v1/stress/run \
  -H "Content-Type: application/json" \
  -d '{
    "duration_sec": 60,
    "concurrency": 32,
    "read_ratio": 0.5,
    "value_size": 1024,
    "target_gid": 1,
    "key_prefix": "stress_test"
  }'
```

**响应**
```json
{
  "code": 0,
  "message": "success",
  "data": {
    "result": {
      "duration_sec": 60.0,
      "total_ops": 923520,
      "success_ops": 920145,
      "tps": 15392.0,
      "success_tps": 15335.75,
      "avg_latency_ms": 3.2,
      "max_latency_ms": 156.8,
      "errors": {
        "wrong_leader": 1200,
        "wrong_group": 1850,
        "version_error": 325
      }
    },
    "before_status": { ... },
    "after_status": { ... }
  }
}
```

---

## 🛠️ 配置说明

### 环境变量

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `DASHBOARD_PORT` | `8080` | Dashboard 服务端口 |
| `SHARDKV_ADDR` | `localhost:2379` | ShardKV 服务地址 |
| `DOCKER_HOST` | `unix:///var/run/docker.sock` | Docker daemon 地址 |

### 配置文件

编辑 `config/default.go` 调整默认参数：

```go
// 自动均衡检查间隔
AutoRebalanceCheckInterval: 10 * time.Second

// 压测超时时间
StressTimeout: 120 * time.Second

// 最大并发数
MaxConcurrency: 256
```

---

## 📊 使用示例

### 场景 1：集群部署和初始化

```bash
# 1. 启动 Dashboard
make all

# 2. 打开浏览器访问
open http://localhost:8080

# 3. 在 Web 界面创建分片组（点击 + 按钮）

# 4. 拖拽分片进行分配

# 5. 启用自动负载均衡
```

### 场景 2：性能基准测试

```bash
# 1. 运行内置压测脚本
make stress

# 2. 查看测试结果
cat ../tmp/stress_results.jsonl | head -20

# 3. 生成性能曲线图
make draw
```

### 场景 3：数据验证

```bash
# 通过 Web UI 执行查询和设置操作，验证数据正确性

# 也可使用 curl
curl -X POST http://localhost:8080/api/v1/kv/put \
  -H "Content-Type: application/json" \
  -d '{"key":"test","type":"string","value":"hello"}'

curl -X POST http://localhost:8080/api/v1/kv/get \
  -H "Content-Type: application/json" \
  -d '{"key":"test"}'
```

---

## 🔧 开发指南

### 项目结构

```
shardkv-dashboard/
├── cmd/
│   ├── main.go              # 程序入口
│   └── server.go            # 服务配置
├── internal/
│   ├── app/                 # 应用初始化
│   │   ├── app.go
│   │   └── router.go
│   ├── handler/             # HTTP 处理器（11 个端点）
│   │   ├── ping.go
│   │   ├── configget.go
│   │   ├── groupcreate.go
│   │   ├── kvget.go
│   │   └── ...
│   └── service/             # 业务逻辑
│       ├── config.go        # 集群管理
│       ├── kv.go            # 键值操作
│       └── stress.go        # 压力测试
├── pkg/
│   ├── shardkv/             # ShardKV 适配层
│   └── docker/              # Docker 集成
├── config/
│   ├── config.go
│   └── default.go
├── web/
│   └── index.html           # 前端 (3000+ 行)
├── scripts/
│   ├── stress.sh            # 压测脚本
│   └── draw.py              # 结果分析
├── Makefile
├── Dockerfile
└── go.mod
```

### 构建和运行

```bash
# 查看所有可用命令
make help

# 首次启动服务
make all

# 重启服务
make restart

# 构建 Docker 镜像
make build

# 性能基准测试
make bench
```

### 代码贡献流程

1. **Fork 项目**
   ```bash
   git clone https://github.com/khyallin/shardkv-dashboard.git
   cd shardkv-dashboard
   ```

2. **创建特性分支**
   ```bash
   git switch -c feature/your-feature-name
   ```

3. **提交更改**
   ```bash
   git commit -am "feat: add your feature"
   ```

4. **推送到 Fork**
   ```bash
   git push origin feature/your-feature-name
   ```

5. **创建 Pull Request**
   在 GitHub 上提交 PR，描述你的改动

### 代码规范

- 遵循 [Effective Go](https://golang.org/doc/effective_go) 风格指南
- 使用 `gofmt` 格式化代码
- 添加必要的代码注释
- 新功能需要配套单元测试

---

## 🐛 常见问题

### Q: Dashboard 是否支持多用户并发访问？

A: 支持。Dashboard 设计为无状态服务，可部署多个副本，使用负载均衡器分发流量。

### Q: 压测结果不准确？

A: 确保：
1. ✅ 被测系统已稳定运行
2. ✅ 网络延迟不异常
3. ✅ 压测持续时间足够长（建议 ≥ 60s）
4. ✅ 查看错误分类，确保请求没有被拒绝

### Q: 如何导出压测结果用于数据分析？

A: 压测结果自动保存为 JSONL 格式：
```bash
# 查看原始数据
cat tmp/stress_results.jsonl

# 或使用 jq 处理
cat tmp/stress_results.jsonl | jq '.data.result.tps'
```

## 📄 许可证

本项目采用 [MIT License](LICENSE)

---

## 🎉 致谢

感谢所有为这个项目做出贡献的开发者！

特别感谢：
- [Gin Web Framework](https://github.com/gin-gonic/gin) - 高性能 HTTP 框架
- [ShardKV](https://github.com/khyallin/shardkv) - 分布式键值存储
- [Docker SDK for Go](https://github.com/moby/moby) - Docker 集成

---

<div align="center">

**⭐ 如果这个项目对你有帮助，请给个 Star！**

[English](README.en.md) | [示例](#使用示例) | [API](#api-文档) | [反馈](https://github.com/khyallin/shardkv-dashboard/issues)

</div>
