BINARY := md2pdf

.PHONY: help build test test-integration test-cover test-cover-all bench bench-cpu bench-mem run clean fmt vet lint sec check check-all tools

.DEFAULT_GOAL := help

help: ## Display this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

tools: ## Install development tools (staticcheck, gosec)
	go get -tool honnef.co/go/tools/cmd/staticcheck
	go get -tool github.com/securego/gosec/v2/cmd/gosec

deps: ## Install dependencies (go-rod, go-yaml, pflag, goldmark, goldmark-highlighting, automaxprocs)
	go get github.com/go-rod/rod
	go get github.com/goccy/go-yaml
	go get github.com/spf13/pflag
	go get github.com/yuin/goldmark
	go get github.com/yuin/goldmark-highlighting/v2
	go get go.uber.org/automaxprocs

build: ## Build the binary
	go build -o $(BINARY) ./cmd/md2pdf

test: ## Run unit tests
	go test -v ./...

test-integration: ## Run integration tests (rod auto-downloads chromium)
	go test -v -tags=integration ./...

test-cover: ## Run unit tests with coverage report
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test-cover-all: ## Run all tests with coverage report
	go test -v -tags=integration -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

bench: ## Run benchmarks
	go test -tags=bench -bench=. -benchmem ./...

bench-cpu: ## Run benchmarks with CPU profiling
	go test -tags=bench -bench=. -benchmem -cpuprofile=cpu.prof ./...
	@echo "Run 'go tool pprof cpu.prof' to analyze"

bench-mem: ## Run benchmarks with memory profiling
	go test -tags=bench -bench=. -benchmem -memprofile=mem.prof ./...
	@echo "Run 'go tool pprof mem.prof' to analyze"

bench-compare: ## Compare benchmarks (usage: make bench-compare OLD=old.txt NEW=new.txt)
	@if [ -z "$(OLD)" ] || [ -z "$(NEW)" ]; then \
		echo "Usage: make bench-compare OLD=old.txt NEW=new.txt"; \
		exit 1; \
	fi
	go run golang.org/x/perf/cmd/benchstat@latest $(OLD) $(NEW)

run: build ## Build and run the shell
	./$(BINARY)

clean: ## Remove build artifacts
	rm -f $(BINARY) coverage.out coverage.html cpu.prof mem.prof

fmt: ## Format source code
	go fmt ./...

vet: ## Run go vet for static analysis
	go vet ./...

lint: ## Run staticcheck linter
	go tool staticcheck ./...

sec: ## Run gosec security scanner
	go tool gosec ./...

check: fmt vet lint sec test ## Run all checks (unit tests only)

check-all: fmt vet lint sec test-integration ## Run all checks including integration tests
