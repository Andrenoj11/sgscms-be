APP_NAME := sgscms-api
MAIN_PACKAGE := ./cmd/api
BINARY_DIRECTORY := ./bin
BINARY_PATH := $(BINARY_DIRECTORY)/$(APP_NAME)

MIGRATION_DIRECTORY := ./migrations

DATABASE_URL := postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSL_MODE)

COMPOSE := docker compose

.DEFAULT_GOAL := help

.PHONY: help
help:
	@echo ""
	@echo "SGS CMS Backend"
	@echo "=============================="
	@echo "make run              Run API locally"
	@echo "make build            Build API binary"
	@echo "make clean            Remove build output"
	@echo "make fmt              Format Go code"
	@echo "make vet              Run go vet"
	@echo "make test             Run tests and compile packages"
	@echo "make test-cover       Generate test coverage"
	@echo "make check            Run fmt, vet, test, and build"
	@echo "make tidy             Run go mod tidy"
	@echo "make swagger          Generate Swagger documentation"
	@echo "make migrate-up       Run all pending migrations"
	@echo "make migrate-down     Roll back one migration"
	@echo "make migrate-force    Force migration version"
	@echo "make seed             Seed admin user"
	@echo "make docker-build     Build Docker image"
	@echo "make docker-up        Start Docker services"
	@echo "make docker-down      Stop Docker services"
	@echo "make docker-reset     Delete containers and volumes"
	@echo "make docker-logs      Follow API logs"
	@echo "make docker-ps        Show Docker service status"
	@echo "make docker-restart   Restart API container"
	@echo ""

.PHONY: run
run:
	go run $(MAIN_PACKAGE)

.PHONY: build
build:
	@mkdir -p $(BINARY_DIRECTORY)
	CGO_ENABLED=0 go build \
		-trimpath \
		-ldflags="-s -w" \
		-o $(BINARY_PATH) \
		$(MAIN_PACKAGE)

.PHONY: clean
clean:
	rm -rf $(BINARY_DIRECTORY)
	rm -f coverage.out

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test:
	go test ./...

.PHONY: test-cover
test-cover:
	go test \
		-coverprofile=coverage.out \
		./...
	go tool cover \
		-func=coverage.out

.PHONY: test-cover-html
test-cover-html:
	go test \
		-coverprofile=coverage.out \
		./...
	go tool cover \
		-html=coverage.out

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: check
check: fmt tidy vet test build
	@echo "All checks passed"

.PHONY: swagger
swagger:
	go run github.com/swaggo/swag/cmd/swag@v1.8.12 init \
		-g cmd/api/main.go \
		-o docs \
		--parseDependency \
		--parseInternal

.PHONY: migrate-up
migrate-up:
	migrate \
		-path $(MIGRATION_DIRECTORY) \
		-database "$(DATABASE_URL)" \
		up

.PHONY: migrate-down
migrate-down:
	migrate \
		-path $(MIGRATION_DIRECTORY) \
		-database "$(DATABASE_URL)" \
		down 1

.PHONY: migrate-version
migrate-version:
	migrate \
		-path $(MIGRATION_DIRECTORY) \
		-database "$(DATABASE_URL)" \
		version

.PHONY: migrate-force
migrate-force:
	@if [ -z "$(VERSION)" ]; then \
		echo "VERSION is required"; \
		echo "Example: make migrate-force VERSION=6"; \
		exit 1; \
	fi

	migrate \
		-path $(MIGRATION_DIRECTORY) \
		-database "$(DATABASE_URL)" \
		force $(VERSION)

.PHONY: migration-create
migration-create:
	@if [ -z "$(NAME)" ]; then \
		echo "NAME is required"; \
		echo "Example: make migration-create NAME=create_contacts"; \
		exit 1; \
	fi

	migrate create \
		-ext sql \
		-dir $(MIGRATION_DIRECTORY) \
		-seq \
		$(NAME)

.PHONY: seed
seed:
	go run ./cmd/seed

.PHONY: docker-build
docker-build:
	$(COMPOSE) build --no-cache api

.PHONY: docker-up
docker-up:
	$(COMPOSE) up \
		-d \
		--build

.PHONY: docker-down
docker-down:
	$(COMPOSE) down \
		--remove-orphans

.PHONY: docker-reset
docker-reset:
	$(COMPOSE) down \
		--volumes \
		--remove-orphans

.PHONY: docker-logs
docker-logs:
	$(COMPOSE) logs \
		-f \
		--tail=200 \
		api

.PHONY: docker-logs-all
docker-logs-all:
	$(COMPOSE) logs \
		-f \
		--tail=200

.PHONY: docker-ps
docker-ps:
	$(COMPOSE) ps

.PHONY: docker-restart
docker-restart:
	$(COMPOSE) restart api

.PHONY: docker-shell
docker-shell:
	$(COMPOSE) exec api /bin/sh

.PHONY: docker-db
docker-db:
	$(COMPOSE) exec postgres \
		psql \
		-U $(DB_USER) \
		-d $(DB_NAME)

.PHONY: docker-migrate-up
docker-migrate-up:
	$(COMPOSE) run \
		--rm \
		migrate \
		-path /migrations \
		-database "postgres://$(DB_USER):$(DB_PASSWORD)@postgres:5432/$(DB_NAME)?sslmode=$(DB_SSL_MODE)" \
		up

.PHONY: docker-migrate-down
docker-migrate-down:
	$(COMPOSE) run \
		--rm \
		migrate \
		-path /migrations \
		-database "postgres://$(DB_USER):$(DB_PASSWORD)@postgres:5432/$(DB_NAME)?sslmode=$(DB_SSL_MODE)" \
		down 1

.PHONY: docker-seed
docker-seed:
	$(COMPOSE) run \
		--rm \
		--entrypoint /app/sgscms-seed \
		api
