.PHONY: help docker-up docker-down docker-restart docker-logs deps build run test clean

help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Available targets:'
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-20s %s\n", $$1, $$2}'

docker-up: ## Start Docker services (PostgreSQL and RabbitMQ)
	docker compose up -d
	@echo "Waiting for services to be ready..."
	@sleep 3
	@echo "Services are running!"

docker-down: ## Stop Docker services
	docker compose down

docker-restart: docker-down docker-up ## Restart Docker services

docker-logs: ## Show Docker logs
	docker compose logs -f

deps: ## Download and verify Go dependencies
	go mod download
	go mod tidy

build: deps ## Build the LibreCash binary
	go build -o librecash

run: docker-up ## Run the LibreCash bot (requires Docker services)
	go run librecash.go

test: ## Run all tests
	go test -v ./...

clean: ## Clean build artifacts
	rm -f librecash
	go clean

db-shell: ## Connect to PostgreSQL database
	docker exec -it -e PGPASSWORD=librecash librecash-db-1 psql -h localhost -U librecash -d librecash

rabbit-ui: ## Open RabbitMQ Management UI in browser
	@echo "Opening RabbitMQ Management UI at http://localhost:8080"
	@echo "Login: guest/guest"
	@command -v xdg-open >/dev/null 2>&1 && xdg-open http://localhost:8080 || \
		command -v open >/dev/null 2>&1 && open http://localhost:8080 || \
		echo "Please open http://localhost:8080 in your browser"

status: ## Check status of all services
	@echo "Docker services:"
	@docker compose ps
	@echo ""
	@echo "Database users count:"
	@docker exec -e PGPASSWORD=librecash librecash-db-1 psql -h localhost -U librecash -d librecash -t -c "SELECT COUNT(*) as user_count FROM users;" 2>/dev/null || echo "Database not accessible"
	@echo ""
	@echo "RabbitMQ status:"
	@curl -s -u guest:guest http://localhost:8080/api/overview 2>/dev/null | python3 -c "import sys, json; data = json.load(sys.stdin); print(f\"RabbitMQ version: {data['rabbitmq_version']}\")" 2>/dev/null || echo "RabbitMQ not accessible"

dev: docker-up ## Start development environment (Docker + bot)
	@echo "Starting LibreCash in development mode..."
	go run librecash.go

all: docker-up deps build ## Build everything and start services