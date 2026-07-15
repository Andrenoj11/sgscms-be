.PHONY: run build test fmt vet tidy db-up db-down db-logs migrate-up migrate-down seed

run:
	go run ./cmd/api

build:
	go build -o bin/sgscms-api ./cmd/api

test:
	go test ./...

fmt:
	go fmt ./...

vet:
	go vet ./...

tidy:
	go mod tidy

db-up:
	docker compose up -d postgres

db-down:
	docker compose down

db-logs:
	docker compose logs -f postgres

migrate-up:
	go run ./cmd/migrate -direction=up

migrate-down:
	go run ./cmd/migrate -direction=down -steps=1

seed:
	go run ./cmd/seed


.PHONY: swagger

swagger:
	swag init -g cmd/api/main.go -o docs --parseDependency --parseInternal