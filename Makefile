init-network:
	docker network create shardkv-net

init-shardkv:
	go get github.com/khyallin/shardkv
	go mod tidy

init: init-network init-shardkv

image:
	docker pull golang:1.25-alpine
	docker build -t khyallin/shardkv-dashboard .
	docker push khyallin/shardkv-dashboard

run:
	docker run -d --name shardkv-dashboard --network shardkv-net -v /var/run/docker.sock:/var/run/docker.sock -p 8080:8080 khyallin/shardkv-dashboard

clear:
	docker rm -f $$(docker ps -aq)

debug: image run

all: init debug clear