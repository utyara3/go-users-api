-- Schema for the users API.
-- Mounted into the postgres container, so `docker compose up` creates the table automatically.

CREATE TABLE IF NOT EXISTS users (
    id   BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL
);
