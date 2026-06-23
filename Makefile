APP := weather-cli
CMD := ./cmd/weather-cli
WEB_CMD := ./cmd/weather-web
WASM_CMD := ./cmd/weather-web-wasm
BIN_DIR := bin
BIN := $(BIN_DIR)/$(APP)
WEB_BIN := $(BIN_DIR)/weather-web
WASM_OUT := web/static/weather-web.wasm
WASM_EXEC := web/static/wasm_exec.js

GO ?= go
GOLANGCI_LINT ?= golangci-lint
GOCACHE ?= /tmp/weather-cli-gocache
GOMODCACHE ?= /tmp/weather-cli-gomod
GOLANGCI_LINT_CACHE ?= /tmp/weather-cli-golangci-lint-cache
CGO_ENABLED ?= 0
GOFLAGS ?= -buildvcs=false

ifneq (,$(wildcard .env))
include .env
export
endif

.PHONY: help run web icon-test build build-web build-wasm test lint fmt tidy clean check docker-build docker-run

help:
	@printf "Weather CLI targets:\n"
	@printf "  make run      Run the TUI using .env values\n"
	@printf "  make web      Run the browser terminal bridge\n"
	@printf "  make icon-test Open the ASCII + Unicode animated icon gallery page\n"
	@printf "  make build    Build $(BIN)\n"
	@printf "  make build-web Build the web server and WASM helper\n"
	@printf "  make test     Run the Go test suite\n"
	@printf "  make lint     Run golangci-lint\n"
	@printf "  make fmt      Format Go sources\n"
	@printf "  make tidy     Tidy Go modules\n"
	@printf "  make check    Run fmt, tidy, test, and lint\n"
	@printf "  make clean    Remove build output\n"

run:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) CGO_ENABLED=$(CGO_ENABLED) $(GO) run $(GOFLAGS) $(CMD)

web: build build-wasm
	WEATHER_CLI_BIN=$(CURDIR)/$(BIN) WEATHER_CLI_STATIC_ROOT=$(CURDIR)/web/static GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) CGO_ENABLED=$(CGO_ENABLED) $(GO) run $(GOFLAGS) $(WEB_CMD)

icon-test:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) CGO_ENABLED=$(CGO_ENABLED) $(GO) run $(GOFLAGS) $(CMD) --page icons

build:
	mkdir -p $(BIN_DIR)
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GOFLAGS) -o $(BIN) $(CMD)

build-web: build build-wasm
	mkdir -p $(BIN_DIR)
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) CGO_ENABLED=$(CGO_ENABLED) $(GO) build $(GOFLAGS) -o $(WEB_BIN) $(WEB_CMD)

build-wasm:
	mkdir -p web/static
	cp "$$($(GO) env GOROOT)/lib/wasm/wasm_exec.js" $(WASM_EXEC)
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) GOOS=js GOARCH=wasm $(GO) build $(GOFLAGS) -ldflags "-X main.version=$$(git rev-parse --short HEAD 2>/dev/null || printf dev)" -o $(WASM_OUT) $(WASM_CMD)

test:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) CGO_ENABLED=$(CGO_ENABLED) $(GO) test $(GOFLAGS) ./...

lint:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) GOLANGCI_LINT_CACHE=$(GOLANGCI_LINT_CACHE) CGO_ENABLED=$(CGO_ENABLED) $(GOLANGCI_LINT) run ./...

fmt:
	gofmt -w cmd internal

tidy:
	GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) CGO_ENABLED=$(CGO_ENABLED) $(GO) mod tidy

check: fmt tidy test lint

docker-build:
	docker build -t weather-cli:latest .

docker-run:
	docker run --rm -p 8080:8080 weather-cli:latest

clean:
	rm -rf $(BIN_DIR) $(WASM_OUT) $(WASM_EXEC)
