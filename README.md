# URL Shortener API

A URL shortening and link management API built with Go, Gin, PostgreSQL, and Redis. This repository is the standalone version of the URL shortener used by my web profile and includes its shared authentication flow.

## Live project

- Demo page: [https://tyomaaans.cloud/url-shortener](https://tyomaaans.cloud/url-shortener)
- API base URL: `https://api.tyomaaans.cloud/v1/url-shortener`
- Swagger UI: [https://api.tyomaaans.cloud/swagger/url-shortener/index.html](https://api.tyomaaans.cloud/swagger/url-shortener/index.html)

Cookie-based requests such as refresh, heartbeat, and logout must include credentials. In a browser client, use `credentials: "include"`.

## Features

- Public short links with a fixed three-day expiration.
- Authenticated short links with optional custom expiration.
- Redis-first redirects with PostgreSQL fallback.
- Link ownership, listing, detail, update, status, and delete operations.
- A background worker that marks expired links inactive and clears their cache.
- Registration, login, refresh token rotation, heartbeat, logout, and multi-session management.
- User profile and admin user management.
- Admin link inspection and management across users.
- Absolute HTTP(S) URL validation for every create and update flow.
- Exact sliding-window limits for public link creation and token-bucket policies for other routes.
- Swagger documentation and a health endpoint.

## Main routes

All application routes use the `/v1/url-shortener` prefix.

### Short links

| Method | Route | Access |
| --- | --- | --- |
| POST | `/s` | Public |
| GET | `/s/:code` | Public redirect |
| POST | `/shortens/me` | Bearer token |
| GET | `/shortens/me` | Bearer token |
| GET | `/shortens/me/:shid` | Bearer token |
| PATCH | `/shortens/me/:shid` | Bearer token |
| PUT | `/shortens/me/:shid?status=active` | Bearer token |
| DELETE | `/shortens/me/:shid` | Bearer token |
| GET | `/admin/shortens` | Admin bearer token |
| GET | `/admin/shortens/:shid` | Admin bearer token |
| GET | `/admin/users/:sub/shortens` | Admin bearer token |
| PATCH | `/admin/users/:sub/shortens/:shid` | Admin bearer token |
| PUT | `/admin/users/:sub/shortens/:shid?status=active` | Admin bearer token |
| DELETE | `/admin/users/:sub/shortens/:shid` | Admin bearer token |

### Authentication and users

| Method | Route | Access |
| --- | --- | --- |
| POST | `/auth/register` | Public |
| POST | `/auth/login` | Public |
| POST | `/auth/refresh` | Session cookies |
| POST | `/auth/heartbeat` | Session cookie |
| POST | `/auth/logout` | Bearer token and cookies |
| GET, PATCH, DELETE | `/users/me` | Bearer token |
| GET, DELETE | `/users/me/sessions` | Bearer token |
| DELETE | `/users/me/sessions/:sid` | Bearer token |
| DELETE | `/users/me/sessions/others` | Bearer token |
| GET, PATCH, DELETE | `/admin/users/:sub` | Admin bearer token |
| GET, DELETE | `/admin/users/:sub/sessions` | Admin bearer token |
| DELETE | `/admin/users/:sub/sessions/:sid` | Admin bearer token |

`GET /health` and `GET /swagger/url-shortener/*any` are public utility routes.

## Public rate limit

Public link creation uses two rolling 72-hour windows:

- 5 attempts for the normalized IP and User-Agent combination.
- 25 attempts for the IP address to limit User-Agent rotation.

Rate-limited responses include `Retry-After`, `X-RateLimit-*` headers, `requests_left`, and `reset_at`.

## Project structure

```text
.
├── docs/url-shortener
├── internal
│   ├── auth-session
│   ├── url-shortener
│   │   ├── domain
│   │   ├── shortener
│   │   └── user
│   ├── config
│   ├── infrastructure
│   ├── middleware
│   └── router
├── pkg
├── main.go
└── docker-compose.yml
```

The auth-session package is included because this standalone API owns its user and session routes. Both features use the same user table, JWT service, Redis client, and middleware.

## Run locally

Requirements:

- Go 1.26.6
- Docker and Docker Compose

```bash
cp .env.example .env
docker compose up --build
```

For a direct Go run, start PostgreSQL and Redis first, update `.env` so the service addresses are reachable from the host, then run:

```bash
go run .
```

The application migrates the `users` and `shorten_storages` tables and creates an admin user from `ADMIN_PASSWORD` when no admin exists.

## Configuration

| Variable | Description |
| --- | --- |
| `APP_ENV` | Use `production` to enable Secure cookies |
| `APP_PORT` | HTTP server port |
| `APP_URL` | Public application URL |
| `DATABASE_URL` | PostgreSQL connection string |
| `REDIS_ADDR` | Redis host and port |
| `REDIS_PASSWORD` | Redis password |
| `JWT_SECRET_KEY` | JWT signing secret |
| `JWT_EXPIRY` | Access token duration, for example `15m` |
| `DEFAULT_REFRESH_EXPIRY` | Remembered session duration |
| `SHORT_REFRESH_EXPIRY` | Non-remembered session duration |
| `ADMIN_PASSWORD` | Password used when seeding the admin account |

Compose also reads `POSTGRES_USER`, `POSTGRES_PASSWORD`, and `POSTGRES_DB`.

## Development checks

```bash
make build
make test
make vet
docker compose config --quiet
```

Regenerate Swagger after changing handler annotations:

```bash
make swagger
```

The tests cover user services and handlers, short-link URL validation, route policy order, pagination, validators, and rate-limit behavior.
