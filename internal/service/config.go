package service

import (
	"fmt"
	"log"
	"math/rand"
	"sync/atomic"
	"time"

	"github.com/khyallin/shardkv/api"
	"github.com/khyallin/shardkv/client"
	"github.com/khyallin/shardkv/config"
	"github.com/khyallin/shardkv/controller"

	"github.com/khyallin/shardkv-dashboard/pkg/shardkv"
)

type ConfigService struct {
	skv    *shardkv.ShardKV
	ctrler *controller.Controller
	client *client.Clerk
	groups []*shardkv.Group
	auto   atomic.Int32
}

func NewConfigService() *ConfigService {
	skv := shardkv.New()
	s := &ConfigService{
		skv:    skv,
		ctrler: skv.MakeCtrler(),
		client: skv.MakeClient(),
		groups: make([]*shardkv.Group, 0),
	}
	s.setup()
	return s
}

func (s *ConfigService) setup() {
	group0 := s.skv.MakeGroup(config.Gid0)
	s.skv.RunGroup(group0)
	group1 := s.skv.MakeGroup(config.Gid1)
	s.skv.RunGroup(group1)
	s.groups = append(s.groups, group0, group1)

	s.ctrler.InitConfig(s.skv.DefaultConfig())
	go s.ticker()
}

func (s *ConfigService) ticker() {
	ticker := time.NewTicker(time.Second * 10)
	for range ticker.C {
		if s.auto.Load() == 0 {
			continue
		}
		err := s.Rebalance()
		if err != nil {
			log.Fatalf(err.Error())
		}
	}
}

func (s *ConfigService) teardown() {
	for _, group := range s.groups {
		s.skv.StopGroup(group)
	}
	for _, group := range s.groups {
		s.skv.RemoveGroup(group)
	}
}

func (s *ConfigService) Get() (int, []int, map[int][]string, error) {
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
	if gid <= 0 || gid > len(s.groups) {
		return fmt.Errorf("ConfigService StopGroup: group %d not found", gid)
	}
	if s.groups[gid].Status != shardkv.StatusRunning {
		return fmt.Errorf("ConfigService StopGroup: group %d is not running", gid)
	}
	s.skv.StopGroup(s.groups[gid])

	cfg := s.ctrler.Query()
	delete(cfg.Groups, config.Tgid(gid))
	for i := range cfg.Shards {
		if cfg.Shards[i] == config.Tgid(gid) {
			cfg.Shards[i] = config.Gid1
		}
	}
	s.ctrler.ChangeConfigTo(cfg)
	return nil
}

func (s *ConfigService) SetAuto(auto bool) error {
	if auto {
		s.auto.Store(1)
	} else {
		s.auto.Store(0)
	}
	return nil
}

func (s *ConfigService) MoveShard(shard int, from int, to int) error {
	cfg := s.ctrler.Query()
	if shard < 0 || shard >= config.NShards {
		return fmt.Errorf("ConfigService MoveShard: shard %d out of range", shard)
	}
	if from <= 0 || from > len(s.groups) {
		return fmt.Errorf("ConfigService MoveShard: group %d not found", from)
	}
	if to <= 0 || to > len(s.groups) {
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

type GroupRunningStatus struct {
	ID         int
	TotalQPS   float64
	DoneQPS    float64
	SuccessQPS float64
	MaxLatency time.Duration
	AvgLatency time.Duration
}

func (s *ConfigService) Rebalance() error {
	groups := make(map[int]*GroupRunningStatus)
	for gid, group := range s.groups {
		if group.Status != shardkv.StatusRunning {
			continue
		}
		totalQPS, doneQPS, successQPS, maxLatency, avgLatency, err := s.client.Status(group.ID)
		if err != api.OK {
			return fmt.Errorf("ConfigService Rebalance: group %d status error: %v", gid, err)
		}
		groups[gid] = &GroupRunningStatus{
			ID:         gid,
			TotalQPS:   totalQPS,
			DoneQPS:    doneQPS,
			SuccessQPS: successQPS,
			MaxLatency: maxLatency,
			AvgLatency: avgLatency,
		}
	}

	cfg := s.ctrler.Query()
	minqps, mingid, maxqps, maxgid := 1e8, -1, -1.0, -1
	for gid, status := range groups {
		if minqps < 0 || status.TotalQPS < minqps {
			minqps = status.TotalQPS
			mingid = gid
		}
		if maxqps < 0 || status.TotalQPS > maxqps {
			maxqps = status.TotalQPS
			maxgid = gid
		}
	}
	if mingid == maxgid {
		return nil
	}
	var shards []int
	for shard, gid := range cfg.Shards {
		if gid == config.Tgid(maxgid) {
			shards = append(shards, shard)
		}
	}

	move := shards[rand.Intn(len(shards))]
	cfg.Shards[move] = config.Tgid(mingid)
	s.ctrler.ChangeConfigTo(cfg)
	return nil
}
