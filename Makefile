.DEFAULT_GOAL := run

.PHONY: fmt vet build all

fmt: 
	go fmt ./...

vet: fmt
	go vet ./...

test: vet
	go test -v -race ./...

build: vet
	go build

run: vet
	go run ./main.go

prod: test
	go build

dcu: 
	docker compose up

dcd:
	docker compose down
