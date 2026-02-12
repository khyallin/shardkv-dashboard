package shardkv

import (
	"fmt"
	"strings"

	"github.com/khyallin/shardkv/config"
)

type Group struct {
	ID      config.Tgid
	Servers []string
}

func (skv *ShardKV) MakeServer(name string, gid config.Tgid, me int, servers string) string {
	containerID, _ := skv.docker.ContainerCreate(
		ImageName,
		name,
		[]string{
			"/shardkv",
			"server",
			"--gid", fmt.Sprintf("%d", gid),
			"--me", fmt.Sprintf("%d", me),
			"--servers", servers,
		},
	)
	return containerID
}

func (skv *ShardKV) MakeGroup(gid config.Tgid) *Group {
	servers := getServers(gid)
	serversStr := strings.Join(servers, ",")

	containerIDs := make([]string, NServer)
	for i := range containerIDs {
		containerIDs[i] = skv.MakeServer(servers[i], gid, i, serversStr)
	}
	return &Group{
		ID:      gid,
		Servers: containerIDs,
	}
}

func (skv *ShardKV) RunGroup(group *Group) {
	for _, server := range group.Servers {
		skv.docker.ContainerStart(server)
	}
}

func (skv *ShardKV) StopGroup(group *Group) {
	for _, server := range group.Servers {
		skv.docker.ContainerStop(server)
	}
}

func (skv *ShardKV) RemoveGroup(group *Group) {
	for _, server := range group.Servers {
		skv.docker.ContainerRemove(server)
	}
}
