# ShardKV Dashboard

<div align="center">

[![Go Version](https://img.shields.io/badge/Go-1.25+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Docker](https://img.shields.io/badge/Docker-Ready-brightgreen.svg)](Dockerfile)

**🎯 Visual Management Platform for Distributed Key-Value Storage** | **Real-time Monitoring** | **Performance Testing** | **One-Click Deploy**

[**中文**](README.md) | [**Quick Start**](#quick-start) | [**Features**](#key-features) | [**API Docs**](#api-documentation)

</div>

---

## ✨ Overview

**ShardKV Dashboard** is an **enterprise-grade distributed systems management tool** that provides complete **visual management, real-time monitoring, data operations, and performance testing** capabilities for ShardKV (Sharded Key-Value Storage) clusters.

Whether you're a system administrator, performance engineer, or distributed systems researcher, ShardKV Dashboard transforms complex cluster management into an intuitive, accessible experience.

### 💡 Why ShardKV Dashboard?

| Feature | Description |
|---------|-------------|
| 🎨 **Zero Learning Curve** | Intuitive web interface, drag-and-drop operations, no CLI needed |
| 📊 **Real-time Insights** | Sub-millisecond monitoring of cluster QPS, latency, success rates |
| 🔧 **Flexible Management** | Dynamically create/stop shard groups, intelligent shard migration |
| ⚡ **Performance Assessment** | Built-in multi-dimensional load testing with custom scenarios |
| 🐳 **Ready Out of the Box** | Docker/Docker Compose for instant deployment |
| 📈 **Professional Analysis** | JSON Lines output + Python plotting scripts |

---

## 🚀 Quick Start

### Prerequisites

- **Go 1.25+** 
- **Docker 29.2+**
- **Python 3.8+** (optional, for analyzing stress test results)

### Using MakeFile (Recommended)

The simplest way to get everything running:

```bash
# 1. Clone the repository
git clone https://github.com/khyallin/shardkv-dashboard.git
cd shardkv-dashboard

# 2. Start all services
make all

# 3. Open your browser
open http://localhost:8080
```

### Using Docker

```bash
# 1. Pull the image
docker pull khyal/shardkv-dashboard

# 2. Start the service
docker run -d --name shardkv-dashboard --network shardkv-net -v /var/run/docker.sock:/var/run/docker.sock -p 8080:8080 khyal/shardkv-dashboard
```
That's it! Dashboard is now running at `http://localhost:8080`.

---

## 📚 Key Features

### 1. 🗂️ Cluster Management

**Intuitive Shard Group Management** - Visual cluster topology editing

```
Cluster Overview → Create Shard Groups → Drag Shards → Performance Monitoring
        ↓
Real-time Display:
  • Shard Distribution
  • QPS, Success Rate, Latency per Group
  • Auto Rebalancing Status
```

**Supported Operations:**
- ✅ Dynamically create new shard groups
- ✅ Drag-and-drop shard migration to target groups (instant effect)
- ✅ Gracefully stop shard groups (no data loss)
- ✅ Auto load balancing (checks every 10s when enabled)

### 2. 📊 Performance Monitoring

**Real-time Metrics Dashboard** - Cluster status at a glance

```json
{
  "total_qps": 15420,        // Total throughput
  "done_qps": 14890,         // Completed throughput
  "success_qps": 14850,      // Successful throughput
  "avg_latency": 2.3,        // Average latency (ms)
  "max_latency": 45.8        // Max latency (ms)
}
```

**Monitoring Dimensions:**
- Cluster-wide QPS and success rate
- Per-shard-group performance metrics
- Real-time latency statistics (min/avg/max)
- Error classification breakdown

### 3. 🔑 Data Operations

**Web Form-based Data Management** - Operate data without coding

**Get Operation:**
```
Enter Key → Instant Query → Display:
  • Value
  • Value Type (string/int/float/bool)
  • Version Number
```

**Put Operation:**
```
Fill Form:
  • Key: data key
  • Type: choose type (string/int/float/bool)
  • Value: input value (auto type validation)
  • Version: specify version (optional)
```

**Supported Data Types:**
| Type | Example | Description |
|------|---------|-------------|
| `string` | `"hello world"` | String |
| `int` | `42` | Integer (-9223372036854775808 ~ 9223372036854775807) |
| `float` | `3.14` | Float (no Inf/NaN) |
| `bool` | `true/false` | Boolean |

### 4. ⚡ Stress Testing

**Enterprise-grade Load Testing** - Multi-dimensional performance evaluation

**Test Dimensions:**
```
Concurrency: 1, 4, 16, 32, 64 ...
Read/Write Ratio: 0% (write-only), 50% (mixed), 100% (read-only)
Data Size: 16B, 64B, 256B, 1KB, 4KB ...
```

**Test Results Include:**
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

**Built-in Analysis Scripts:**
```bash
# Run stress tests and export results
scripts/stress.sh

# Generate performance curves with Python
python3 scripts/draw.py tmp/stress_results.jsonl
```

---

## 🏗️ Architecture

### System Architecture

```
┌─────────────────────────────────────────────────────┐
│           ShardKV Dashboard                         │
├─────────────────────────────────────────────────────┤
│                                                     │
│  ┌──────────────────┐         ┌────────────────┐  │
│  │   Web Frontend   │         │   REST API     │  │
│  │ (HTML/CSS/JS)    │◄────────►│    (Gin)       │  │
│  │ • Cluster Mgmt   │         │ • 11 endpoints │  │
│  │ • Drag Editing   │         │ • Unified      │  │
│  │ • Real-time Mon. │         │   Response     │  │
│  └──────────────────┘         └────────────────┘  │
│                                      ▲             │
│        Service Layer (Business Logic) │            │
│        ┌──────────────────────────────┴────────┐  │
│        │                                       │  │
│   ┌────▼──────┐ ┌────────────┐ ┌──────────┐  │  │
│   │ConfigServ │ │ KVService  │ │StressServ│  │  │
│   │• Cluster  │ │• Get/Put   │ │• Testing │  │  │
│   │  Mgmt     │ │• Type Check│ │• Result  │  │  │
│   │• Shard    │ │            │ │  Stats   │  │  │
│   │  Migration│ │            │ │          │  │  │
│   └────┬──────┘ └────┬───────┘ └────┬─────┘  │  │
│        │              │             │        │  │
└────────┼──────────────┼─────────────┼────────┘  │
         │              │             │           │
         ▼              ▼             ▼           │
    ┌──────────────────────────────────────────┐  │
    │      ShardKV System (Managed)            │  │
    │  • Controller (cluster config)           │  │
    │  • Shard Groups                          │  │
    │  • State Machine                         │  │
    └──────────────────────────────────────────┘  │
```

### API Endpoints Overview

| Endpoint | Method | Function | Auth |
|----------|--------|----------|------|
| `/ping` | GET | Health check | ✅ Public |
| `/` | GET | Web page | ✅ Public |
| **Cluster Management** | | | |
| `/api/v1/group/get` | POST | Get cluster config | 🔒 App |
| `/api/v1/group/status` | POST | Query shard group status | 🔒 App |
| `/api/v1/group/create` | POST | Create shard group | 🔒 App |
| `/api/v1/group/stop` | POST | Stop shard group | 🔒 App |
| **Configuration** | | | |
| `/api/v1/config` | POST | Manual shard migration | 🔒 App |
| `/api/v1/config/auto` | POST | Auto-balance toggle | 🔒 App |
| **Data Operations** | | | |
| `/api/v1/kv/get` | POST | Query key-value | 🔒 App |
| `/api/v1/kv/put` | POST | Set key-value | 🔒 App |
| **Stress Testing** | | | |
| `/api/v1/stress/run` | POST | Run stress test | 🔒 App |

### Response Format

All API responses follow a unified format:

```json
{
  "code": 0,                  // 0=success, non-0=failure
  "message": "success",       // Operation message
  "data": { ... }             // Business data (optional)
}
```

**Error Response Example:**
```json
{
  "code": -1,
  "message": "failed to create group: timeout",
  "data": null
}
```

---

## 📖 API Documentation

### Get Cluster Configuration

**Request**
```bash
curl -X POST http://localhost:8080/api/v1/group/get \
  -H "Content-Type: application/json" \
  -d '{}'
```

**Response**
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

### Query Shard Group Status

**Request**
```bash
curl -X POST http://localhost:8080/api/v1/group/status \
  -H "Content-Type: application/json" \
  -d '{"gid": 1}'
```

**Response**
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

### Key-Value Query (Get)

**Request**
```bash
curl -X POST http://localhost:8080/api/v1/kv/get \
  -H "Content-Type: application/json" \
  -d '{"key": "user_123"}'
```

**Response**
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

### Key-Value Set (Put)

**Request**
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

**Response**
```json
{
  "code": 0,
  "message": "success",
  "data": null
}
```

---

### Run Stress Test

**Request**
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

**Response**
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

## 🛠️ Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DASHBOARD_PORT` | `8080` | Dashboard service port |
| `SHARDKV_ADDR` | `localhost:2379` | ShardKV service address |
| `DOCKER_HOST` | `unix:///var/run/docker.sock` | Docker daemon address |

### Configuration File

Edit `config/default.go` to adjust default parameters:

```go
// Auto-rebalance check interval
AutoRebalanceCheckInterval: 10 * time.Second

// Stress test timeout
StressTimeout: 120 * time.Second

// Maximum concurrency
MaxConcurrency: 256
```

---

## 📊 Usage Examples

### Scenario 1: Cluster Deployment & Initialization

```bash
# 1. Start Dashboard
make all

# 2. Open browser
open http://localhost:8080

# 3. Create shard groups via Web UI (click + button)

# 4. Drag-and-drop shard allocation

# 5. Enable auto load balancing
```

### Scenario 2: Performance Baseline Test

```bash
# 1. Run stress test
make stress

# 2. View test results
cat ../tmp/stress_results.jsonl | head -20

# 3. Generate performance curves
make draw
```

### Scenario 3: Data Validation

```bash
# Execute query and set operations via Web UI to validate data correctness

# Or use curl
curl -X POST http://localhost:8080/api/v1/kv/put \
  -H "Content-Type: application/json" \
  -d '{"key":"test","type":"string","value":"hello"}'

curl -X POST http://localhost:8080/api/v1/kv/get \
  -H "Content-Type: application/json" \
  -d '{"key":"test"}'
```

---

## 🔧 Development Guide

### Project Structure

```
shardkv-dashboard/
├── cmd/
│   ├── main.go              # Program entry point
│   └── server.go            # Server configuration
├── internal/
│   ├── app/                 # Application initialization
│   │   ├── app.go
│   │   └── router.go
│   ├── handler/             # HTTP handlers (11 endpoints)
│   │   ├── ping.go
│   │   ├── configget.go
│   │   ├── groupcreate.go
│   │   ├── kvget.go
│   │   └── ...
│   └── service/             # Business logic
│       ├── config.go        # Cluster management
│       ├── kv.go            # Key-value operations
│       └── stress.go        # Stress testing
├── pkg/
│   ├── shardkv/             # ShardKV adapter
│   └── docker/              # Docker integration
├── config/
│   ├── config.go
│   └── default.go
├── web/
│   └── index.html           # Frontend (3000+ lines)
├── scripts/
│   ├── stress.sh            # Stress test script
│   └── draw.py              # Result analysis
├── Makefile
├── Dockerfile
└── go.mod
```

### Build & Run

```bash
# View all available commands
make help

# First time startup
make all

# Restart service
make restart

# Build Docker image
make build

# Performance benchmark
make bench
```

### Contributing Process

1. **Fork the Project**
   ```bash
   git clone https://github.com/khyallin/shardkv-dashboard.git
   cd shardkv-dashboard
   ```

2. **Create Feature Branch**
   ```bash
   git switch -c feature/your-feature-name
   ```

3. **Commit Changes**
   ```bash
   git commit -am "feat: add your feature"
   ```

4. **Push to Fork**
   ```bash
   git push origin feature/your-feature-name
   ```

5. **Create Pull Request**
   Submit a PR on GitHub describing your changes

### Code Standards

- Follow [Effective Go](https://golang.org/doc/effective_go) style guide
- Format code with `gofmt`
- Add necessary code comments
- Include unit tests for new features

---

## 🐛 FAQ

### Q: Does Dashboard support concurrent multi-user access?

A: Yes. Dashboard is designed as a stateless service and can be deployed with multiple replicas using a load balancer.

### Q: Stress test results are inaccurate?

A: Ensure:
1. ✅ Target system is running stably
2. ✅ Network latency is normal
3. ✅ Test duration is long enough (recommend ≥ 60s)
4. ✅ Check error breakdown to ensure requests aren't rejected

### Q: How to export stress test results for data analysis?

A: Results are automatically saved in JSONL format:
```bash
# View raw data
cat tmp/stress_results.jsonl

# Or process with jq
cat tmp/stress_results.jsonl | jq '.data.result.tps'
```

## 📄 License

This project is licensed under the [MIT License](LICENSE)

---

## 🎉 Acknowledgments

Thanks to all developers who contributed to this project!

Special thanks to:
- [Gin Web Framework](https://github.com/gin-gonic/gin) - High-performance HTTP framework
- [ShardKV](https://github.com/khyallin/shardkv) - Distributed key-value storage
- [Docker SDK for Go](https://github.com/moby/moby) - Docker integration

---

<div align="center">

**⭐ If this project helps you, please give it a Star!**

[中文](README.md) | [Examples](#usage-examples) | [API](#api-documentation) | [Feedback](https://github.com/khyallin/shardkv-dashboard/issues)

</div>
