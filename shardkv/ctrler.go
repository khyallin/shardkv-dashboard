package shardkv

import (
	"github.com/khyallin/shardkv/config"
	"github.com/khyallin/shardkv/controller"
)

func (skv *ShardKV) MakeCtrler() *controller.Controller {
	servers := getServers(config.Gid0)
	return controller.MakeController(servers)
}