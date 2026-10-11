BENCH ?= .
PPROF_DIR := .tmp/pprof
RUSTFS_PORT := 19901
RUSTFS_DATA := .tmp/rustfs/data

.PHONY: gen lint test bench verify pprof analyze clean dev

dev:
	for p in 19901 9000 8801 8800; do fuser -kn tcp $$p 2>/dev/null && echo "killed on $$p" || true; done
	docker compose up -d --build
	docker compose ps
	docker compose logs --tail 50

gen:
	kitex -type protobuf -streamx -module github.com/sanbei101/im -gen-path proto/pb proto/gateway.proto

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
