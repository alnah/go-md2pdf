BINARY := go-md2pdf

.PHONY: help build test test-integration test-cover test-cover-all run clean fmt vet lint sec check check-all tools

.DEFAULT_GOAL := help

help: ## Display this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-15s\033[0m %s\n", $$1, $$2}'

tools: ## Install development tools (staticcheck, gosec)
	go get -tool honnef.co/go/tools/cmd/staticcheck
	go get -tool github.com/securego/gosec/v2/cmd/gosec

deps: ## Install dependencies (chromedp, go-yaml, pflag, goldmark)
	go get github.com/chromedp/chromedp
	go get github.com/goccy/go-yaml
	go get https://github.com/spf13/pflag
	go get https://github.com/yuin/goldmark

build: ## Build the binary
	go build -o $(BINARY) .

test: ## Run unit tests
	go test -v ./...

test-integration: ## Run integration tests (requires pandoc, chrome)
	go test -v -tags=integration ./...

test-cover: ## Run unit tests with coverage report
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

test-cover-all: ## Run all tests with coverage report (requires pandoc)
	go test -v -tags=integration -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

run: build ## Build and run the shell
	./$(BINARY)

clean: ## Remove build artifacts
	rm -f $(BINARY) coverage.out coverage.html

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
