.DEFAULT_GOAL := help

INSTALL_DIR ?= /usr/local/bin
BIN_NAME ?= dbctl
CONFIG ?= config.yaml
ENV ?=

.PHONY: help build install uninstall run dry-run clean test

help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2}'

build: ## Build dbctl binary into bin/
	@mkdir -p bin
	@go build -o bin/$(BIN_NAME) main.go
	@echo "Built bin/$(BIN_NAME)"

install: build ## Install dbctl to INSTALL_DIR (default: /usr/local/bin, or pass INSTALL_DIR=~/.local/bin)
	@mkdir -p $(INSTALL_DIR)
	@install -m 755 bin/$(BIN_NAME) $(INSTALL_DIR)/$(BIN_NAME)
	@echo "Installed $(BIN_NAME) to $(INSTALL_DIR)/$(BIN_NAME)"

uninstall: ## Remove dbctl from INSTALL_DIR
	@rm -f $(INSTALL_DIR)/$(BIN_NAME)
	@echo "Removed $(INSTALL_DIR)/$(BIN_NAME)"

run: ## Run dbctl against config.yaml (optional: make run CONFIG=my-config.yaml ENV="-env .env")
	@if [ -x "$(INSTALL_DIR)/$(BIN_NAME)" ]; then \
		"$(INSTALL_DIR)/$(BIN_NAME)" -config $(CONFIG) $(ENV); \
	elif [ -x "./bin/$(BIN_NAME)" ]; then \
		"./bin/$(BIN_NAME)" -config $(CONFIG) $(ENV); \
	else \
		go run main.go -config $(CONFIG) $(ENV); \
	fi

dry-run: ## Simulate execution without modifying databases
	@if [ -x "$(INSTALL_DIR)/$(BIN_NAME)" ]; then \
		"$(INSTALL_DIR)/$(BIN_NAME)" -dry-run -config $(CONFIG) $(ENV); \
	elif [ -x "./bin/$(BIN_NAME)" ]; then \
		"./bin/$(BIN_NAME)" -dry-run -config $(CONFIG) $(ENV); \
	else \
		go run main.go -dry-run -config $(CONFIG) $(ENV); \
	fi

test: ## Run test suite
	@go test -v ./...

clean: ## Remove build artifacts
	@rm -rf bin
	@echo "Cleaned bin/"
