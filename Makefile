.PHONY: all build test bench run-gateway run-upstream docker-up docker-down clean

BIN_DIR := ./bin

all: build test

build:
	@mkdir -p $(BIN_DIR)
	go build -o $(BIN_DIR)/gateway ./cmd/gateway
	go build -o $(BIN_DIR)/upstream ./cmd/mock-upstream
	@echo "Binaries compiled to $(BIN_DIR)/"

test:
	go test -v -race ./pkg/...

bench:
	@echo "Running Comparative Algorithm Benchmarks with Memory Profiling..."
	go test -v -bench=. -benchmem ./pkg/limiter/

run-gateway:
	go run ./cmd/gateway -addr :8080 -upstream http://localhost:8081

run-upstream:
	go run ./cmd/mock-upstream -port 8081

docker-up:
	docker compose up --build

docker-down:
	docker compose down

clean:
	rm -rf $(BIN_DIR)
