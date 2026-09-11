.DEFAULT_GOAL := help

INSTALL_DIR ?= /usr/local/bin
BIN_NAME ?= dbctl
CONFIG ?= config.yaml
ENV ?=
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_DATE ?= $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS := -ldflags "-X github.com/adnanex/dbctl/cmd.Version=$(VERSION) -X github.com/adnanex/dbctl/cmd.Commit=$(COMMIT) -X github.com/adnanex/dbctl/cmd.BuildDate=$(BUILD_DATE)"

.PHONY: help build install uninstall run dry-run clean test

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build dbctl binary into bin/
	@mkdir -p bin
	@go build $(LDFLAGS) -o bin/$(BIN_NAME) main.go
	@echo "Built bin/$(BIN_NAME) ($(VERSION))"

install: build ## Install dbctl to INSTALL_DIR (default: /usr/local/bin, or pass INSTALL_DIR=~/.local/bin)
	@mkdir -p $(INSTALL_DIR)
	@install -m 755 bin/$(BIN_NAME) $(INSTALL_DIR)/$(BIN_NAME)
	@echo "Installed $(BIN_NAME) to $(INSTALL_DIR)/$(BIN_NAME)"

uninstall: ## Remove dbctl from INSTALL_DIR
	@rm -f $(INSTALL_DIR)/$(BIN_NAME)
	@echo "Removed $(INSTALL_DIR)/$(BIN_NAME)"

run: ## Run dbctl provision against config.yaml (optional: make run CONFIG=my-config.yaml ENV="-e .env")
	@if [ -x "$(INSTALL_DIR)/$(BIN_NAME)" ]; then \
		"$(INSTALL_DIR)/$(BIN_NAME)" provision -c $(CONFIG) $(ENV); \
	elif [ -x "./bin/$(BIN_NAME)" ]; then \
		"./bin/$(BIN_NAME)" provision -c $(CONFIG) $(ENV); \
	else \
		go run $(LDFLAGS) main.go provision -c $(CONFIG) $(ENV); \
	fi

dry-run: ## Simulate execution without modifying databases
	@if [ -x "$(INSTALL_DIR)/$(BIN_NAME)" ]; then \
		"$(INSTALL_DIR)/$(BIN_NAME)" provision --dry-run -c $(CONFIG) $(ENV); \
	elif [ -x "./bin/$(BIN_NAME)" ]; then \
		"./bin/$(BIN_NAME)" provision --dry-run -c $(CONFIG) $(ENV); \
	else \
		go run $(LDFLAGS) main.go provision --dry-run -c $(CONFIG) $(ENV); \
	fi

test: ## Run test suite
	@go test -v ./...

clean: ## Remove build artifacts
	@rm -rf bin
	@echo "Cleaned bin/"
