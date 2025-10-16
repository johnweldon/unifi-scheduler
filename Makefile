.PHONY: help
help: ## Display this help message
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

.PHONY: all
all: help

.PHONY: build
build: clean ## Build snapshot release using goreleaser
	goreleaser release --auto-snapshot --clean

.PHONY: publish
publish: clean ## Publish release using goreleaser
	goreleaser release --clean

.PHONY: clean
clean: ## Clean build artifacts and vendor directory
	go clean .
	-rm -rf vendor dist

.PHONY: vendor
vendor: ## Create vendor directory
	go mod vendor

