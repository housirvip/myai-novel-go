.PHONY: help build server cli test test-race vet lint fmt run clean docker docker-up docker-down

help:
	@echo "make build        # 编译 server + novel 双二进制"
	@echo "make server       # 仅编译 server"
	@echo "make cli          # 仅编译 novel"
	@echo "make test         # go test ./..."
	@echo "make test-race    # go test -race -count=1 ./..."
	@echo "make vet          # go vet ./..."
	@echo "make fmt          # gofmt -w ."
	@echo "make run          # 起服务(走 .env 默认配置)"
	@echo "make docker       # docker build"
	@echo "make docker-up    # docker compose up -d"
	@echo "make docker-down  # docker compose down -v"
	@echo "make clean        # 清理 bin/ 与 SQLite db"

build: server cli

server:
	go build -o bin/server ./cmd/server

cli:
	go build -o bin/novel ./cmd/novel

test:
	go test ./...

test-race:
	go test -race -count=1 ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

run: server
	./bin/server

docker:
	docker build -t myai-novel-go:dev .

docker-up:
	docker compose up -d --build

docker-down:
	docker compose down -v

clean:
	rm -rf bin/
	rm -f data/novel.db data/cli-smoke.db
