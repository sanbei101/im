GO ?= go
STORE_PKG := ./internal/store
BENCH := BenchmarkWriteMessages100Batch|BenchmarkWriteMessagesBatch|BenchmarkWriteMessage|BenchmarkReadMessages
PPROF_DIR ?= .tmp/pprof

.PHONY: fmt vet test race bench pprof analyze verify clean

fmt:
	@test -z "$$($(GO)fmt -l .)" || (echo "gofmt required"; $(GO)fmt -l .; exit 1)

vet:
	$(GO) vet ./...

test:
	$(GO) test ./...

race:
	$(GO) test -race $(STORE_PKG) -count=1

bench:
	$(GO) test $(STORE_PKG) -run '^$$' -bench '$(BENCH)' -benchmem -count=1

pprof:
	rm -rf $(PPROF_DIR)
	mkdir -p $(PPROF_DIR)
	$(GO) test $(STORE_PKG) -run '^$$' -bench '^BenchmarkWriteMessagesBatch$$' -benchtime=5s -count=1 \
		-cpuprofile=$(PPROF_DIR)/cpu.prof \
		-memprofile=$(PPROF_DIR)/mem.prof \
		-mutexprofile=$(PPROF_DIR)/mutex.prof \
		-blockprofile=$(PPROF_DIR)/block.prof

analyze:
	$(GO) tool pprof -top -cum $(PPROF_DIR)/cpu.prof
	$(GO) tool pprof -top -alloc_space $(PPROF_DIR)/mem.prof
	$(GO) tool pprof -top -cum $(PPROF_DIR)/mutex.prof
	$(GO) tool pprof -top -cum $(PPROF_DIR)/block.prof

verify: fmt vet test race bench

clean:
	rm -rf .tmp
