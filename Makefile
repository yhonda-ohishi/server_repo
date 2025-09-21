.PHONY: help
help: ## Show this help message
	@echo "Usage: make [target]"
	@echo ""
	@echo "Available targets:"
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

.PHONY: install-tools
install-tools: ## Install required tools
	go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
	go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
	go install github.com/bufbuild/buf/cmd/buf@latest

.PHONY: generate
generate: ## Generate code from proto files
	buf generate

.PHONY: build
build: ## Build the server binary
	go build -o bin/server cmd/server/main.go

.PHONY: run
run: ## Run the server
	go run cmd/server/main.go

.PHONY: test
test: ## Run unit tests
	go test -v ./...

.PHONY: test-integration
test-integration: ## Run integration tests
	go test -v ./tests/integration/...

.PHONY: test-coverage
test-coverage: ## Run tests with coverage
	go test -v -cover -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

.PHONY: clean
clean: ## Clean generated files and binaries
	rm -rf bin/ coverage.* swagger/*.json proto/*.pb.go proto/*.pb.gw.go

.PHONY: lint
lint: ## Run linters
	buf lint

.PHONY: docker-build
docker-build: ## Build Docker image
	docker build -t etc-meisai-gateway .

.PHONY: docker-run
docker-run: ## Run Docker container
	docker run -p 8080:8080 -p 9090:9090 etc-meisai-gateway