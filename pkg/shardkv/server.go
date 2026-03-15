package shardkv

import (
	"fmt"
	"strings"

	"github.com/khyallin/shardkv/config"
)

type Status int

const (
	StatusInit Status = iota
	StatusRunning
	StatusStopped
	StatusRemoved
)

type Group struct {
	ID      config.Tgid
	Status  Status
	Servers []string
}

func GetServers(gid config.Tgid) []string {
	servers := make([]string, NServer)
	for i := range servers {
		servers[i] = fmt.Sprintf("shardkv-server-%d-%d", gid, i)
	}
	return servers
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
	servers := GetServers(gid)
	serversStr := strings.Join(servers, ",")

	containerIDs := make([]string, NServer)
	for i := range containerIDs {
		containerIDs[i] = skv.MakeServer(servers[i], gid, i, serversStr)
	}
	return &Group{
		ID:      gid,
		Status:  StatusInit,
		Servers: containerIDs,
	}
}

func (skv *ShardKV) RunGroup(group *Group) {
	if group.Status != StatusInit && group.Status != StatusStopped {
		return
	}
	group.Status = StatusRunning
	for _, server := range group.Servers {
		skv.docker.ContainerStart(server)
	}
}

func (skv *ShardKV) StopGroup(group *Group) {
	if group.Status != StatusRunning {
		return
	}
	group.Status = StatusStopped
	for _, server := range group.Servers {
		skv.docker.ContainerStop(server)
	}
}

func (skv *ShardKV) RemoveGroup(group *Group) {
	if group.Status != StatusStopped {
		return
	}
	group.Status = StatusRemoved
	for _, server := range group.Servers {
		skv.docker.ContainerRemove(server)
	}
}
