.PHONY: all sqlc generate build clean test test-search help

# Load .env file if it exists
-include .env
export

# Path to vss.json - update this to point to your VSS repo
VSS_JSON ?= vss.json

all: sqlc generate build ## Generate DB and build binary (default)

sqlc: ## Generate Go code from SQL using sqlc
	sqlc generate

generate: sqlc ## Create signals.db with embeddings (requires OPENAI_API_KEY)
	@if [ -z "$$OPENAI_API_KEY" ]; then \
		echo "Error: OPENAI_API_KEY not set. Add it to .env or export it."; \
		exit 1; \
	fi
	go run ./cmd/generate -vss $(VSS_JSON) -db signals.db

build: signals.db ## Build vssss binary with embedded DB
	go build -o vssss ./cmd/vssss

test: ## Run unit tests
	go test ./internal/... -v

test-search: build ## Build and run a test query
	@if [ -z "$$OPENAI_API_KEY" ]; then \
		echo "Error: OPENAI_API_KEY not set."; \
		exit 1; \
	fi
	./vssss -n 5 "engine temperature"

clean: ## Remove build artifacts
	rm -f signals.db vssss

help: ## Show this help
	@echo "VSS Semantic Search (vssss)"
	@echo ""
	@echo "Setup: Create .env with OPENAI_API_KEY=sk-..."
	@echo ""
	@echo "Targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' Makefile | sed 's/:.*## /\t/' | awk -F'\t' '{printf "  %-15s %s\n", $$1, $$2}'
