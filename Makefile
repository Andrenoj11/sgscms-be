.PHONY: run test vet fmt up down
run:
	go run ./cmd/api
test:
	go test ./...
vet:
	go vet ./...
fmt:
	gofmt -w cmd internal migrations
up:
	docker compose up --build
down:
	docker compose down
