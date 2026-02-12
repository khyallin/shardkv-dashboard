package shardkv

import (
	"github.com/khyallin/shardkv/client"
	"github.com/khyallin/shardkv/config"
)

func (skv *ShardKV) MakeClient() *client.Clerk {
	servers := getServers(config.Gid0)
	return client.MakeClerk(servers)
}
