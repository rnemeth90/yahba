COMMAND_NAME      := yahba
VERSION           ?= dev
PACKAGE_PATH      := ./main.go
BUILD_DIR         := build/bin
LDFLAGS           := -ldflags="-X main.version=$(VERSION)"

TARGETS := \
    linux-386 \
    linux-amd64 \
    linux-arm \
    linux-arm64 \
    darwin-amd64 \
    windows-386 \
    windows-amd64

OBJECTS := $(foreach target,$(TARGETS),$(BUILD_DIR)/$(COMMAND_NAME)-$(target)$(if $(findstring windows,$(target)),.exe,))

# Docker settings
DOCKER_IMAGE      := ryannemeth/yahba
DOCKER_TAG_LATEST := $(DOCKER_IMAGE):latest
DOCKER_TAG_VER    := $(DOCKER_IMAGE):$(VERSION)

.DEFAULT_GOAL := help

# -------- BUILD & RELEASE --------

.PHONY: release
release: check-env fmt create-build-dir $(OBJECTS)
	@echo "Built all release targets to $(BUILD_DIR)"

$(BUILD_DIR)/$(COMMAND_NAME)-%:
	@echo "🔧 Building for $*..."
	GOOS=$(firstword $(subst -, ,$*)) GOARCH=$(lastword $(subst -, ,$*)) \
		go build $(LDFLAGS) -o $@ $(PACKAGE_PATH)

build:
	@echo "Building for host OS..."
	go build $(LDFLAGS) -o $(BUILD_DIR)/$(COMMAND_NAME) $(PACKAGE_PATH)

# -------- DOCKER --------

.PHONY: docker
docker: check-env
	@echo "Building Docker image..."
	docker build -t $(DOCKER_TAG_LATEST) -t $(DOCKER_TAG_VER) .

.PHONY: docker-push
docker-push: docker
	@echo "Pushing Docker image..."
	docker push $(DOCKER_TAG_LATEST)
	docker push $(DOCKER_TAG_VER)

.PHONY: docker-run
docker-run:
	@echo "Running Docker container..."
	docker run --rm $(DOCKER_TAG_LATEST) --help

.PHONY: docker-clean
docker-clean:
	@echo "Cleaning up Docker images..."
	docker rmi -f $(DOCKER_TAG_LATEST) $(DOCKER_TAG_VER) || true

# -------- DEV UTILS --------

.PHONY: fmt
fmt:
	@echo "Formatting Go files..."
	go fmt ./...

.PHONY: test
test:
	@echo "Running tests..."
	go test ./... -v

.PHONY: clean
clean:
	@echo "Removing build artifacts..."
	rm -rf $(BUILD_DIR)

.PHONY: create-build-dir
create-build-dir:
	@mkdir -p $(BUILD_DIR)

.PHONY: check-env
check-env:
ifndef VERSION
	$(error VERSION is undefined. Use VERSION=x.y.z)
endif

.PHONY: help
help:
	@echo ""
	@echo "Usage:"
	@echo "  make release VERSION=1.2.3   Build binaries for all platforms"
	@echo "  make docker                  Build Docker image with latest and version tags"
	@echo "  make docker-push             Push Docker image to registry"
	@echo "  make docker-run              Run image interactively (prints help)"
	@echo "  make clean                   Clean up binaries"
	@echo "  make build                   Build binaries"
	@echo "  make docker-clean            Remove local Docker images"
	@echo "  make test                    Run unit tests"
	@echo "  make fmt                     Format Go source code"
	@echo ""
