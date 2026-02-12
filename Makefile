image:
	sudo docker build -t khyallin/shardkv-dashboard .

run:
	sudo docker run --network shardkv-net -v /var/run/docker.sock:/var/run/docker.sock khyallin/shardkv-dashboard

all: image run