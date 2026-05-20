ENV ?= development

.PHONY: run build test lint migrate-up migrate-down migrate-create swagger mocks clean

run:
	@bash -c 'set -a; [ -f .env.$(ENV) ] && source .env.$(ENV); set +a; go run ./cmd/api'

build:
	go build -o bin/api ./cmd/api

test:
	go test ./... -count=1 -race

lint:
	golangci-lint run ./...

migrate-up:
	@bash -c 'set -a; [ -f .env.$(ENV) ] && source .env.$(ENV); set +a; migrate -path migrations -database "$${DATABASE_URL}" up'

migrate-down:
	@bash -c 'set -a; [ -f .env.$(ENV) ] && source .env.$(ENV); set +a; migrate -path migrations -database "$${DATABASE_URL}" down 1'

migrate-create:
	migrate create -ext sql -dir migrations -seq $(name)

swagger:
	swag init -g cmd/api/main.go -o docs

mocks:
	go generate ./...

clean:
	rm -rf bin/
