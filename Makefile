ENV ?= development

.PHONY: run build test lint migrate-up migrate-down migrate-create swagger mocks clean run-worker build-worker docker-build docker-build-worker

run:
	@bash -c 'set -a; [ -f .env.$(ENV) ] && source .env.$(ENV); set +a; go run ./cmd/api'

build:
	go build -o bin/api ./cmd/api

run-worker:
	@bash -c 'set -a; [ -f .env.$(ENV) ] && source .env.$(ENV); set +a; go run ./cmd/worker'

build-worker:
	go build -o bin/worker ./cmd/worker

docker-build:
	docker build -t task-management:latest .

docker-build-worker:
	docker build -f Dockerfile.worker -t task-management-worker:latest .

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
