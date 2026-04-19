init-network:
	docker network create shardkv-net

init-shardkv:
	GONOPROXY=github.com/khyallin/* go get github.com/khyallin/shardkv@latest
	go mod tidy

init: init-network init-shardkv

image:
	docker pull golang:1.25-alpine
	docker build -t khyal/shardkv-dashboard .
	docker push khyal/shardkv-dashboard

build:
	docker build -t khyal/shardkv-dashboard .

run:
	docker run -d --name shardkv-dashboard --network shardkv-net -v /var/run/docker.sock:/var/run/docker.sock -p 8080:8080 khyal/shardkv-dashboard

clear:
	docker rm -f $$(docker ps -aq)

debug: build run

restart: clear debug
