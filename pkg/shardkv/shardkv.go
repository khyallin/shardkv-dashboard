package shardkv

import (
	"github.com/khyallin/shardkv-dashboard/pkg/docker"
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
