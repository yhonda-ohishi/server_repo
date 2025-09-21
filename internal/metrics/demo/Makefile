BINARY_NAME=metrics-demo

.PHONY: build run clean test

build:
	go build -o $(BINARY_NAME) .

run: build
	./$(BINARY_NAME)

run-dev:
	go run .

clean:
	rm -f $(BINARY_NAME)

test:
	go test ../...

# Test endpoints after starting the server
test-endpoints:
	@echo "Testing endpoints..."
	@echo "Health check:"
	@curl -s http://localhost:8080/health | jq .
	@echo ""
	@echo "Users list:"
	@curl -s http://localhost:8080/api/users | jq .
	@echo ""
	@echo "User by ID:"
	@curl -s http://localhost:8080/api/users/123 | jq .
	@echo ""
	@echo "Custom metric endpoint:"
	@curl -s http://localhost:8080/api/custom-metric | jq .
	@echo ""
	@echo "Metrics (first 20 lines):"
	@curl -s http://localhost:8080/metrics | head -20

help:
	@echo "Available targets:"
	@echo "  build         - Build the demo binary"
	@echo "  run           - Build and run the demo"
	@echo "  run-dev       - Run with go run (development)"
	@echo "  clean         - Remove binary"
	@echo "  test          - Run tests"
	@echo "  test-endpoints- Test the demo endpoints (server must be running)"
	@echo "  help          - Show this help"