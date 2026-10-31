# go-users-api

A small REST API for managing users, written in Go with PostgreSQL.
Built as a study project: every part of it is something I wanted to understand
better — stdlib HTTP routing, working with a connection pool, error handling,
graceful shutdown, testing and containerization.

## Stack

- **Go 1.26+**, standard library `net/http` (no framework)
- **PostgreSQL 18** via [`pgx/v5`](https://github.com/jackc/pgx) + `pgxpool`
- **Docker / Docker Compose** (multi-stage build, healthchecks)
- **Nix flake + direnv** for a reproducible dev shell (optional)
- **Standard library testing**: `httptest`, table-driven tests, `-race`

## Features

- Full CRUD over users: get by id, create, rename, delete
- Go 1.22+ `ServeMux` routing with method + path patterns (`GET /users/{id}`)
- `UserRepository` interface — HTTP layer is tested against a fake repository,
  PostgreSQL implementation is covered by integration tests
- Sentinel errors (`ErrUserNotFound`) with `errors.Is` and `%w` error wrapping,
  so `pgx.ErrNoRows` never leaks into the HTTP layer
- 3-second `context` timeouts on database calls
- Graceful shutdown on `SIGINT`/`SIGTERM` with a 5-second drain window
- Uniform JSON error responses: `{"error": "..."}`
- Multi-stage Docker build (static binary, `CGO_ENABLED=0`, Alpine-based image)
- `docker compose up` starts Postgres with a healthcheck and waits for it
  to become healthy before starting the API

## API

| Method | Path | Description | Success | Errors |
| --- | --- | --- | --- | --- |
| `GET` | `/users/{id}` | Get a user by id | `200` | `400`, `404`, `500` |
| `POST` | `/users` | Create a user (`{"name": "..."}`) | `201` | `400`, `500` |
| `PUT` | `/users/{id}` | Change a user name (`{"name": "..."}`) | `200` | `400`, `404`, `500` |
| `DELETE` | `/users/{id}` | Delete a user | `204` | `400`, `404`, `500` |

### Examples

```sh
# create (name is trimmed, "  Alex  " becomes "Alex")
http POST :8080/users name="  Alex  "
# => {"id":1,"name":"Alex"}

# read
http GET :8080/users/1
# => {"id":1,"name":"Alex"}

# rename
http PUT :8080/users/1 name="Alex Updated"
# => {"id":1,"name":"Alex Updated"}

# delete
http DELETE :8080/users/1
# => 204 No Content

# errors are always JSON
http GET :8080/users/abc
# => {"error":"Invalid user ID"}  (400)
```

## Project structure

```text
.
├── main.go          # wiring: db pool, routes, server, graceful shutdown
├── server.go        # HTTP handlers, request validation, JSON responses
├── users.go         # UserRepository interface + Postgres implementation
├── db.go            # pgxpool creation from DATABASE_URL
├── init.sql         # schema, mounted into the postgres container
├── server_test.go   # handler tests with a fake repository (httptest)
├── users_test.go    # integration tests against real Postgres
├── db_test.go       # connection setup tests
├── Dockerfile       # multi-stage build
├── docker-compose.yml
├── flake.nix        # dev shell: go, gopls, gofumpt, golangci-lint, httpie
└── Makefile
```

## Getting started

**Prerequisites:** Go 1.26+, Docker. Optional: Nix with `direnv` (`use flake`)
for a ready-made dev shell.

```sh
# 1. configure environment (compose reads .env automatically)
cp .env.example .env

# 2. start Postgres (table `users` is created from init.sql automatically)
dcu          # nix alias, or: docker compose up -d db

# 3. run the API on :8080
make run
```

To run everything in containers instead:

```sh
docker compose up --build
```

The schema itself is trivial:

```sql
CREATE TABLE IF NOT EXISTS users (
    id   BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL
);
```

## Tests

```sh
make test             # go vet + go test -race ./...
make test-integration # the same, but runs integration tests too
```

Unit tests cover every handler through the real router with a fake repository:
happy paths, validation (bad id, empty name, malformed JSON) and mapping of
repository errors to status codes.

Integration tests live in `users_test.go`, use a real PostgreSQL and are
skipped unless the connection string is provided:

```sh
docker compose up -d db
TEST_DATABASE_URL=postgres://app:secret@localhost:5432/app make test-integration
```

They create, read, rename and delete a user, and check that operations on a
missing user return `ErrUserNotFound`.

## Make targets

| Target | Description |
| --- | --- |
| `make run` | vet + run |
| `make test` | vet + `go test -race ./...` |
| `make test-integration` | vet + tests including real Postgres |
| `make build` | vet + build binary |
| `make prod` | test + build |
| `make fmt` / `make vet` | format / static checks |

## Design decisions

- **Interface + fake repository.** Handlers depend on `UserRepository`, not on
  PostgreSQL, so HTTP behaviour is testable without a database. The interface is
  asserted at compile time: `_ UserRepository = (*PostgresUsers)(nil)`.
- **Sentinel error instead of `pgx.ErrNoRows`.** The repository translates
  driver-specific errors into `ErrUserNotFound`; handlers map it to `404` with
  `errors.Is`, while everything else is wrapped with `%w` and logged as `500`.
- **Stdlib routing, no framework.** Go 1.22+ `ServeMux` patterns are enough for
  this API and keep the dependency tree small.
- **`RETURNING` instead of extra round-trips.** `UPDATE`/`DELETE` use
  `RETURNING id, name` / `RowsAffected()` to detect a missing user in a single
  query.
- **Graceful shutdown.** `signal.NotifyContext` + `server.Shutdown` with a
  timeout lets in-flight requests finish before the process exits.
- **Connection pool.** `pgxpool` is created once in `main`, pinged at startup
  (fail fast) and closed on exit.
