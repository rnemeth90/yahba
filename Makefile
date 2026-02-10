BINARY_NAME = yahba
MODULE = github.com/rnemeth90/yahba
BUILD_DIR = build/bin
LDFLAGS = -ldflags="-X main.version=$(VERSION)"

TARGETS = \
	linux-386 \
	linux-amd64 \
	linux-arm \
	linux-arm64 \
	darwin-amd64 \
	darwin-arm64 \
	windows-386 \
	windows-amd64

.PHONY: build test format clean release tag help check-version

build: ## Build for the current platform
	go build -o $(BINARY_NAME) .

test: ## Run all tests
	go test ./...

format: ## Format all Go source files
	gofmt -w -s .

clean: ## Remove build artifacts
	rm -rf build $(BINARY_NAME)

release: check-version format $(BUILD_DIR) ## Build release binaries for all platforms (requires VERSION)
	@for target in $(TARGETS); do \
		os=$$(echo $$target | cut -d'-' -f1); \
		arch=$$(echo $$target | cut -d'-' -f2); \
		output=$(BUILD_DIR)/$(BINARY_NAME)-$$target-v$(VERSION); \
		if [ "$$os" = "windows" ]; then output=$$output.exe; fi; \
		echo "Building $$output..."; \
		GOOS=$$os GOARCH=$$arch go build $(LDFLAGS) -o $$output .; \
	done

tag: check-version ## Create and push a version tag (requires VERSION)
	@if git rev-parse "v$(VERSION)" >/dev/null 2>&1; then \
		echo "Error: tag v$(VERSION) already exists"; \
		exit 1; \
	fi
	git tag -a "v$(VERSION)" -m "Release v$(VERSION)"
	git push origin "v$(VERSION)"

$(BUILD_DIR):
	mkdir -p $(BUILD_DIR)

check-version:
ifndef VERSION
	$(error VERSION is undefined. Usage: make release VERSION=1.0.0)
endif

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
