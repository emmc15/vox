# Diaz - Speech-to-Text Application Makefile
# Builds self-contained binaries for multiple platforms

# Application name
APP_NAME := diaz
VERSION := 0.1.0

# Directories
BUILD_DIR := build
CMD_DIR := cmd/$(APP_NAME)
INTERNAL_DIR := internal
MODELS_DIR := models

# Build info
BUILD_TIME := $(shell date -u '+%Y-%m-%d_%H:%M:%S')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_BRANCH := $(shell git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown")

# Go build flags
LDFLAGS := -s -w \
	-X main.Version=$(VERSION) \
	-X main.BuildTime=$(BUILD_TIME) \
	-X main.GitCommit=$(GIT_COMMIT) \
	-X main.GitBranch=$(GIT_BRANCH)

# CGO flags for static linking
CGO_ENABLED := 1
CGO_LDFLAGS_LINUX := -static
CGO_LDFLAGS_DARWIN :=
CGO_LDFLAGS_WINDOWS := -static

# Platform-specific variables
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# Binary names
BINARY_LINUX_AMD64 := $(BUILD_DIR)/$(APP_NAME)-linux-amd64
BINARY_LINUX_ARM64 := $(BUILD_DIR)/$(APP_NAME)-linux-arm64
BINARY_DARWIN_AMD64 := $(BUILD_DIR)/$(APP_NAME)-darwin-amd64
BINARY_DARWIN_ARM64 := $(BUILD_DIR)/$(APP_NAME)-darwin-arm64
BINARY_WINDOWS_AMD64 := $(BUILD_DIR)/$(APP_NAME)-windows-amd64.exe

# Colors for output
COLOR_RESET := \033[0m
COLOR_BOLD := \033[1m
COLOR_GREEN := \033[32m
COLOR_YELLOW := \033[33m
COLOR_BLUE := \033[34m

.PHONY: all build clean test fmt vet lint deps help install run dev
.PHONY: build-linux build-darwin build-windows build-all
.PHONY: docker-build docker-build-all

## Default target
all: clean deps build

## help: Display this help message
help:
	@echo "$(COLOR_BOLD)Diaz - Speech-to-Text Application$(COLOR_RESET)"
	@echo ""
	@echo "$(COLOR_BOLD)Available targets:$(COLOR_RESET)"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/##/$(COLOR_GREEN)/' | column -t -s ':' | sed 's/$$/$(COLOR_RESET)/'

## build: Build binary for current platform
build: deps
	@echo "$(COLOR_BLUE)Building $(APP_NAME) for $(GOOS)/$(GOARCH)...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=$(CGO_ENABLED) \
	GOOS=$(GOOS) \
	GOARCH=$(GOARCH) \
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) ./$(CMD_DIR)
	@echo "$(COLOR_GREEN)Build complete: $(BUILD_DIR)/$(APP_NAME)$(COLOR_RESET)"

## build-linux: Build for Linux AMD64 (statically linked)
build-linux:
	@echo "$(COLOR_BLUE)Building for Linux AMD64 (static)...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 \
	GOOS=linux \
	GOARCH=amd64 \
	CC=gcc \
	CGO_LDFLAGS="$(CGO_LDFLAGS_LINUX)" \
	go build -ldflags "$(LDFLAGS) -extldflags '-static'" \
		-tags 'osusergo netgo static_build' \
		-o $(BINARY_LINUX_AMD64) ./$(CMD_DIR)
	@echo "$(COLOR_GREEN)Built: $(BINARY_LINUX_AMD64)$(COLOR_RESET)"

## build-linux-arm64: Build for Linux ARM64
build-linux-arm64:
	@echo "$(COLOR_BLUE)Building for Linux ARM64...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 \
	GOOS=linux \
	GOARCH=arm64 \
	CC=aarch64-linux-gnu-gcc \
	go build -ldflags "$(LDFLAGS)" \
		-o $(BINARY_LINUX_ARM64) ./$(CMD_DIR)
	@echo "$(COLOR_GREEN)Built: $(BINARY_LINUX_ARM64)$(COLOR_RESET)"

## build-darwin: Build for macOS AMD64
build-darwin:
	@echo "$(COLOR_BLUE)Building for macOS AMD64...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 \
	GOOS=darwin \
	GOARCH=amd64 \
	go build -ldflags "$(LDFLAGS)" \
		-o $(BINARY_DARWIN_AMD64) ./$(CMD_DIR)
	@echo "$(COLOR_GREEN)Built: $(BINARY_DARWIN_AMD64)$(COLOR_RESET)"

## build-darwin-arm64: Build for macOS ARM64 (Apple Silicon)
build-darwin-arm64:
	@echo "$(COLOR_BLUE)Building for macOS ARM64 (Apple Silicon)...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 \
	GOOS=darwin \
	GOARCH=arm64 \
	go build -ldflags "$(LDFLAGS)" \
		-o $(BINARY_DARWIN_ARM64) ./$(CMD_DIR)
	@echo "$(COLOR_GREEN)Built: $(BINARY_DARWIN_ARM64)$(COLOR_RESET)"

## build-windows: Build for Windows AMD64
build-windows:
	@echo "$(COLOR_BLUE)Building for Windows AMD64...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	CGO_ENABLED=1 \
	GOOS=windows \
	GOARCH=amd64 \
	CC=x86_64-w64-mingw32-gcc \
	go build -ldflags "$(LDFLAGS)" \
		-o $(BINARY_WINDOWS_AMD64) ./$(CMD_DIR)
	@echo "$(COLOR_GREEN)Built: $(BINARY_WINDOWS_AMD64)$(COLOR_RESET)"

## build-all: Build for all platforms
build-all: build-linux build-darwin build-darwin-arm64 build-windows
	@echo "$(COLOR_GREEN)All platform builds complete!$(COLOR_RESET)"
	@ls -lh $(BUILD_DIR)/

## docker-build-linux: Build Linux binary using Docker (ensures static linking)
docker-build-linux:
	@echo "$(COLOR_BLUE)Building Linux binary in Docker...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	docker run --rm \
		-v $(PWD):/workspace \
		-w /workspace \
		-e CGO_ENABLED=1 \
		golang:1.21-alpine \
		sh -c 'apk add --no-cache gcc musl-dev && \
		       go build -ldflags "$(LDFLAGS) -extldflags \"-static\"" \
		       -tags "osusergo netgo static_build" \
		       -o $(BINARY_LINUX_AMD64) ./$(CMD_DIR)'
	@echo "$(COLOR_GREEN)Docker build complete: $(BINARY_LINUX_AMD64)$(COLOR_RESET)"

## docker-build-all: Build all binaries using Docker
docker-build-all:
	@echo "$(COLOR_YELLOW)Note: Docker cross-compilation requires additional setup$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)Currently only building Linux AMD64 in Docker$(COLOR_RESET)"
	@$(MAKE) docker-build-linux

## install: Install binary to system
install: build
	@echo "$(COLOR_BLUE)Installing $(APP_NAME)...$(COLOR_RESET)"
	install -d $(DESTDIR)/usr/local/bin
	install -m 755 $(BUILD_DIR)/$(APP_NAME) $(DESTDIR)/usr/local/bin/
	@echo "$(COLOR_GREEN)Installed to /usr/local/bin/$(APP_NAME)$(COLOR_RESET)"

## uninstall: Uninstall binary from system
uninstall:
	@echo "$(COLOR_BLUE)Uninstalling $(APP_NAME)...$(COLOR_RESET)"
	rm -f /usr/local/bin/$(APP_NAME)
	@echo "$(COLOR_GREEN)Uninstalled$(COLOR_RESET)"

## run: Build and run the application
run: build
	@echo "$(COLOR_BLUE)Running $(APP_NAME)...$(COLOR_RESET)"
	./$(BUILD_DIR)/$(APP_NAME)

## dev: Run in development mode (with race detector)
dev:
	@echo "$(COLOR_BLUE)Running in development mode...$(COLOR_RESET)"
	go run -race ./$(CMD_DIR)

## test: Run all tests
test:
	@echo "$(COLOR_BLUE)Running tests...$(COLOR_RESET)"
	go test -v -race -coverprofile=coverage.out ./...
	@echo "$(COLOR_GREEN)Tests complete$(COLOR_RESET)"

## test-coverage: Run tests and show coverage
test-coverage: test
	@echo "$(COLOR_BLUE)Generating coverage report...$(COLOR_RESET)"
	go tool cover -html=coverage.out -o coverage.html
	@echo "$(COLOR_GREEN)Coverage report: coverage.html$(COLOR_RESET)"

## bench: Run benchmarks
bench:
	@echo "$(COLOR_BLUE)Running benchmarks...$(COLOR_RESET)"
	go test -bench=. -benchmem ./...

## fmt: Format code
fmt:
	@echo "$(COLOR_BLUE)Formatting code...$(COLOR_RESET)"
	go fmt ./...
	@echo "$(COLOR_GREEN)Code formatted$(COLOR_RESET)"

## vet: Run go vet
vet:
	@echo "$(COLOR_BLUE)Running go vet...$(COLOR_RESET)"
	go vet ./...
	@echo "$(COLOR_GREEN)Vet complete$(COLOR_RESET)"

## lint: Run linters (requires golangci-lint)
lint:
	@echo "$(COLOR_BLUE)Running linters...$(COLOR_RESET)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run ./...; \
		echo "$(COLOR_GREEN)Lint complete$(COLOR_RESET)"; \
	else \
		echo "$(COLOR_YELLOW)golangci-lint not installed. Skipping...$(COLOR_RESET)"; \
		echo "Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

## deps: Download and tidy dependencies
deps:
	@echo "$(COLOR_BLUE)Downloading dependencies...$(COLOR_RESET)"
	go mod download
	go mod tidy
	@echo "$(COLOR_GREEN)Dependencies ready$(COLOR_RESET)"

## deps-update: Update all dependencies
deps-update:
	@echo "$(COLOR_BLUE)Updating dependencies...$(COLOR_RESET)"
	go get -u ./...
	go mod tidy
	@echo "$(COLOR_GREEN)Dependencies updated$(COLOR_RESET)"

## clean: Remove build artifacts
clean:
	@echo "$(COLOR_BLUE)Cleaning build artifacts...$(COLOR_RESET)"
	rm -rf $(BUILD_DIR)
	rm -f coverage.out coverage.html
	@echo "$(COLOR_GREEN)Clean complete$(COLOR_RESET)"

## clean-all: Remove build artifacts and downloaded models
clean-all: clean
	@echo "$(COLOR_BLUE)Cleaning models...$(COLOR_RESET)"
	rm -rf $(MODELS_DIR)/*.zip
	@echo "$(COLOR_GREEN)Full clean complete$(COLOR_RESET)"

## models-download: Download speech recognition models
models-download:
	@echo "$(COLOR_BLUE)Downloading models...$(COLOR_RESET)"
	@mkdir -p $(MODELS_DIR)
	@echo "$(COLOR_YELLOW)Model download to be implemented$(COLOR_RESET)"
	@echo "$(COLOR_YELLOW)Manual download: https://alphacephei.com/vosk/models$(COLOR_RESET)"

## size: Show binary sizes
size:
	@echo "$(COLOR_BOLD)Binary sizes:$(COLOR_RESET)"
	@ls -lh $(BUILD_DIR)/ 2>/dev/null || echo "No binaries found. Run 'make build' first."

## info: Show build information
info:
	@echo "$(COLOR_BOLD)Build Information:$(COLOR_RESET)"
	@echo "  App Name:    $(APP_NAME)"
	@echo "  Version:     $(VERSION)"
	@echo "  Git Commit:  $(GIT_COMMIT)"
	@echo "  Git Branch:  $(GIT_BRANCH)"
	@echo "  Build Time:  $(BUILD_TIME)"
	@echo "  GOOS:        $(GOOS)"
	@echo "  GOARCH:      $(GOARCH)"
	@echo "  CGO Enabled: $(CGO_ENABLED)"

## check: Run all checks (fmt, vet, lint, test)
check: fmt vet lint test
	@echo "$(COLOR_GREEN)All checks passed!$(COLOR_RESET)"

## release: Build release binaries for all platforms
release: clean check build-all
	@echo "$(COLOR_GREEN)Release build complete!$(COLOR_RESET)"
	@$(MAKE) size

## quick: Quick build without dependencies check
quick:
	@echo "$(COLOR_BLUE)Quick build...$(COLOR_RESET)"
	@mkdir -p $(BUILD_DIR)
	go build -ldflags "$(LDFLAGS)" -o $(BUILD_DIR)/$(APP_NAME) ./$(CMD_DIR)
	@echo "$(COLOR_GREEN)Quick build complete$(COLOR_RESET)"

.DEFAULT_GOAL := help
