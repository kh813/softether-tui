# Build/cross-compile options for softether-tui. Mirrors the targets and
# ldflags .github/workflows/*.yml and .goreleaser.yaml use, so `make cross`
# is a quick local stand-in for a GoReleaser run (see app_specs.md 11.1).

BINARY := softether-tui
DIST   := dist

# Overridable from the environment, e.g. `make build VERSION=v0.2.0`.
VERSION ?= $(shell git describe --tags --dirty --always 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X main.version=$(VERSION) \
	-X main.commit=$(COMMIT) \
	-X main.date=$(DATE)

# GOOS/GOARCH pairs `make cross` builds; keep in sync with .goreleaser.yaml.
PLATFORMS := linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

# Matches the current environment (or an overridden GOOS, e.g.
# `GOOS=windows make build`), so `make build` names the output .exe when
# targeting Windows.
EXT :=
ifeq ($(shell go env GOOS),windows)
EXT := .exe
endif

.DEFAULT_GOAL := build

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | \
		awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-12s\033[0m %s\n", $$1, $$2}'

.PHONY: build
build: ## Build a binary for the current (or GOOS/GOARCH-overridden) platform
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY)$(EXT) .

.PHONY: run
run: build ## Build and run
	./$(BINARY)$(EXT)

.PHONY: install
install: ## Build and install to $GOBIN (or $GOPATH/bin)
	go install -trimpath -ldflags "$(LDFLAGS)" .

.PHONY: fmt
fmt: ## Fail if gofmt would reformat anything
	@out="$$(gofmt -l .)"; \
	if [ -n "$$out" ]; then echo "$$out"; exit 1; fi

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: test
test: ## Run tests
	go test ./...

.PHONY: lint
lint: ## Run golangci-lint (must be installed separately)
	golangci-lint run

.PHONY: check
check: fmt vet test ## Run the same checks as CI (fmt + vet + test)

.PHONY: cross
cross: clean-dist ## Cross-compile release binaries for all PLATFORMS into dist/
	@mkdir -p $(DIST)
	@for platform in $(PLATFORMS); do \
		os=$${platform%/*}; arch=$${platform#*/}; \
		ext=""; [ "$$os" = "windows" ] && ext=".exe"; \
		out=$(DIST)/$(BINARY)_$${os}_$${arch}$$ext; \
		echo "==> $$os/$$arch -> $$out"; \
		GOOS=$$os GOARCH=$$arch CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o $$out . || exit 1; \
	done

.PHONY: checksums
checksums: ## Write dist/checksums.txt for whatever is currently in dist/
	@cd $(DIST) && files=$$(ls) && \
		( command -v sha256sum >/dev/null 2>&1 && sha256sum $$files || shasum -a 256 $$files ) > checksums.txt
	@cat $(DIST)/checksums.txt

.PHONY: clean-dist
clean-dist:
	rm -rf $(DIST)

.PHONY: clean
clean: clean-dist ## Remove build artifacts
	rm -f $(BINARY) $(BINARY).exe
