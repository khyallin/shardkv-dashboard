package service

import (
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/khyallin/shardkv/api"
	"github.com/khyallin/shardkv/client"
	"github.com/khyallin/shardkv/config"
	"github.com/khyallin/shardkv/controller"

	"github.com/khyallin/shardkv-dashboard/pkg/rebalance"
	"github.com/khyallin/shardkv-dashboard/pkg/shardkv"
)

type ConfigService struct {
	mu sync.Mutex

	skv        *shardkv.ShardKV
	ctrler     *controller.Controller
	client     *client.Clerk
	groups     []*shardkv.Group
	rebalancer rebalance.Rebalancer

	auto atomic.Int32
	dead atomic.Int32
}

func NewConfigService() *ConfigService {
	skv := shardkv.New()
	s := &ConfigService{
		skv:        skv,
		ctrler:     shardkv.MakeCtrler(),
		client:     shardkv.MakeClient(),
		groups:     make([]*shardkv.Group, 0),
		rebalancer: rebalance.New(),
	}
	s.setup()
	return s
}

func (s *ConfigService) GetConfig() (bool, string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	auto := s.auto.Load() == 1
	mode := "default"
	switch s.rebalancer.(type) {
	case *rebalance.NullRebalancer:
		mode = "null"
	case *rebalance.NumRebalancer:
		mode = "num"
	case *rebalance.QpsRebalancer:
		mode = "qps"
	case *rebalance.LatencyAwareRebalancer:
		mode = "latency"
	case *rebalance.SuccessAwareRebalancer:
		mode = "success"
	case *rebalance.GradualRebalancer:
		mode = "gradual"
	case *rebalance.MultiDimensionRebalancer:
		mode = "multidim"
	default:
		return false, "", fmt.Errorf("ConfigService Mode Error %T", s.rebalancer)
	}
	return auto, mode, nil
}

func (s *ConfigService) SetAuto(auto bool) error {
	if auto {
		s.auto.Store(1)
	} else {
		s.auto.Store(0)
	}
	return nil
}

func (s *ConfigService) SetMode(mode string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch mode {
	case "null":
		s.rebalancer = &rebalance.NullRebalancer{}
	case "num":
		s.rebalancer = &rebalance.NumRebalancer{}
	case "qps":
		s.rebalancer = &rebalance.QpsRebalancer{}
	case "latency":
		s.rebalancer = &rebalance.LatencyAwareRebalancer{}
	case "success":
		s.rebalancer = &rebalance.SuccessAwareRebalancer{}
	case "gradual":
		s.rebalancer = &rebalance.GradualRebalancer{}
	case "default", "multidim":
		s.rebalancer = rebalance.New()
	default:
		return fmt.Errorf("ConfigService SetMode: mode %s not supported", mode)
	}
	return nil
}

func (s *ConfigService) setup() {
	group0 := s.skv.MakeGroup(config.Gid0)
	s.skv.RunGroup(group0)
	group1 := s.skv.MakeGroup(config.Gid1)
	s.skv.RunGroup(group1)
	s.groups = append(s.groups, group0, group1)

	s.ctrler.InitConfig(shardkv.DefaultConfig())
	go s.ticker()
}

func (s *ConfigService) ticker() {
	ticker := time.NewTicker(time.Second * 10)
	for range ticker.C {
		if s.dead.Load() == 1 {
			return
		}
		if s.auto.Load() == 0 {
			continue
		}

		err := s.Rebalance()
		if err != nil {
			log.Printf("ConfigService Rebalance err: %v", err)
		}
	}
}

func (s *ConfigService) teardown() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.dead.Load() == 1 {
		return
	}
	s.dead.Store(1)

	for _, group := range s.groups {
		s.skv.StopGroup(group)
	}
	for _, group := range s.groups {
		s.skv.RemoveGroup(group)
	}
}

func (s *ConfigService) GetGroup() (int, []int, map[int][]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg := s.ctrler.Query()
	shards := make([]int, len(cfg.Shards))
	for i, gid := range cfg.Shards {
		shards[i] = int(gid)
	}
	groups := make(map[int][]string)
	for gid, servers := range cfg.Groups {
		groups[int(gid)] = servers
	}
	return int(cfg.Num), shards, groups, nil
}

func (s *ConfigService) CreateGroup() (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	gid := config.Tgid(len(s.groups))
	group := s.skv.MakeGroup(gid)
	s.skv.RunGroup(group)
	s.groups = append(s.groups, group)

	cfg := s.ctrler.Query()
	cfg.Groups[gid] = shardkv.GetServers(gid)
	s.ctrler.ChangeConfigTo(cfg)
	return int(gid), nil
}

func (s *ConfigService) StopGroup(gid int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if gid <= 0 || gid >= len(s.groups) {
		return fmt.Errorf("ConfigService StopGroup: group %d not found", gid)
	}
	if s.groups[gid].Status != shardkv.StatusRunning {
		return fmt.Errorf("ConfigService StopGroup: group %d is not running", gid)
	}
	for _, shard := range s.ctrler.Query().Shards {
		if shard == config.Tgid(gid) {
			return fmt.Errorf("ConfigService StopGroup: group %d has shard %d", gid, shard)
		}
	}
	s.skv.StopGroup(s.groups[gid])

	cfg := s.ctrler.Query()
	delete(cfg.Groups, config.Tgid(gid))
	s.ctrler.ChangeConfigTo(cfg)
	return nil
}

func (s *ConfigService) MoveShard(shard int, from int, to int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cfg := s.ctrler.Query()
	if shard < 0 || shard >= config.NShards {
		return fmt.Errorf("ConfigService MoveShard: shard %d out of range", shard)
	}
	if from <= 0 || from >= len(s.groups) {
		return fmt.Errorf("ConfigService MoveShard: group %d not found", from)
	}
	if to <= 0 || to >= len(s.groups) {
		return fmt.Errorf("ConfigService MoveShard: group %d not found", to)
	}
	if from == to {
		return fmt.Errorf("ConfigService MoveShard: from %d and to %d are the same", from, to)
	}
	if cfg.Shards[shard] != config.Tgid(from) {
		return fmt.Errorf("ConfigService MoveShard: shard %d from %d error", shard, from)
	}
	if s.groups[from].Status != shardkv.StatusRunning || s.groups[to].Status != shardkv.StatusRunning {
		return fmt.Errorf("ConfigService MoveShard: groups %d or %d are not running", from, to)
	}
	cfg.Shards[shard] = config.Tgid(to)
	s.ctrler.ChangeConfigTo(cfg)
	return nil
}

func (s *ConfigService) GroupStatus(gid int) (float64, float64, float64, time.Duration, time.Duration, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if gid <= 0 || gid >= len(s.groups) {
		return 0, 0, 0, 0, 0, fmt.Errorf("ConfigService GetGroupRunningStatus: group %d not found", gid)
	}
	if s.groups[gid].Status != shardkv.StatusRunning {
		return 0, 0, 0, 0, 0, fmt.Errorf("ConfigService GetGroupRunningStatus: group %d is not running", gid)
	}
	totalQPS, doneQPS, successQPS, maxLatency, avgLatency, err := s.client.Status(config.Tgid(gid))
	if err != api.OK {
		return 0, 0, 0, 0, 0, fmt.Errorf("ConfigService GetGroupRunningStatus: group %d status error: %v", gid, err)
	}
	return totalQPS, doneQPS, successQPS, maxLatency, avgLatency, nil
}

func (s *ConfigService) Rebalance() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	groups := make(map[int]*rebalance.GroupRunningStatus)
	for gid, group := range s.groups {
		if gid == 0 || group.Status != shardkv.StatusRunning {
			continue
		}
		totalQPS, doneQPS, successQPS, maxLatency, avgLatency, err := s.client.Status(group.ID)
		if err != api.OK {
			return fmt.Errorf("ConfigService Rebalance: group %d status error: %v", gid, err)
		}
		groups[gid] = &rebalance.GroupRunningStatus{
			ID:         gid,
			TotalQPS:   totalQPS,
			DoneQPS:    doneQPS,
			SuccessQPS: successQPS,
			MaxLatency: maxLatency,
			AvgLatency: avgLatency,
		}
	}
	cfg := s.ctrler.Query()
	s.rebalancer.Rebalance(cfg, groups)
	s.ctrler.ChangeConfigTo(cfg)
	return nil
}
