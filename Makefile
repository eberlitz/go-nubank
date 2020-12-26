.DEFAULT_GOAL := help

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
.PHONY: help

define print-target
    @printf "Executing target: \033[36m$@\033[0m\n"
endef

deps: ## Install necessary project dependencies
	$(call print-target)
	@-cd tools && go install mvdan.cc/gofumpt/gofumports
	@-cd tools && go install github.com/golangci/golangci-lint/cmd/golangci-lint
.PHONY: deps

build: ## Runs go build to check if it compiles
	$(call print-target)
	@go build -o /dev/null ./...
.PHONY: build

lint: ## Runs linter
	@golangci-lint run ./...
.PHONY: lint

test: ## Runs test cases
	@go test -cover -race ./...
.PHONY: test

fmt: ## Runs gofumports
	$(call print-target)
	@-gofumports -l -w -local github.com/eberlitz/go-nubank . || true
.PHONY: fmt

mod-tidy: ## go mod tidy
	$(call print-target)
	@go mod tidy
	@cd tools && go mod tidy
.PHONY: mod-tidy
