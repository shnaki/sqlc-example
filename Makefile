.PHONY: dev build test lint fmt sqlc \
        migrate-up migrate-down migrate-create \
        docker-up docker-down \
        run-01 run-02 run-03 run-04 run-05 run-06

DB_URL ?= postgres://sqlcdemo:sqlcdemo@localhost:5437/sqlcdemo?sslmode=disable
MIGRATIONS_DIR := db/migrations

dev:
	@echo "利用可能なサンプル: 01-basics 02-transactions 03-relations 04-dynamic 05-batch 06-advanced"
	@echo "実行方法: make run-01  /  go run ./cmd/01-basics"

build:
	go build ./...

test:
	go test ./... -count=1

lint:
	go tool golangci-lint run ./...

fmt:
	gofmt -w .
	go tool goimports -w .

sqlc:
	go tool sqlc generate

migrate-up:
	go run ./cmd/migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" up

migrate-down:
	go run ./cmd/migrate -path $(MIGRATIONS_DIR) -database "$(DB_URL)" down 1

migrate-create:
	go run ./cmd/migrate create -ext sql -dir $(MIGRATIONS_DIR) -seq

docker-up:
	docker compose up -d

docker-down:
	docker compose down

run-01:
	go run ./cmd/01-basics

run-02:
	go run ./cmd/02-transactions

run-03:
	go run ./cmd/03-relations

run-04:
	go run ./cmd/04-dynamic

run-05:
	go run ./cmd/05-batch

run-06:
	go run ./cmd/06-advanced
