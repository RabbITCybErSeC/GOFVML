BINS := gofvml gofvml-convert gofvml-upload
BUILD_DIR ?= bin
DIST_DIR ?= dist
GO ?= go
GOFLAGS ?= -trimpath
CGO_ENABLED ?= 0

LINUX_TARGETS ?= linux/amd64 linux/arm64 linux/arm/v7 linux/386

.PHONY: all build build-all test vet clean list-targets

all: build

build:
	@mkdir -p $(BUILD_DIR)
	@for bin in $(BINS); do \
		echo "building $(BUILD_DIR)/$$bin"; \
		CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GOFLAGS) -o $(BUILD_DIR)/$$bin ./cmd/$$bin; \
	done

build-all:
	@mkdir -p $(DIST_DIR)
	@for target in $(LINUX_TARGETS); do \
		os=$${target%%/*}; \
		arch_variant=$${target#*/}; \
		arch=$${arch_variant%%/*}; \
		arm=$${arch_variant#*/}; \
		if [ "$$arm" = "$$arch_variant" ]; then arm=""; fi; \
		for bin in $(BINS); do \
			out="$(DIST_DIR)/$$bin-$$os-$$arch"; \
			if [ -n "$$arm" ]; then out="$$out-$$arm"; fi; \
			echo "building $$out"; \
			if [ -n "$$arm" ]; then \
				CGO_ENABLED=$(CGO_ENABLED) GOOS=$$os GOARCH=$$arch GOARM=$${arm#v} $(GO) build $(GOFLAGS) -o $$out ./cmd/$$bin; \
			else \
				CGO_ENABLED=$(CGO_ENABLED) GOOS=$$os GOARCH=$$arch $(GO) build $(GOFLAGS) -o $$out ./cmd/$$bin; \
			fi; \
		done; \
	done

test:
	$(GO) test ./...

vet:
	$(GO) vet ./...

clean:
	rm -rf $(BUILD_DIR) $(DIST_DIR)

list-targets:
	@printf '%s\n' $(LINUX_TARGETS)
