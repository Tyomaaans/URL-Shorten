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

| Method | Endpoint | Access | Notes |
| --- | --- | --- | --- |
| POST | `/api/v1/auth/register` | Public | Daftarkan akun baru dengan email & password |
| POST | `/api/v1/auth/login` | Public | Login dan dapatkan access token + refresh token |
| POST | `/api/v1/auth/refresh` | Public | Perbarui access token menggunakan refresh token yang valid |
| POST | `/api/v1/auth/logout` | Authenticated | Cabut sesi aktif saat ini dan invalidasi token |

</details>

---

<details>
<summary><strong>Users</strong> — 7 endpoints</summary>

<details>
<summary>&nbsp;&nbsp;&nbsp;&nbsp;<strong>CRUD</strong> — 4 endpoints</summary>

| Method | Endpoint | Access | Notes |
| --- | --- | --- | --- |
| GET | `/api/v1/users/me` | Authenticated | Ambil data profil pengguna yang sedang login |
| PATCH | `/api/v1/users/me` | Authenticated | Perbarui sebagian data profil (nama, dll.) |
| PUT | `/api/v1/users/me` | Authenticated | Ganti password akun saat ini |
| DELETE | `/api/v1/users/me` | Authenticated | Hapus akun dan semua data yang terkait secara permanen |

</details>

<details>
<summary>&nbsp;&nbsp;&nbsp;&nbsp;<strong>Sessions</strong> — 4 endpoints</summary>

| Method | Endpoint | Access | Notes |
| --- | --- | --- | --- |
| GET | `/api/v1/users/me/sessions` | Authenticated | Tampilkan semua sesi aktif milik pengguna saat ini |
| DELETE | `/api/v1/users/me/sessions/:sid` | Authenticated | Cabut satu sesi spesifik berdasarkan ID sesi (`sid`) |
| DELETE | `/api/v1/users/me/sessions/others` | Authenticated | Cabut semua sesi aktif kecuali sesi saat ini |
| DELETE | `/api/v1/users/me/sessions` | Authenticated | Cabut seluruh sesi aktif termasuk sesi saat ini |

</details>

</details>

---

<details>
<summary><strong>Shortens</strong> — 8 endpoints</summary>

<details>
<summary>&nbsp;&nbsp;&nbsp;&nbsp;<strong>Public</strong> — 2 endpoints</summary>

| Method | Endpoint | Access | Notes |
| --- | --- | --- | --- |
| POST | `/api/v1/s` | Public | Buat short link tanpa akun; expired otomatis dalam 3 hari |
| GET | `/api/v1/s/:code` | Public | Redirect ke URL asli berdasarkan kode pendek (`code`) |

</details>

<details>
<summary>&nbsp;&nbsp;&nbsp;&nbsp;<strong>Authenticated</strong> — 6 endpoints</summary>

| Method | Endpoint | Access | Notes |
| --- | --- | --- | --- |
| POST | `/api/v1/shortens/me` | Authenticated | Buat short link dengan opsi `expires_at` kustom atau permanen |
| GET | `/api/v1/shortens/me` | Authenticated | Tampilkan semua short link milik pengguna yang login |
| GET | `/api/v1/shortens/me/:shid` | Authenticated | Ambil metadata lengkap dari satu short link berdasarkan ID |
| PATCH | `/api/v1/shortens/me/:shid` | Authenticated | Perbarui target URL atau detail short link |
| PUT | `/api/v1/shortens/me/:shid/status` | Authenticated | Ubah status aktif link via query string `?active=true/false` |
| DELETE | `/api/v1/shortens/me/:shid` | Authenticated | Hapus short link dan bersihkan cache yang terkait |

</details>

</details>

---

<details>
<summary><strong>Admin</strong> — 16 endpoints</summary>

<details>
<summary>&nbsp;&nbsp;&nbsp;&nbsp;<strong>User CRUD</strong> — 5 endpoints</summary>

| Method | Endpoint | Access | Notes |
| --- | --- | --- | --- |
| GET | `/api/v1/admin/users` | Authenticated + Admin Secret | Tampilkan daftar semua pengguna terdaftar di sistem |
| GET | `/api/v1/admin/users/:sub` | Authenticated + Admin Secret | Ambil data profil pengguna tertentu berdasarkan ID (`sub`) |
| PATCH | `/api/v1/admin/users/:sub` | Authenticated + Admin Secret | Perbarui sebagian data profil pengguna target |
| PUT | `/api/v1/admin/users/:sub` | Authenticated + Admin Secret | Reset atau ubah password pengguna target secara paksa |
| DELETE | `/api/v1/admin/users/:sub` | Authenticated + Admin Secret | Hapus akun pengguna target beserta seluruh datanya |

</details>

<details>
<summary>&nbsp;&nbsp;&nbsp;&nbsp;<strong>User Sessions</strong> — 3 endpoints</summary>

| Method | Endpoint | Access | Notes |
| --- | --- | --- | --- |
| GET | `/api/v1/admin/users/:sub/sessions` | Authenticated + Admin Secret | Tampilkan semua sesi aktif milik pengguna target |
| DELETE | `/api/v1/admin/users/:sub/sessions/:sid` | Authenticated + Admin Secret | Cabut satu sesi spesifik milik pengguna target berdasarkan `sid` |
| DELETE | `/api/v1/admin/users/:sub/sessions` | Authenticated + Admin Secret | Cabut seluruh sesi aktif milik pengguna target |

</details>

<details>
<summary>&nbsp;&nbsp;&nbsp;&nbsp;<strong>Shortens</strong> — 5 endpoints</summary>

| Method | Endpoint | Access | Notes |
| --- | --- | --- | --- |
| GET | `/api/v1/admin/shortens` | Authenticated + Admin Secret | Tampilkan semua short link yang ada di seluruh sistem |
| GET | `/api/v1/admin/shortens/:shid` | Authenticated + Admin Secret | Ambil detail satu short link sistem berdasarkan ID |
| GET | `/api/v1/admin/users/:sub/shortens` | Authenticated + Admin Secret | Tampilkan semua short link milik pengguna target (`sub`) |
| PATCH | `/api/v1/admin/users/:sub/shortens/:shid` | Authenticated + Admin Secret | Perbarui short link milik pengguna target secara paksa |
| DELETE | `/api/v1/admin/users/:sub/shortens/:shid` | Authenticated + Admin Secret | Hapus short link milik pengguna target secara paksa |

</details>

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