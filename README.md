# URL Shortener API

A production-ready REST API for high-performance URL shortening and link management built with Go. Designed for low-latency redirection and automated lifecycle management — featuring public ephemeral links, authenticated full link ownership, Redis caching with TTL, and automated background cleanup via Go Workers.

---

## Architecture Overview

```
Client
  │
  ▼
Gin HTTP Server
  │
  ├── Auth Middleware & Admin Secret Middleware
  │     └── Token validation via Redis (no DB round-trip)
  │
  ├── Handlers → Services → Repositories
  │                             │
  │                         PostgreSQL (Persistent URL & metadata store)
  │
  └── Redis Cache
        ├── Fast link resolution (code -> original URL)
        ├── Dynamic TTL matching link expiration date
        └── Permanent URL caching for authenticated users
  │
  └── Go Background Worker (Every 1 min tick)
        └── Scans expired links → syncs DB `is_active = false` → purges Redis cache

```

---

## Tech Stack

| Layer | Technology | Reason |
| --- | --- | --- |
| Language | Go | High performance, lightweight goroutines for worker processes |
| Framework | Gin | Fast HTTP routing and clean middleware chaining |
| Database | PostgreSQL | Relational datastore for persistent URL metadata and user ownership |
| Cache Store | Redis | Sub-millisecond URL redirection lookups with TTL enforcement |
| Background Jobs | Go Ticker Worker | Periodically syncs expired state between cache and database |
| Containerization | Docker & Docker Compose | Consistent local and containerized deployments |

---

## Features

### Dual-Tier Link Generation

* **Public Ephemeral Shortening:** Guest users can quickly shorten URLs without registering. Public links strictly enforce a **3-day expiration time (`expires_at`)**.
* **Authenticated Custom Expiration:** Logged-in users gain full control to specify custom expiration timestamps or create permanent links with no expiry.

### High-Performance Redis Caching & Expiry Lifecycle

* **Redis-First Redirection:** All active short links are cached in Redis to achieve instant redirects (`GET /s/:code`) without database load.
* **Synchronized TTL:** Links with an `expires_at` timestamp inherit a matching TTL inside Redis and self-expire automatically. Permanent links persist continuously until deleted or deactivated.
* **Automated Worker Cleanup:** A Go background worker runs every minute to detect newly expired URLs in PostgreSQL, update their state to `is_active = false`, and evict stale keys from Redis.

### Granular Link Management

* **User Link Dashboard:** Authenticated users can list, inspect, update destinations, toggle active states, or soft-delete their own short URLs.
* **Dynamic Status Toggling:** Toggle short link availability on demand via explicit query parameters (`PUT /s/:shid/status?active=true|false`).
* **Admin Link Supervision:** System administrators can audit, modify destination URLs, or remove short links across all users.

---

## API Endpoints

<details>
<summary><strong>Auth</strong> — 4 endpoints</summary>

| Method | Endpoint | Access |
| --- | --- | --- |
| POST | `/api/v1/auth/register` | Public |
| POST | `/api/v1/auth/login` | Public |
| POST | `/api/v1/auth/refresh` | Public |
| POST | `/api/v1/auth/logout` | Authenticated |

</details>

<details>
<summary><strong>Users</strong> — 2 endpoints</summary>

| Method | Endpoint | Access |
| --- | --- | --- |
| GET | `/api/v1/users/me` | Authenticated |
| PATCH | `/api/v1/users/me` | Authenticated |

</details>

<details>
<summary><strong>Sessions</strong> — 4 endpoints</summary>

| Method | Endpoint | Access |
| --- | --- | --- |
| GET | `/api/v1/users/me/sessions` | Authenticated |
| DELETE | `/api/v1/users/me/sessions/:sid` | Authenticated |
| DELETE | `/api/v1/users/me/sessions/others` | Authenticated |
| DELETE | `/api/v1/users/me/sessions` | Authenticated |

</details>

<details>
<summary><strong>Shorten</strong> — 8 endpoints</summary>

| Method | Endpoint | Access | Notes |
| --- | --- | --- | --- |
| POST | `/api/v1/s` | Public | Fixed 3-day expiration time |
| GET | `/api/v1/s/:code` | Public | Redirects short code to original target URL |

| Method | Endpoint | Access | Notes |
| --- | --- | --- | --- |
| POST | `/api/v1/shortens/me` | Authenticated | Supports custom `expires_at` or permanent links |
| GET | `/api/v1/shortens/me` | Authenticated | List all links owned by current user |
| GET | `/api/v1/shortens/me/:shid` | Authenticated | Get detailed metadata of a specific link |
| PATCH | `/api/v1/shortens/me/:shid` | Authenticated | Update target URL or link details |
| PUT | `/api/v1/shortens/me/:shid/status` | Authenticated | Update active status using query string (`?active=true/false`) |
| DELETE | `/api/v1/shortens/me/:shid` | Authenticated | Delete link and purge associated cache |

</details>

<details>
<summary><strong>Admin</strong> — 11 endpoints</summary>

| Method | Endpoint | Access |
| --- | --- | --- |
| GET | `/api/v1/admin/users` | Authenticated + Admin Secret |
| GET | `/api/v1/admin/users/:sub` | Authenticated + Admin Secret |
| PATCH | `/api/v1/admin/users/:sub` | Authenticated + Admin Secret |
| GET | `/api/v1/admin/users/:sub/sessions` | Authenticated + Admin Secret |
| DELETE | `/api/v1/admin/users/:sub/sessions/:sid` | Authenticated + Admin Secret |
| DELETE | `/api/v1/admin/users/:sub/sessions` | Authenticated + Admin Secret |
| GET | `/api/v1/admin/shortens` | Authenticated + Admin Secret | List all system-wide short links |
| GET | `/api/v1/admin/shortens/:shid` | Authenticated + Admin Secret | Get any short link by ID |
| GET | `/api/v1/admin/users/:sub/shortens` | Authenticated + Admin Secret | List all short links belonging to target user (`:sub`) |
| PATCH | `/api/v1/admin/users/:sub/shortens/:shid` | Authenticated + Admin Secret | Force update target user's short link |
| DELETE | `/api/v1/admin/users/:sub/shortens/:shid` | Authenticated + Admin Secret | Force delete target user's short link |

</details>

---

## Getting Started

### Prerequisites

* [Docker](https://docs.docker.com/get-docker/) & Docker Compose
* Go 1.22+

### Run Locally

```bash
# Clone the repository
git clone https://github.com/Tyomaaans/Auth-Session.git
cd auth-session-api

# Copy environment variables
cp .env.example .env

# Start PostgreSQL and Redis containers
docker compose up -d

```

### Environment Variables

See `.env.example` for all required variables. Key configs:

```env
# App
APP_ENV=
APP_PORT=
APP_URL=

# Database
POSTGRES_USER=
POSTGRES_PASSWORD=
POSTGRES_DB=
DATABASE_URL=

# Redis
REDIS_ADDR=
REDIS_PASSWORD=

# Admin
ADMIN_SECRET_KEY=

```

---

## Project Status

| Feature | Status | Notes |
| --- | --- | --- |
| Public Shorten Endpoint (`POST /s`) | ✅ Done | Auto expiration fixed at 3 days |
| Public Redirection (`GET /s/:code`) | ✅ Done | Fetches from Redis cache first |
| Authenticated Shorten Creation | ✅ Done | Supports custom `expires_at` or permanent links |
| Redis TTL & Permanent Caching | ✅ Done | Automatic TTL for expiring links, zero TTL for permanent |
| Active Status Toggle Query | ✅ Done | `/shortens/me/:shid/status?active=true/false` |
| Go Cleanup Worker | ✅ Done | Runs every 1 min: sets DB `is_active = false` & evicts Redis |
| User Link Dashboard CRUD | ✅ Done | View, update, or remove personal links |
| Admin Link Management | ✅ Done | Inspect, modify, or purge any link across users |