init-network:
	sudo docker network create shardkv-net

init-shardkv:
	go get github.com/khyallin/shardkv
	go mod tidy

init: init-network init-shardkv

image:
	sudo docker build -t khyallin/shardkv-dashboard .
	docker push khyallin/shardkv-dashboard

run:
	sudo docker run -d --name shardkv-dashboard --network shardkv-net -v /var/run/docker.sock:/var/run/docker.sock -p 8080:8080 khyallin/shardkv-dashboard

clear:
	sudo docker rm -f $$(docker ps -aq)

debug: image run

all: init debug clear