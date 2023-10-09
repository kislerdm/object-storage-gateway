.DEFAULT_GOAL := help

.PHONY: help
help: ## Prints help message.
	@ grep -h -E '^[a-zA-Z0-9_-].+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[1m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: tests
tests: ## Runs units tests.
	@ go test -coverprofile=.coverage.out -timeout=2m -tags=unittest -v ./...
	@ go tool cover -func .coverage.out | tail -1 && rm .coverage.out

URL := http://localhost:3000/object

.PHONY: e2etest
e2etests: ## Runs e2e tests.
	@ cd e2e-test && ./e2e-tests.sh

.PHONY: lint
lint: ## Runs golangci linters.
	@ docker run -t --rm -v ${PWD}:/app -w /app golangci/golangci-lint:v1.54.2 golangci-lint run -v ./...
