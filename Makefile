BENCH ?= .
PPROF_DIR := .tmp/pprof
BIN_DIR := bin
RUSTFS_PORT := 19901
RUSTFS_DATA := .tmp/rustfs/data

.PHONY: gen lint test bench verify pprof analyze clean build dev stop

build:
	@mkdir -p $(BIN_DIR)
	go build -trimpath -ldflags="-s -w" -o $(BIN_DIR)/meta ./cmd/meta
go build -trimpath -ldflags="-s -w" -o $(BIN_DIR)/api ./cmd/api
	go build -trimpath -ldflags="-s -w" -o $(BIN_DIR)/gateway ./cmd/gateway

dev: build
	@mkdir -p .tmp
	for p in 9000 8801 8802 8800; do fuser -kn tcp $$p 2>/dev/null && echo "killed on $$p" || true; done
	@nohup ./$(BIN_DIR)/meta > .tmp/meta.log 2>&1 & echo $$! > .tmp/meta.pid
	@nohup ./$(BIN_DIR)/api > .tmp/api.log 2>&1 & echo $$! > .tmp/api.pid
	@nohup ./$(BIN_DIR)/gateway > .tmp/gateway.log 2>&1 & echo $$! > .tmp/gateway.pid
	@sleep 1
	@echo "meta log: .tmp/meta.log"; tail -n 20 .tmp/meta.log
	@echo "api log: .tmp/api.log"; tail -n 20 .tmp/api.log
	@echo "gateway log: .tmp/gateway.log"; tail -n 20 .tmp/gateway.log

stop:
	@for f in .tmp/meta.pid .tmp/api.pid .tmp/gateway.pid; do \
		if test -f $$f; then kill $$(cat $$f) 2>/dev/null; rm -f $$f; fi; \
	done

gen:
	kitex -type protobuf -streamx -module github.com/sanbei101/im -gen-path proto/pb proto/gateway.proto
	kitex -type protobuf -module github.com/sanbei101/im -gen-path proto/pb proto/meta.proto

lint:
	golangci-lint fmt
	golangci-lint run

test:
	@mkdir -p $(RUSTFS_DATA)
	@if ! curl -s --max-time 1 -o /dev/null http://127.0.0.1:$(RUSTFS_PORT)/; then \
		RUSTFS_CONSOLE_ENABLE=false rustfs server --address 127.0.0.1:$(RUSTFS_PORT) \
			--access-key rustfsadmin --secret-key rustfsadmin $(RUSTFS_DATA) \
			>.tmp/rustfs.log 2>&1 & echo $$! > .tmp/rustfs.pid; \
	fi
	@until curl -s --max-time 1 -o /dev/null http://127.0.0.1:$(RUSTFS_PORT)/; do sleep 0.2; done
	@go test -race ./...; status=$$?; \
		if test -f .tmp/rustfs.pid; then kill $$(cat .tmp/rustfs.pid) 2>/dev/null; rm -f .tmp/rustfs.pid; fi; \
		exit $$status

bench:
	go test -p 4 ./... -run '^$$' -bench '$(BENCH)' -benchmem -count=1

pprof:
	@mkdir -p $(PPROF_DIR)
	go test -p 4 ./internal/store -run '^$$' -bench '$(BENCH)' -benchmem -count=1 \
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
