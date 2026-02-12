package shardkv

import (
	"github.com/khyallin/shardkv/config"
)

func (skv *ShardKV) DefaultConfig() *config.Config {
	servers := getServers(config.Gid1)
	cfg := config.MakeConfig()
	cfg.JoinBalance(map[config.Tgid][]string{
		config.Gid1: servers,
	})
	return cfg
}
