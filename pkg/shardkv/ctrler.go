package shardkv

import (
	"github.com/khyallin/shardkv/config"
	"github.com/khyallin/shardkv/controller"
)

func MakeCtrler() *controller.Controller {
	servers := GetServers(config.Gid0)
	return controller.MakeController(servers)
}
