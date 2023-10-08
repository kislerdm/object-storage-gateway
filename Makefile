.DEFAULT_GOAL := help

.PHONY: help
help: ## Prints help message.
	@ grep -h -E '^[a-zA-Z0-9_-].+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[1m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: tests
tests: ## Runs units tests.
	@ go test -tags=unittest ./...

URL := http://localhost:3000/object

.PHONY: e2etest
e2etest: ## Runs e2e tests.
	@./e2e-test/test_tinytextfile.sh

lint: ## Runs golangci linters.
	@ docker run -t --rm -v ${PWD}:/app -w /app golangci/golangci-lint:v1.54.2 golangci-lint run -v ./...
