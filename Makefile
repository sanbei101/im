BENCH ?= .
PPROF_DIR := .tmp/pprof
RUSTFS_PORT := 19901
RUSTFS_DATA := .tmp/rustfs/data

.PHONY: gen lint test bench verify pprof analyze clean

gen:
	kitex -type protobuf -streamx -module github.com/sanbei101/im -gen-path proto/pb proto/gateway.proto

lint:
	golangci-lint fmt
	golangci-lint run

# 归档测试需要对象存储:未运行时临时启动一个,数据落在 .tmp,测完即停
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
	go test ./... -run '^$$' -bench '$(BENCH)' -benchmem -count=1

verify: lint test bench

pprof:
	rm -rf $(PPROF_DIR) && mkdir -p $(PPROF_DIR)
	go test ./internal/store -run '^$$' -bench 'BenchmarkStoreMessageWrite/batch-100' -benchtime=5s -count=1 \
		-cpuprofile=$(PPROF_DIR)/cpu.prof \
		-memprofile=$(PPROF_DIR)/mem.prof \
		-mutexprofile=$(PPROF_DIR)/mutex.prof \
		-blockprofile=$(PPROF_DIR)/block.prof

analyze:
	go tool pprof -top -cum $(PPROF_DIR)/cpu.prof
	go tool pprof -top -alloc_space $(PPROF_DIR)/mem.prof
	go tool pprof -top -cum $(PPROF_DIR)/mutex.prof
	go tool pprof -top -cum $(PPROF_DIR)/block.prof

clean:
	rm -rf .tmp
