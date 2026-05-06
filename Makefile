BASE ?= http://127.0.0.1:8080
OUT ?= tmp/stress_results.jsonl
OUT_REBALANCE ?= tmp/rebalance_results.jsonl
FIG_DIR ?= tmp
TARGET_GID ?= 1
DURATION ?= 30
REPEAT ?= 3
PROFILE ?= full
PYTHON ?= python3

.PHONY: init-network init-shardkv init image build run clear debug restart all stress-precheck stress rebalance draw draw-rebalance bench bench-quick bench-rebalance

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

restart: clear build run

all: init build run

stress-precheck:
	curl --noproxy '*' -fsS $(BASE)/ping >/dev/null

stress: stress-precheck
	mkdir -p $(dir $(OUT))
	BASE=$(BASE) OUT=$(OUT) TARGET_GID=$(TARGET_GID) DURATION=$(DURATION) REPEAT=$(REPEAT) PROFILE=$(PROFILE) bash scripts/stress.sh

rebalance: stress-precheck
	mkdir -p $(dir $(OUT_REBALANCE))
	BASE=$(BASE) OUT=$(OUT_REBALANCE) TARGET_GID=$(TARGET_GID) DURATION=$(DURATION) REPEAT=$(REPEAT) PROFILE=$(PROFILE) bash scripts/rebalance.sh

draw:
	mkdir -p $(FIG_DIR)
	OUT=$(OUT) FIG_DIR=$(FIG_DIR) $(PYTHON) scripts/draw.py

draw-rebalance:
	mkdir -p $(FIG_DIR)
	OUT=$(OUT_REBALANCE) FIG_DIR=$(FIG_DIR) $(PYTHON) scripts/draw_rebalance.py

bench: stress draw

bench-quick:
	$(MAKE) stress DURATION=1 PROFILE=quick REPEAT=1
	$(MAKE) draw

bench-rebalance-quick:
	$(MAKE) rebalance DURATION=1 PROFILE=quick REPEAT=1
	$(MAKE) draw-rebalance

bench-rebalance:
	$(MAKE) rebalance DURATION=60 PROFILE=full REPEAT=5
	$(MAKE) draw-rebalance
