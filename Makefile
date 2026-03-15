APP := distlang
CMD := ./cmd/distlang
RELEASE_DIR := dist/release
RELEASE_BIN_DIR := $(RELEASE_DIR)/bin
RELEASE_ASSETS := $(RELEASE_DIR)/assets
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64 windows/arm64

.PHONY: help run build build-cross package checksums release-local test fmt tidy clean

help:
	@echo "Targets:"
	@echo "  make run    - run distlang locally"
	@echo "  make build  - build binary into ./bin"
	@echo "  make build-cross - cross-compile release binaries"
	@echo "  make package - package release binaries into tar.gz"
	@echo "  make checksums - generate SHA256 checksums for release assets"
	@echo "  make release-local - run package + checksums"
	@echo "  make test   - run tests"
	@echo "  make fmt    - format Go code"
	@echo "  make tidy   - tidy go modules"
	@echo "  make clean  - remove build artifacts"

run:
	go run $(CMD)

build:
	mkdir -p bin
	go build -o bin/$(APP) $(CMD)

build-cross:
	mkdir -p $(RELEASE_BIN_DIR)
	@set -e; \
	for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		out="$(RELEASE_BIN_DIR)/$(APP)_$${os}_$${arch}$$ext"; \
		echo "Building $$out"; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -o "$$out" $(CMD); \
	done

package: build-cross
	mkdir -p $(RELEASE_ASSETS)
	@set -e; \
	for platform in $(PLATFORMS); do \
		os=$${platform%/*}; \
		arch=$${platform#*/}; \
		ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		archive="$(RELEASE_ASSETS)/$(APP)_$${os}_$${arch}.tar.gz"; \
		echo "Packaging $$archive"; \
		tar -C "$(RELEASE_BIN_DIR)" -czf "$$archive" "$(APP)_$${os}_$${arch}$$ext"; \
	done

checksums: package
	mkdir -p $(RELEASE_ASSETS)
	@if command -v sha256sum >/dev/null 2>&1; then \
		sha256sum $(RELEASE_ASSETS)/*.tar.gz > $(RELEASE_ASSETS)/checksums.txt; \
	else \
		shasum -a 256 $(RELEASE_ASSETS)/*.tar.gz > $(RELEASE_ASSETS)/checksums.txt; \
	fi

release-local: checksums

test:
	go test ./...

fmt:
	go fmt ./...

tidy:
	go mod tidy

clean:
	rm -rf bin $(RELEASE_DIR)
