package shardkv

import (
	"fmt"

	"github.com/khyallin/shardkv-dashboard/docker"
	"github.com/khyallin/shardkv/config"
)

type ShardKV struct {
	docker *docker.Docker
}

const (
	NServer   = 5
	NGroup    = 10
	ImageName = "khyallin/shardkv"
)

func New() *ShardKV {
	d := docker.New()
	if !d.ImagePull(ImageName) {
		d.Close()
		return nil
	}
	return &ShardKV{docker: d}
}

func (skv *ShardKV) Close() {
	skv.docker.Close()
}

func getServers(gid config.Tgid) []string {
	servers := make([]string, NServer)
	for i := range servers {
		servers[i] = fmt.Sprintf("shardkv-server-%d-%d", gid, i)
	}
	return servers
}
