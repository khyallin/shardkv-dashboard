package service

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net"
	gorpc "net/rpc"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/khyallin/shardkv/api"
	"github.com/khyallin/shardkv/client"
	"github.com/khyallin/shardkv/config"
	"github.com/khyallin/shardkv/controller"

	"github.com/khyallin/shardkv-dashboard/pkg/shardkv"
)

const (
	stressErrWrongLeader = api.Err("ErrWrongLeader")
	stressErrWrongGroup  = api.Err("ErrWrongGroup")
)

type StressRunConfig struct {
	Duration    time.Duration
	Concurrency int
	ReadRatio   int
	ValueSize   int
	TargetGID   int
	KeyPrefix   string
}

type StressErrorBreakdown struct {
	WrongLeader int64 `json:"wrong_leader"`
	WrongGroup  int64 `json:"wrong_group"`
	VersionErr  int64 `json:"version_error"`
	Other       int64 `json:"other"`
}

type StressGroupStatus struct {
	TotalQPS   float64       `json:"total_qps"`
	DoneQPS    float64       `json:"done_qps"`
	SuccessQPS float64       `json:"success_qps"`
	MaxLatency time.Duration `json:"max_latency"`
	AvgLatency time.Duration `json:"avg_latency"`
	Err        string        `json:"err,omitempty"`
}

type StressResult struct {
	DurationSec float64 `json:"duration_sec"`

	Concurrency int    `json:"concurrency"`
	ReadRatio   int    `json:"read_ratio"`
	ValueSize   int    `json:"value_size"`
	TargetGID   int    `json:"target_gid"`
	KeyPrefix   string `json:"key_prefix"`

	TotalOps   int64 `json:"total_ops"`
	SuccessOps int64 `json:"success_ops"`
	FailedOps  int64 `json:"failed_ops"`
	ReadOps    int64 `json:"read_ops"`
	WriteOps   int64 `json:"write_ops"`

	TPS        float64       `json:"tps"`
	SuccessTPS float64       `json:"success_tps"`
	AvgLatency time.Duration `json:"avg_latency"`
	MaxLatency time.Duration `json:"max_latency"`

	Errors       StressErrorBreakdown `json:"errors"`
	BeforeStatus StressGroupStatus    `json:"before_status"`
	AfterStatus  StressGroupStatus    `json:"after_status"`
}

type StressService struct {
	mu            sync.Mutex
	running       bool
	ctrlerServers []string
}

func NewStressService() *StressService {
	return &StressService{
		ctrlerServers: shardkv.GetServers(config.Gid0),
	}
}

func (s *StressService) Run(cfg StressRunConfig) (*StressResult, error) {
	if err := validateStressRunConfig(cfg); err != nil {
		return nil, err
	}

	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return nil, fmt.Errorf("stress run already in progress")
	}
	s.running = true
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		s.running = false
		s.mu.Unlock()
	}()

	clusterCfg, err := s.loadCurrentConfig()
	if err != nil {
		return nil, err
	}

	targetGID := config.Tgid(cfg.TargetGID)
	targetServers, ok := clusterCfg.Groups[targetGID]
	if !ok || len(targetServers) == 0 {
		return nil, fmt.Errorf("stress precheck: target_gid=%d not found", cfg.TargetGID)
	}
	if !groupOwnsAnyShard(clusterCfg, targetGID) {
		return nil, fmt.Errorf("stress precheck: target_gid=%d has no shard", cfg.TargetGID)
	}

	beforeStatus, err := probeGroupStatus(targetServers, 2*time.Second)
	if err != nil {
		return nil, fmt.Errorf("stress precheck: target_gid=%d unavailable: %w", cfg.TargetGID, err)
	}

	keyPool := &stressKeyPool{keys: make([]string, 0, 1024)}
	counters := &stressCounters{}
	value := strings.Repeat("x", cfg.ValueSize)

	var putSeq uint64
	var wg sync.WaitGroup
	startAt := time.Now()
	deadline := startAt.Add(cfg.Duration)

	for workerID := 0; workerID < cfg.Concurrency; workerID++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			ck := client.MakeClerk(s.ctrlerServers)
			rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id+1)*7919))

			for time.Now().Before(deadline) {
				doRead := rng.Intn(100) < cfg.ReadRatio
				if doRead {
					key, ok := keyPool.Random(rng)
					if ok {
						opStart := time.Now()
						_, _, err := ck.Get(key)
						counters.Observe(true, err, time.Since(opStart))
						continue
					}
				}

				seq := atomic.AddUint64(&putSeq, 1) - 1
				key := keyForTargetGID(cfg.KeyPrefix, seq, targetGID, clusterCfg)
				opStart := time.Now()
				err := ck.Put(key, value, 0)
				counters.Observe(false, err, time.Since(opStart))
				if err == api.OK {
					keyPool.Add(key)
				}
			}
		}(workerID)
	}

	wg.Wait()
	elapsed := time.Since(startAt)

	afterStatus, statusErr := probeGroupStatus(targetServers, 2*time.Second)
	if statusErr != nil {
		afterStatus.Err = statusErr.Error()
	}

	result := counters.BuildResult(cfg, elapsed)
	result.BeforeStatus = beforeStatus
	result.AfterStatus = afterStatus
	return result, nil
}

func validateStressRunConfig(cfg StressRunConfig) error {
	if cfg.Duration <= 0 {
		return fmt.Errorf("duration must be > 0")
	}
	if cfg.Concurrency < 1 {
		return fmt.Errorf("concurrency must be >= 1")
	}
	if cfg.ReadRatio < 0 || cfg.ReadRatio > 100 {
		return fmt.Errorf("read_ratio must be in [0,100]")
	}
	if cfg.ValueSize < 1 || cfg.ValueSize > 1024*1024 {
		return fmt.Errorf("value_size must be in [1,1048576]")
	}
	if cfg.TargetGID <= 0 {
		return fmt.Errorf("target_gid must be > 0")
	}
	if strings.TrimSpace(cfg.KeyPrefix) == "" {
		return fmt.Errorf("key_prefix cannot be empty")
	}
	return nil
}

func (s *StressService) loadCurrentConfig() (*config.Config, error) {
	ctrler := controller.MakeController(s.ctrlerServers)
	value, _, err := ctrler.Get("config")
	if err != api.OK {
		return nil, fmt.Errorf("stress precheck: controller unavailable: %v", err)
	}
	if value == "" {
		return nil, fmt.Errorf("stress precheck: empty config from controller")
	}

	var cfg config.Config
	if err := json.Unmarshal([]byte(value), &cfg); err != nil {
		return nil, fmt.Errorf("stress precheck: decode config: %v", err)
	}
	if cfg.Groups == nil {
		return nil, fmt.Errorf("stress precheck: invalid config from controller")
	}
	return &cfg, nil
}

func groupOwnsAnyShard(cfg *config.Config, gid config.Tgid) bool {
	for _, owner := range cfg.Shards {
		if owner == gid {
			return true
		}
	}
	return false
}

func keyForTargetGID(prefix string, seq uint64, gid config.Tgid, cfg *config.Config) string {
	key := fmt.Sprintf("%s%d", prefix, seq)
	if cfg.Shards[config.Key2Shard(key)] == gid {
		return key
	}
	for salt := 0; salt < 256; salt++ {
		candidate := fmt.Sprintf("%s%d-%d", prefix, seq, salt)
		if cfg.Shards[config.Key2Shard(candidate)] == gid {
			return candidate
		}
	}
	return key
}

func probeGroupStatus(servers []string, timeout time.Duration) (StressGroupStatus, error) {
	var lastErr error
	for _, server := range servers {
		reply, err := callGroupStatus(server, timeout)
		if err != nil {
			lastErr = err
			continue
		}
		if reply.Err == api.OK {
			return StressGroupStatus{
				TotalQPS:   reply.TotalQPS,
				DoneQPS:    reply.DoneQPS,
				SuccessQPS: reply.SuccessQPS,
				MaxLatency: reply.MaxLatency,
				AvgLatency: reply.AvgLatency,
			}, nil
		}
		lastErr = fmt.Errorf("%s status error: %v", server, reply.Err)
	}

	if lastErr == nil {
		lastErr = fmt.Errorf("no reachable server")
	}
	return StressGroupStatus{Err: lastErr.Error()}, lastErr
}

func callGroupStatus(server string, timeout time.Duration) (*api.StatusReply, error) {
	conn, err := net.DialTimeout("tcp", server+config.Port, timeout)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %v", server, err)
	}
	_ = conn.SetDeadline(time.Now().Add(timeout))

	rpcClient := gorpc.NewClient(conn)
	defer rpcClient.Close()

	reply := &api.StatusReply{}
	if err := rpcClient.Call("KVServer.Status", &api.StatusArgs{}, reply); err != nil {
		return nil, fmt.Errorf("call %s: %v", server, err)
	}
	return reply, nil
}

type stressKeyPool struct {
	mu   sync.RWMutex
	keys []string
}

func (p *stressKeyPool) Add(key string) {
	p.mu.Lock()
	p.keys = append(p.keys, key)
	p.mu.Unlock()
}

func (p *stressKeyPool) Random(rng *rand.Rand) (string, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if len(p.keys) == 0 {
		return "", false
	}
	idx := rng.Intn(len(p.keys))
	return p.keys[idx], true
}

type stressCounters struct {
	totalOps     int64
	successOps   int64
	failedOps    int64
	readOps      int64
	writeOps     int64
	totalLatency int64
	maxLatency   int64
	wrongLeader  int64
	wrongGroup   int64
	versionErr   int64
	otherErr     int64
}

func (c *stressCounters) Observe(isRead bool, err api.Err, latency time.Duration) {
	atomic.AddInt64(&c.totalOps, 1)
	if isRead {
		atomic.AddInt64(&c.readOps, 1)
	} else {
		atomic.AddInt64(&c.writeOps, 1)
	}

	latencyNs := latency.Nanoseconds()
	atomic.AddInt64(&c.totalLatency, latencyNs)
	for {
		old := atomic.LoadInt64(&c.maxLatency)
		if latencyNs <= old || atomic.CompareAndSwapInt64(&c.maxLatency, old, latencyNs) {
			break
		}
	}

	if err == api.OK {
		atomic.AddInt64(&c.successOps, 1)
		return
	}

	atomic.AddInt64(&c.failedOps, 1)
	switch err {
	case stressErrWrongLeader:
		atomic.AddInt64(&c.wrongLeader, 1)
	case stressErrWrongGroup:
		atomic.AddInt64(&c.wrongGroup, 1)
	case api.ErrVersion:
		atomic.AddInt64(&c.versionErr, 1)
	default:
		atomic.AddInt64(&c.otherErr, 1)
	}
}

func (c *stressCounters) BuildResult(cfg StressRunConfig, elapsed time.Duration) *StressResult {
	totalOps := atomic.LoadInt64(&c.totalOps)
	successOps := atomic.LoadInt64(&c.successOps)
	failedOps := atomic.LoadInt64(&c.failedOps)
	readOps := atomic.LoadInt64(&c.readOps)
	writeOps := atomic.LoadInt64(&c.writeOps)
	totalLatency := atomic.LoadInt64(&c.totalLatency)
	maxLatency := atomic.LoadInt64(&c.maxLatency)

	elapsedSec := elapsed.Seconds()
	if elapsedSec <= 0 {
		elapsedSec = 1
	}

	var avgLatency time.Duration
	if totalOps > 0 {
		avgLatency = time.Duration(totalLatency / totalOps)
	}

	return &StressResult{
		DurationSec: elapsedSec,

		Concurrency: cfg.Concurrency,
		ReadRatio:   cfg.ReadRatio,
		ValueSize:   cfg.ValueSize,
		TargetGID:   cfg.TargetGID,
		KeyPrefix:   cfg.KeyPrefix,

		TotalOps:   totalOps,
		SuccessOps: successOps,
		FailedOps:  failedOps,
		ReadOps:    readOps,
		WriteOps:   writeOps,

		TPS:        float64(totalOps) / elapsedSec,
		SuccessTPS: float64(successOps) / elapsedSec,
		AvgLatency: avgLatency,
		MaxLatency: time.Duration(maxLatency),

		Errors: StressErrorBreakdown{
			WrongLeader: atomic.LoadInt64(&c.wrongLeader),
			WrongGroup:  atomic.LoadInt64(&c.wrongGroup),
			VersionErr:  atomic.LoadInt64(&c.versionErr),
			Other:       atomic.LoadInt64(&c.otherErr),
		},
	}
}
