# go_learn

Small Go learning playground. 

## Structure

- **Root** — current things that i'm working for
- **shitty_cli/** — beginning practice projects: `bank`, `fibonacci`, `notifies`, `orderManagementSystem`, `wordCount`

## Prerequisites

- Go 1.26+
- Docker (for PostgreSQL)
- Optional: Nix (`direnv`) for a ready dev shell with linting tools

## Getting started

```sh
# start PostgreSQL (user: app, password: secret, db: app)
# in-flake alias
dcu          # or: docker compose up -d

# run main.go file
make run

# stop docker container
dcd
```

## Make targets

| Target | Description |
| --- | --- |
| `make run` | vet + run |
| `make test` | vet + `go test -race ./...` |
| `make build` | vet + build binary |
| `make prod` | test + build |
| `make fmt` / `make vet` | format / static checks |
