# Makefile for building the slave tool
# Default build target is Windows, with explicit target for Linux

# Variables
MAIN_FILE=cmd/slave/main.go
BUILD_DIR=build/bin
VERSION?=0.0.1
BUILD_TIME?=$(shell date +%FT%T%z)

LDFLAGS=-ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}"

# Create build directory
$(BUILD_DIR):
	@mkdir -p $(BUILD_DIR)

# Default target - build for Windows
build: build-windows linux

# Build for Windows (default)
build-windows: $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -v -o $(BUILD_DIR)/slave.exe $(MAIN_FILE)

# Build for Linux
linux: $(BUILD_DIR)
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -v -o $(BUILD_DIR)/slave $(MAIN_FILE)

# Build for current platform (native build)
native: $(BUILD_DIR)
	go build ${LDFLAGS} -v -o $(BUILD_DIR)/slave $(MAIN_FILE)

# Install dependencies
deps:
	go mod tidy

# Clean build directory
clean:
	-@rm -rf $(BUILD_DIR)/slave* 2>/dev/null || rmdir /s /q $(BUILD_DIR)/slave* 2>nul

# Display help
help:
	@echo "Makefile for building the slave tool"
	@echo ""
	@echo "Targets:"
	@echo "  make          - Build for All platforms"
	@echo "  make build    - Build for Windows"
	@echo "  make linux    - Build for Linux"
	@echo "  make native   - Build for current platform"
	@echo "  make clean    - Remove build directory"
	@echo "  make deps     - Install dependencies"
	@echo "  make help     - Display this help message"

# Default target
.DEFAULT_GOAL := build

# Declare phony targets
.PHONY: build build-windows linux native clean deps help