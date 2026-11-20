.DEFAULT_GOAL := run

.PHONY: fmt vet build test test-integration run prod

fmt:
	go fmt ./...

vet: fmt
	go vet ./...

test: vet
	go test -v -race ./...

# Requires a running postgres
test-integration: vet
	go test -v -race -run Integration ./...

build: vet
	go build -o go-users-api ./cmd/api

run: vet
	go run ./cmd/api

prod: test
	go build -o go-users-api ./cmd/api
