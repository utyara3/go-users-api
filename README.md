# go-users-api

A small REST API for managing users, written in Go with PostgreSQL.
Built as a study project: every part of it is something I wanted to understand
better — stdlib HTTP routing, working with a connection pool, error handling,
graceful shutdown, testing and containerization.

## Stack

- **Go 1.27+**, standard library `net/http` (no framework)
- **PostgreSQL 18** via [`pgx/v5`](https://github.com/jackc/pgx) + `pgxpool`
- **Docker / Docker Compose** (multi-stage build, healthchecks)
- **Nginx** as reverse proxy / ingress in front of the Go service
- **Nix flake + direnv** for a reproducible dev shell (optional)
- **Standard library testing**: `httptest`, table-driven tests, `-race`

## Features

- Full CRUD over users: get by id, create, rename, delete
- `GET /health` liveness endpoint (used by the Compose healthcheck)
- Go 1.22+ `ServeMux` routing with method + path patterns (`GET /users/{id}`)
- `UserRepository` interface — HTTP layer is tested against a fake repository,
  PostgreSQL implementation is covered by integration tests
- Sentinel errors (`ErrUserNotFound`, `ErrWrongUserID`, `ErrWrongUserName`,
  `ErrInvalidJSON`, `ErrDBNotFound`) with `errors.Is` and `%w` wrapping, so
  `pgx.ErrNoRows` never leaks into the HTTP layer
- One error-mapping point: every handler failure goes through `handleError`,
  which maps each sentinel to its 4xx status and logs everything else as a 5xx
- Structured JSON logs (`slog`), level set via `LOG_LEVEL` (default `info`)
- Graceful shutdown on `SIGINT`/`SIGTERM` with a 5-second drain window
- Uniform JSON responses; errors are always `{"error": "..."}`
- Multi-stage Docker build (static binary, `CGO_ENABLED=0`, Alpine-based image,
  non-root `app` user)
- Ordered startup: Postgres must be healthy before the API starts; the API must
  answer `GET /health` before Nginx starts
- The Go backend has **no host port mapping** in Compose — all external traffic
  enters through Nginx over the internal Docker bridge

## API

| Method | Path | Description | Success | Errors |
| --- | --- | --- | --- | --- |
| `GET` | `/health` | Liveness probe | `200` | — |
| `GET` | `/users/{id}` | Get a user by id | `200` | `400`, `404`, `500` |
| `POST` | `/users` | Create a user (`{"name": "..."}`) | `201` | `400`, `500` |
| `PUT` | `/users/{id}` | Change a user name (`{"name": "..."}`) | `200` | `400`, `404`, `500` |
| `DELETE` | `/users/{id}` | Delete a user | `204` | `400`, `404`, `500` |

### Examples

```sh
# liveness
http GET :8080/health
# => {"status":"ok"}

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
# => {"error":"Wrong user ID"}  (400)
```

## Project structure

```text
.
├── cmd/
│   └── api/                # entrypoint
│       ├── main.go         # wiring: pool, service, routes, graceful shutdown
│       ├── logger.go       # slog JSON logger, LOG_LEVEL parsing
│       └── logger_test.go
├── internal/               # private packages: nothing outside may import them
│   ├── httpapi/            # HTTP layer
│   │   ├── handlers.go     # ServeMux routes, handlers, error mapping
│   │   └── handlers_test.go # tests in package httpapi_test
│   ├── user/               # domain layer, no HTTP or SQL knowledge
│   │   ├── user.go         # User model, ErrUserNotFound
│   │   ├── service.go      # UserService: validation + business rules
│   │   └── repository.go   # UserRepository interface
│   ├── postgres/           # PostgreSQL adapter
│   │   ├── pool.go         # pgxpool creation from DATABASE_URL
│   │   ├── user_repository.go # UserRepository implementation
│   │   ├── pool_test.go    # tests in package postgres_test
│   │   └── users_test.go   # integration tests against real Postgres
├── init.sql                # schema, mounted into the postgres container
├── Dockerfile              # multi-stage build, non-root user
├── .dockerignore
├── .env.example
├── docker-compose.yml
├── nginx/
│   └── nginx.conf          # reverse proxy config (mounted into nginx container)
├── flake.nix               # dev shell: go, gopls, gofumpt, golangci-lint, httpie
└── Makefile
```

Layers: `internal/httpapi` → `internal/user` (`UserService`, validation) →
`UserRepository` interface → `internal/postgres` (PostgreSQL implementation).
Dependencies point inward: the HTTP layer knows the service, the service knows
only the repository interface, and the repository translates SQL errors into
domain errors. HTTP handlers never see the database, the service never sees
HTTP. `cmd/api` is the single place that wires all three together.

## Getting started

**Prerequisites:** Go 1.27+, Docker. Optional: Nix with `direnv` (`use flake`)
for a ready-made dev shell.

```sh
# 1. configure environment (compose reads .env automatically)
cp .env.example .env

# 2. start Postgres (table `users` is created from init.sql automatically)
dcu          # nix alias, or: docker compose up -d db

# 3. run the API on :8080
make run
```

To run everything in containers instead (including Nginx):

```sh
docker compose up --build
```

### Database URL, two contexts

The app only reads `DATABASE_URL` from the environment (see
`internal/postgres/pool.go`), so the value depends on where it runs:

- **On the host** (`make run`, `make test-integration`) — `localhost:5432`,
  as shipped in `.env.example`.
- **Inside Compose** — `db:5432`. The `api` service overrides the variable via
  its `environment:` block, so the `.env` file stays host-friendly.

The schema itself is trivial:

```sql
CREATE TABLE IF NOT EXISTS users (
    id   BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL
);
```

### Environment variables

All of them live in `.env` (see `.env.example`); Compose passes them to the
`api` service via `env_file`.

| Variable | Default | Purpose |
| --- | --- | --- |
| `DATABASE_URL` | — (required, app fails fast without it) | Postgres connection string (`internal/postgres/pool.go`) |
| `TEST_DATABASE_URL` | — | Connection string for integration tests; they skip without it |
| `LOG_LEVEL` | `info` | `slog` level: `debug`, `info`, `warn`, `error` (`cmd/api/logger.go`); invalid values fall back to `info` |

## Tests

```sh
make test             # go vet + go test -race ./...
make test-integration # the same, but runs integration tests too
```

Unit tests cover every handler through the real router with a fake repository:
happy paths, validation (bad id, empty name, malformed JSON) and mapping of
repository errors to status codes. Two more guarantees are pinned by tests:
5xx errors end up in the structured logs with the request context, and
`LOG_LEVEL` controls the logger level.

Unit tests live in **external test packages** — `internal/httpapi` is tested
as `package httpapi_test`, `internal/postgres` as `package postgres_test`.
Such a test imports the package under test by its real path and can touch only
its exported API, i.e. exactly what any other consumer sees. The exception is
`cmd/api/logger_test.go` (`package main`): it covers unexported helpers
(`newLogger`, `logLevelFromEnv`), which an external package cannot reach.

Integration tests live in `internal/postgres/users_test.go`, use a real
PostgreSQL and are skipped unless the connection string is provided:

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

- **Interfaces between layers.** The service depends on a `UserRepository`
  interface declared on the consumer side in `internal/user`, so the HTTP
  layer is tested against a fake repository (through the real service), and
  the service can be unit tested without a database. PostgreSQL conformance
  is asserted at compile time:
  `_ user.UserRepository = (*postgres.PostgresUsers)(nil)`.
  The HTTP layer, in turn, talks to the concrete `UserService` and gets it
  through the `httpapi.NewServer` constructor; `Server` keeps its fields
  unexported, so nothing can rewire it from the outside.
- **Sentinel errors, one mapping point.** The repository translates
  driver-specific errors into `ErrUserNotFound`; validation failures are
  `ErrWrongUserID`/`ErrWrongUserName`, decode failures are `ErrInvalidJSON`.
  Every handler routes errors through `handleError`, which maps each sentinel
  to its 4xx status with `errors.Is` and logs everything else as a structured
  5xx entry with the original error, method and path.
- **Stdlib routing, no framework.** Go 1.22+ `ServeMux` patterns are enough and
  keep the dependency tree small.
- **`RETURNING` instead of extra round-trips.** `UPDATE`/`DELETE` use
  `RETURNING id, name` / `RowsAffected()` to detect a missing user in a single
  query.
- **Graceful shutdown.** `signal.NotifyContext` + `server.Shutdown` with a
  timeout lets in-flight requests finish before the process exits.
- **Connection pool.** `pgxpool` is created once in `main` with an explicit
  `MaxConns`/`MinConns`/lifetime configuration, pinged at startup (fail fast)
  and closed on exit.
- **`/health` liveness endpoint.** A dedicated, DB-free handler gives Compose
  a cheap way to tell that the HTTP layer is actually serving, not just that
  the TCP socket is bound.
- **Nginx facade over the Go service.** The API container has no host port
  mapping in Compose — it's reachable only through Nginx on the internal
  bridge. That gives a single public HTTP entrypoint, a natural place to
  terminate TLS later, and shields the Go process from raw TCP exposure.
  The database port is published to the host for local tooling and integration
  tests.

## Infrastructure & Network Architecture

The Go application is isolated inside the inner Docker network and is not
exposed to the host. All external traffic is routed by **Nginx**.

```
Client ──HTTP:80──> nginx ──bridge──> api:8080 ──TCP──> db:5432
```

- **Reverse proxy:** Nginx hides the internal ports and architecture behind a
  single public entrypoint (`:80`).
- **Fail-safe:** shields the Go service from raw TCP and is ready for future
  SSL/TLS termination (e.g. Let's Encrypt).
- **Ordered startup.** Postgres must pass `pg_isready` before the API starts;
  the API must answer `GET /health` (`wget --spider`) before Nginx starts — no
  `502 Bad Gateway` on a cold start.
- **Only the entrypoints are published.** `nginx` exposes `80`, `db` exposes
  `5432` for local tooling and integration tests; `api` is reachable only via
  the internal bridge.

## Local network testing from a phone

You can hit the stack from a secondary device (phone, tablet) on the same
network — handy on tethered hotspots or dorm Wi-Fi.

```sh
# 1. start the stack
docker compose up --build

# 2. find the host IP on the active interface
ip route | grep src
# e.g. wlp0s20f3 ... src 192.168.1.42

# 3. from the phone (same Wi-Fi) hit Nginx directly
http GET  http://<HOST_IP>/health
http POST http://<HOST_IP>/users name="Anri"
http GET  http://<HOST_IP>/users/1
```

Replace `<HOST_IP>` with the address from step 2. Requests will show up in the
host's `docker compose` logs, forwarded through `nginx-1`.
