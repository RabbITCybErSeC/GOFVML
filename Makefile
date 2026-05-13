MAIN_BIN := gofvml
TOOL_BINS := gofvml-convert gofvml-upload
BINS := $(MAIN_BIN) $(TOOL_BINS)
BUILD_DIR ?= bin
DIST_DIR ?= dist
GO ?= go
GOFLAGS ?= -trimpath
CGO_ENABLED ?= 0

LINUX_TARGETS ?= linux/amd64 linux/arm64 linux/arm/v7 linux/386

.PHONY: all build build-tools build-all build-all-tools build-linux-amd64 build-linux-arm64 build-linux-armv7 build-linux-386 test vet clean list-targets

all: build

build:
	@mkdir -p $(BUILD_DIR)
	@echo "building $(BUILD_DIR)/$(MAIN_BIN)"
	@CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$(MAIN_BIN) ./cmd/$(MAIN_BIN)

build-tools:
	@mkdir -p $(BUILD_DIR)
	@for bin in $(BINS); do \
		echo "building $(BUILD_DIR)/$$bin"; \
		CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$$bin ./cmd/$$bin; \
	done

build-all:
	@$(MAKE) build-targets BINS_TO_BUILD="$(MAIN_BIN)"

build-all-tools:
	@$(MAKE) build-targets BINS_TO_BUILD="$(BINS)"

build-linux-amd64:
	@$(MAKE) build-target TARGET=linux/amd64

build-linux-arm64:
	@$(MAKE) build-target TARGET=linux/arm64

build-linux-armv7:
	@$(MAKE) build-target TARGET=linux/arm/v7

build-linux-386:
	@$(MAKE) build-target TARGET=linux/386

.PHONY: build-targets build-target

build-targets:
	@mkdir -p $(DIST_DIR)
	@for target in $(LINUX_TARGETS); do \
		$(MAKE) build-target TARGET=$$target BINS_TO_BUILD="$(BINS_TO_BUILD)"; \
	done

build-target:
	@mkdir -p $(DIST_DIR)
	@target="$(TARGET)"; \
		os=$${target%%/*}; \
		arch_variant=$${target#*/}; \
		arch=$${arch_variant%%/*}; \
		arm=$${arch_variant#*/}; \
		if [ "$$arm" = "$$arch_variant" ]; then arm=""; fi; \
		for bin in $(or $(BINS_TO_BUILD),$(MAIN_BIN)); do \
			out="$(DIST_DIR)/$$bin-$$os-$$arch"; \
			if [ -n "$$arm" ]; then out="$$out-$$arm"; fi; \
			echo "building $$out"; \
			if [ -n "$$arm" ]; then \
				CGO_ENABLED=$(CGO_ENABLED) GOOS=$$os GOARCH=$$arch GOARM=$${arm#v} $(GO) build $(GOFLAGS) -o $$out ./cmd/$$bin; \
			else \
				CGO_ENABLED=$(CGO_ENABLED) GOOS=$$os GOARCH=$$arch $(GO) build $(GOFLAGS) -o $$out ./cmd/$$bin; \
			fi; \
		done

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

clean:
	rm -rf $(BUILD_DIR) $(DIST_DIR)

list-targets:
	@printf '%s\n' $(LINUX_TARGETS)
