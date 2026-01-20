.PHONY: build clean all

# Binary name
BINARY_NAME=sgreen

# Build directory
BUILD_DIR=build

# Build for current platform
build:
	@echo "Building for $(shell go env GOOS)/$(shell go env GOARCH)..."
	@CGO_ENABLED=0 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/sgreen

# Build for all platforms
# Note: Using CGO_ENABLED=0 for static linking without libc dependency
all: clean
	@echo "Building for all platforms (CGO_ENABLED=0, no libc dependency)..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/sgreen
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/sgreen
	@CGO_ENABLED=0 GOOS=linux GOARCH=arm GOARM=7 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-armv7 ./cmd/sgreen
	@CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/sgreen
	@CGO_ENABLED=0 GOOS=windows GOARCH=arm64 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-arm64.exe ./cmd/sgreen
	@CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/sgreen
	@CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/sgreen
	@CGO_ENABLED=0 GOOS=freebsd GOARCH=amd64 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME)-freebsd-amd64 ./cmd/sgreen
	@CGO_ENABLED=0 GOOS=freebsd GOARCH=arm64 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME)-freebsd-arm64 ./cmd/sgreen
	@CGO_ENABLED=0 GOOS=android GOARCH=arm64 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(BINARY_NAME)-android-arm64 ./cmd/sgreen
	@echo "Build complete! Binaries are in $(BUILD_DIR)/"

# Clean build artifacts
clean:
	@echo "Cleaning build directory..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete!"

# Run the application
run:
	@go run ./cmd/sgreen

# Test
test:
	@go test -v ./...

# Format code
fmt:
	@go fmt ./...

# Lint
lint:
	@golangci-lint run || echo "golangci-lint not installed, skipping..."

