package service

import (
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/khyallin/shardkv/api"
	"github.com/khyallin/shardkv/client"
	"github.com/khyallin/shardkv/config"

	"github.com/khyallin/shardkv-dashboard/pkg/shardkv"
)

const (
	stressErrWrongLeader = api.Err("ErrWrongLeader")
	stressErrWrongGroup  = api.Err("ErrWrongGroup")
	stressKeyPoolLimit   = 64
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

	ck := client.MakeClerk(s.ctrlerServers)
	clusterCfg := shardkv.DefaultConfig()

	targetGID := config.Tgid(cfg.TargetGID)
	targetAny := cfg.TargetGID <= 0
	var beforeStatus StressGroupStatus
	if !targetAny {
		targetServers, ok := clusterCfg.Groups[targetGID]
		if !ok || len(targetServers) == 0 {
			return nil, fmt.Errorf("stress precheck: target_gid=%d not found", cfg.TargetGID)
		}
		if !groupOwnsAnyShard(clusterCfg, targetGID) {
			return nil, fmt.Errorf("stress precheck: target_gid=%d has no shard", cfg.TargetGID)
		}

		var err error
		beforeStatus, err = probeGroupStatus(ck, targetGID)
		if err != nil {
			return nil, fmt.Errorf("stress precheck: target_gid=%d unavailable: %w", cfg.TargetGID, err)
		}
	}

	keyPool := &stressKeyPool{keys: make([]string, 0, stressKeyPoolLimit)}
	if err := bootstrapStressKeyPool(ck, keyPool, cfg.KeyPrefix, targetGID, targetAny, clusterCfg); err != nil {
		return nil, fmt.Errorf("stress bootstrap: %w", err)
	}
	counters := &stressCounters{}
	value := strings.Repeat("x", cfg.ValueSize)
	var wg sync.WaitGroup
	startAt := time.Now()
	deadline := startAt.Add(cfg.Duration)

	for workerID := 0; workerID < cfg.Concurrency; workerID++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			workerCk := client.MakeClerk(s.ctrlerServers)
			rng := rand.New(rand.NewSource(time.Now().UnixNano() + int64(id+1)*7919))

			for time.Now().Before(deadline) {
				doRead := rng.Intn(100) < cfg.ReadRatio
				if doRead {
					key, ok := keyPool.Random(rng)
					if ok {
						opStart := time.Now()
						_, _, err := workerCk.Get(key)
						counters.Observe(true, err, time.Since(opStart))
						continue
					}
				}

				key, ok := keyPool.Random(rng)
				if !ok {
					continue
				}
				_, version, getErr := workerCk.Get(key)
				if getErr != api.OK {
					counters.Observe(false, getErr, 0)
					continue
				}
				opStart := time.Now()
				err := workerCk.Put(key, value, version)
				counters.Observe(false, err, time.Since(opStart))
			}
		}(workerID)
	}

	wg.Wait()
	elapsed := time.Since(startAt)

	var afterStatus StressGroupStatus
	if !targetAny {
		statusErr := error(nil)
		afterStatus, statusErr = probeGroupStatus(ck, targetGID)
		if statusErr != nil {
			afterStatus.Err = statusErr.Error()
		}
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
	if cfg.TargetGID < 0 {
		return fmt.Errorf("target_gid must be >= 0")
	}
	if strings.TrimSpace(cfg.KeyPrefix) == "" {
		return fmt.Errorf("key_prefix cannot be empty")
	}
	return nil
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

func bootstrapStressKeyPool(ck *client.Clerk, keyPool *stressKeyPool, prefix string, gid config.Tgid, targetAny bool, cfg *config.Config) error {
	for seq := 0; keyPool.Len() < stressKeyPoolLimit; seq++ {
		key := ""
		if targetAny {
			key = fmt.Sprintf("%s%d", prefix, seq)
		} else {
			key = keyForTargetGID(prefix, uint64(seq), gid, cfg)
		}
		_, version, err := ck.Get(key)
		switch err {
		case api.OK:
			keyPool.Add(key)
		case api.ErrNoKey:
			if putErr := ck.Put(key, "", 0); putErr != api.OK {
				return fmt.Errorf("seed put %s: %v", key, putErr)
			}
			keyPool.Add(key)
			_ = version
		default:
			return fmt.Errorf("seed get %s: %v", key, err)
		}
	}
	return nil
}

func probeGroupStatus(ck *client.Clerk, gid config.Tgid) (StressGroupStatus, error) {
	totalQps, doneQps, successQps, maxLatency, avgLatency, err := ck.Status(gid)
	if err == api.OK {
		return StressGroupStatus{
			TotalQPS:   totalQps,
			DoneQPS:    doneQps,
			SuccessQPS: successQps,
			MaxLatency: maxLatency,
			AvgLatency: avgLatency,
		}, nil
	}

	return StressGroupStatus{Err: string(err)}, fmt.Errorf("status error: %v", err)
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

func (p *stressKeyPool) Len() int {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return len(p.keys)
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
