# VPN Route Manager Makefile

# Variables
BINARY_NAME=vpn-route-manager
MAIN_PATH=cmd/vpn-route-manager
BUILD_DIR=build
DIST_DIR=dist
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "1.0.0")
LDFLAGS=-ldflags "-X main.version=$(VERSION)"

# Go commands
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOMOD=$(GOCMD) mod
GOFMT=$(GOCMD) fmt

# Detect OS and architecture
UNAME_S := $(shell uname -s)
UNAME_M := $(shell uname -m)

ifeq ($(UNAME_S),Darwin)
	OS=darwin
endif

ifeq ($(UNAME_M),x86_64)
	ARCH=amd64
else ifeq ($(UNAME_M),arm64)
	ARCH=arm64
endif

# Default target
.DEFAULT_GOAL := build

# Build for current platform
.PHONY: build
build:
	@echo "🔨 Building $(BINARY_NAME) for $(OS)/$(ARCH)..."
	@mkdir -p $(BUILD_DIR)
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./$(MAIN_PATH)
	@codesign --force --sign - $(BUILD_DIR)/$(BINARY_NAME) 2>/dev/null || true
	@echo "✅ Build complete: $(BUILD_DIR)/$(BINARY_NAME)"

# Build for all Mac architectures
.PHONY: build-all
build-all: build-amd64 build-arm64

.PHONY: build-amd64
build-amd64:
	@echo "🔨 Building for darwin/amd64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./$(MAIN_PATH)
	@codesign --force --sign - $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 2>/dev/null || true

.PHONY: build-arm64
build-arm64:
	@echo "🔨 Building for darwin/arm64..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 $(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./$(MAIN_PATH)
	@codesign --force --sign - $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 2>/dev/null || true

# Run the application
.PHONY: run
run: build
	./$(BUILD_DIR)/$(BINARY_NAME)

# Install to /usr/local/bin
.PHONY: install
install: build
	@echo "📦 Installing $(BINARY_NAME) to /usr/local/bin..."
	@sudo cp $(BUILD_DIR)/$(BINARY_NAME) /usr/local/bin/
	@sudo chmod 755 /usr/local/bin/$(BINARY_NAME)
	@echo "✅ Installed to /usr/local/bin/$(BINARY_NAME)"

# Uninstall from /usr/local/bin
.PHONY: uninstall
uninstall:
	@echo "🗑️  Uninstalling $(BINARY_NAME)..."
	@sudo rm -f /usr/local/bin/$(BINARY_NAME)
	@echo "✅ Uninstalled"

# Clean build artifacts
.PHONY: clean
clean:
	@echo "🧹 Cleaning..."
	@rm -rf $(BUILD_DIR) $(DIST_DIR)
	$(GOCLEAN)
	@echo "✅ Clean complete"

# Run tests
.PHONY: test
test:
	@echo "🧪 Running tests..."
	$(GOTEST) -v ./...

# Format code
.PHONY: fmt
fmt:
	@echo "🎨 Formatting code..."
	$(GOFMT) ./...

# Check formatting
.PHONY: fmt-check
fmt-check:
	@echo "🔍 Checking code formatting..."
	@if [ -n "$$(gofmt -l .)" ]; then \
		echo "❌ Code is not formatted. Run 'make fmt'"; \
		exit 1; \
	else \
		echo "✅ Code is properly formatted"; \
	fi

# Download dependencies
.PHONY: deps
deps:
	@echo "📥 Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# Update dependencies
.PHONY: deps-update
deps-update:
	@echo "🔄 Updating dependencies..."
	$(GOGET) -u ./...
	$(GOMOD) tidy

# Lint code (requires golangci-lint)
.PHONY: lint
lint:
	@echo "🔍 Linting code..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "⚠️  golangci-lint not installed. Install with: brew install golangci-lint"; \
	fi

# Create distribution package
.PHONY: package
package: build
	@echo "📦 Creating distribution package..."
	@rm -rf vpn-route-manager-package
	@mkdir -p vpn-route-manager-package
	@cp $(BUILD_DIR)/$(BINARY_NAME) vpn-route-manager-package/
	@cp installer.sh vpn-route-manager-package/
	@cp uninstaller.sh vpn-route-manager-package/
	@cp README.md vpn-route-manager-package/
	@chmod +x vpn-route-manager-package/*.sh
	@chmod +x vpn-route-manager-package/$(BINARY_NAME)
	@echo "✅ Package created in vpn-route-manager-package/"
	@echo ""
	@echo "📋 Package contents:"
	@ls -la vpn-route-manager-package/
	@echo ""
	@echo "🚀 To create zip: make dist"

# Create distribution zip
.PHONY: dist
dist: package
	@echo "📦 Creating distribution zip..."
	@zip -r vpn-route-manager-$(VERSION).zip vpn-route-manager-package
	@echo "✅ Created: vpn-route-manager-$(VERSION).zip"

# Development mode - build and run with debug
.PHONY: dev
dev:
	@echo "🚀 Running in development mode..."
	$(GOBUILD) $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./$(MAIN_PATH)
	./$(BUILD_DIR)/$(BINARY_NAME) debug

# Check everything before commit
.PHONY: check
check: fmt-check lint test
	@echo "✅ All checks passed!"

# Show help
.PHONY: help
help:
	@echo "VPN Route Manager - Available targets:"
	@echo ""
	@echo "  make build        - Build for current platform"
	@echo "  make build-all    - Build for all Mac architectures"
	@echo "  make run          - Build and run"
	@echo "  make install      - Install to /usr/local/bin"
	@echo "  make uninstall    - Uninstall from /usr/local/bin"
	@echo "  make clean        - Clean build artifacts"
	@echo "  make test         - Run tests"
	@echo "  make fmt          - Format code"
	@echo "  make lint         - Lint code"
	@echo "  make deps         - Download dependencies"
	@echo "  make package      - Create distribution package"
	@echo "  make dist         - Create distribution zip file"
	@echo "  make dev          - Run in development mode"
	@echo "  make check        - Run all checks"
	@echo ""
	@echo "Variables:"
	@echo "  VERSION=$(VERSION)"
	@echo "  OS=$(OS)"
	@echo "  ARCH=$(ARCH)"