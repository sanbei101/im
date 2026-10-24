BENCH ?= .
PPROF_DIR := .tmp/pprof
BIN_DIR := bin

.PHONY: gen lint test bench verify pprof analyze clean build dev stop

build:
	@mkdir -p $(BIN_DIR)
	go build -trimpath -ldflags="-s -w" -o $(BIN_DIR)/api ./cmd/api
	go build -trimpath -ldflags="-s -w" -o $(BIN_DIR)/gateway ./cmd/gateway

dev: build
	@mkdir -p .tmp
	for p in 9000 8801 8800; do fuser -kn tcp $$p 2>/dev/null && echo "killed on $$p" || true; done
	@nohup ./$(BIN_DIR)/api > .tmp/api.log 2>&1 & echo $$! > .tmp/api.pid
	@nohup ./$(BIN_DIR)/gateway > .tmp/gateway.log 2>&1 & echo $$! > .tmp/gateway.pid
	@sleep 1
	@echo "api log: .tmp/api.log"; tail -n 20 .tmp/api.log
	@echo "gateway log: .tmp/gateway.log"; tail -n 20 .tmp/gateway.log

stop:
	@for f in .tmp/api.pid .tmp/gateway.pid; do \
		if test -f $$f; then kill $$(cat $$f) 2>/dev/null; rm -f $$f; fi; \
	done

gen:
	kitex -type protobuf -streamx -module github.com/sanbei101/im -gen-path proto/pb proto/gateway.proto

lint:
	golangci-lint fmt
	golangci-lint run

test:
	@if test -z "$$TEST_POSTGRES_URL"; then \
		for db in im_test im_api_test im_store_test; do \
			psql -d postgres -Atqc "SELECT 1 FROM pg_database WHERE datname='$$db'" | grep -q 1 || createdb $$db; \
		done; \
	fi
	@TEST_POSTGRES_URL="$${TEST_POSTGRES_URL:-postgres://127.0.0.1:5432/im_test?sslmode=disable}" \
		go test -p 1 -race ./...

bench:
	@if test -z "$$TEST_POSTGRES_URL"; then \
		psql -d postgres -Atqc "SELECT 1 FROM pg_database WHERE datname='im_test'" | grep -q 1 || createdb im_test; \
	fi
	@TEST_POSTGRES_URL="$${TEST_POSTGRES_URL:-postgres://127.0.0.1:5432/im_test?sslmode=disable}" \
		go test -p 4 ./... -run '^$$' -bench '$(BENCH)' -benchmem -count=1

pprof:
	@mkdir -p $(PPROF_DIR)
	@TEST_POSTGRES_URL="$${TEST_POSTGRES_URL:-postgres://127.0.0.1:5432/im_test?sslmode=disable}" \
		go test -p 4 ./pkg/engine -run '^$$' -bench '$(BENCH)' -benchmem -count=1 \
		-cpuprofile $(PPROF_DIR)/cpu.prof \
		-memprofile $(PPROF_DIR)/mem.prof \
		-mutexprofile $(PPROF_DIR)/mutex.prof \
		-blockprofile $(PPROF_DIR)/block.prof

verify: lint test bench

analyze:
	go tool pprof -top -cum $(PPROF_DIR)/cpu.prof
	go tool pprof -top -alloc_space $(PPROF_DIR)/mem.prof
	go tool pprof -top -cum $(PPROF_DIR)/mutex.prof
	go tool pprof -top -cum $(PPROF_DIR)/block.prof

clean:
	rm -rf .tmp
