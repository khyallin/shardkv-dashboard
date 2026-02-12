package main

import (
	"log"

	"github.com/khyallin/shardkv-dashboard/shardkv"
	"github.com/khyallin/shardkv/api"
	"github.com/khyallin/shardkv/config"
)

func main() {
	skv := shardkv.New()
	defer skv.Close()

	groups := make(map[config.Tgid]*shardkv.Group)
	for gid := config.Gid0; gid <= config.Gid1; gid++ {
		groups[gid] = skv.MakeGroup(gid)
	}
	for _, group := range groups {
		skv.RunGroup(group)
	}

	ctrler := skv.MakeCtrler()
	config := skv.DefaultConfig()
	ctrler.InitConfig(config)

	client := skv.MakeClient()
	err := client.Put("key", "value", 0)
	if err != api.OK {
		log.Printf("Put error: %v", err)
	}
	value, version, err := client.Get("key")
	if err != api.OK {
		log.Printf("Get error: %v", err)
	}
	log.Printf("Get key=%s, value=%s, version=%d", "key", value, version)

	for _, group := range groups {
		skv.StopGroup(group)
		skv.RemoveGroup(group)
	}
}
