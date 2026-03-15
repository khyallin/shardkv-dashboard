package docker

import (
	"context"
	"io"
	"log"
	"os"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

type Docker struct {
	client *client.Client
}

func New() *Docker {
	log.Print("Docker.New()|Start")
	defer log.Print("Docker.New()|End")

	client, err := client.New(client.FromEnv)
	if err != nil {
		log.Fatalf("Docker.New()|Fail|err=%v", err)
	}
	return &Docker{client: client}
}

func (d *Docker) Close() {
	d.client.Close()
}

func (d *Docker) ImagePull(imageName string) bool {
	log.Printf("Docker.ImagePull()|Start|name=%s", imageName)
	defer log.Printf("Docker.ImagePull()|End|name=%s", imageName)

	resp, err := d.client.ImagePull(context.Background(), imageName, client.ImagePullOptions{})
	if err != nil {
		log.Printf("Docker.ImagePull()|Fail|name=%s|err=%v", imageName, err)
		return false
	}
	defer resp.Close()
	io.Copy(os.Stdout, resp)
	return true
}

func (d *Docker) ContainerCreate(imageName, containerName string, cmd []string) (string, bool) {
	log.Printf("Docker.ContainerCreate()|Start|imageName=%s|containerName=%s", imageName, containerName)
	defer log.Printf("Docker.ContainerCreate()|End|imageName=%s|containerName=%s", imageName, containerName)

	createResp, err := d.client.ContainerCreate(context.Background(), client.ContainerCreateOptions{
		Config: &container.Config{
			Image: imageName,
			Env:   []string{"DEBUG=1"},
			Cmd:   cmd,
		},
		NetworkingConfig: &network.NetworkingConfig{
			EndpointsConfig: map[string]*network.EndpointSettings{
				"shardkv-net": {},
			},
		},
		Name: containerName,
	})
	if err != nil {
		log.Printf("Docker.ContainerCreate()|Fail|err=%v|createResp=%+v", err, createResp)
		return "", false
	}
	return createResp.ID, true
}

func (d *Docker) ContainerStart(containerID string) bool {
	log.Printf("Docker.ContainerStart()|Start|containerID=%s", containerID)
	defer log.Printf("Docker.ContainerStart()|End|containerID=%s", containerID)

	_, err := d.client.ContainerStart(context.Background(), containerID, client.ContainerStartOptions{})
	if err != nil {
		log.Printf("Docker.ContainerStart()|Fail|containerID=%s|err=%v", containerID, err)
		return false
	}
	return true
}

func (d *Docker) ContainerStop(containerID string) bool {
	log.Printf("Docker.ContainerStop()|Start|containerID=%s", containerID)
	defer log.Printf("Docker.ContainerStop()|End|containerID=%s", containerID)

	_, err := d.client.ContainerStop(context.Background(), containerID, client.ContainerStopOptions{})
	if err != nil {
		log.Printf("Docker.ContainerStop()|Fail|containerID=%s|err=%v", containerID, err)
		return false
	}
	return true
}

func (d *Docker) ContainerRemove(containerID string) bool {
	log.Printf("Docker.ContainerRemove()|Start|containerID=%s", containerID)
	defer log.Printf("Docker.ContainerRemove()|End|containerID=%s", containerID)

	_, err := d.client.ContainerRemove(context.Background(), containerID, client.ContainerRemoveOptions{Force: true})
	if err != nil {
		log.Printf("Docker.ContainerRemove()|Fail|containerID=%s|err=%v", containerID, err)
		return false
	}
	return true
}
