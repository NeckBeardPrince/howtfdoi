.PHONY: build install test lint fmt clean release help

# Variables
BINARY_NAME=howtfdoi
VERSION?=1.0.4
BUILD_FLAGS=-ldflags="-s -w -X main.version=$(VERSION)"

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

build: ## Build the binary
	go build $(BUILD_FLAGS) -o $(BINARY_NAME)

install: ## Install to GOPATH/bin
	go install $(BUILD_FLAGS)

test: ## Run tests
	go test -v -race -coverprofile=coverage.txt -covermode=atomic ./...

lint: ## Run linters
	golangci-lint run

fmt: ## Format code
	go fmt ./...
	goimports -w .

clean: ## Remove build artifacts
	rm -f $(BINARY_NAME)
	rm -f coverage.txt
	rm -f howtfdoi-*

run: build ## Build and run
	./$(BINARY_NAME)

# Cross-compilation targets
build-linux: ## Build for Linux
	GOOS=linux GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BINARY_NAME)-linux-amd64

build-darwin: ## Build for macOS
	GOOS=darwin GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BINARY_NAME)-darwin-amd64
	GOOS=darwin GOARCH=arm64 go build $(BUILD_FLAGS) -o $(BINARY_NAME)-darwin-arm64

build-windows: ## Build for Windows
	GOOS=windows GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BINARY_NAME)-windows-amd64.exe

build-all: build-linux build-darwin build-windows ## Build for all platforms

release: clean build-all ## Create release artifacts
	tar czf $(BINARY_NAME)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64
	tar czf $(BINARY_NAME)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64
	tar czf $(BINARY_NAME)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64
	zip $(BINARY_NAME)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe

pre-commit-install: ## Install pre-commit hooks
	pre-commit install
	pre-commit install --hook-type commit-msg

pre-commit-run: ## Run pre-commit on all files
	pre-commit run --all-files
