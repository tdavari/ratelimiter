# Rate Limiter — README

Simple README for the **distributed sliding-window rate limiter** (Go + Redis).

---

## What this repo contains

- **`internal/ratelimiter`** — sliding-window limiter implementation (Lua + Go), tests and benchmark.
- **`main.go`** — small demo that uses the limiter and prints allow/reject for a few users.
- **`.env`** — Redis config for app & tests.
- **`docker-compose.yml`** — Redis + test runner service that runs unit tests & benchmarks.
- **`go.mod`, `go.sum`**

**Goal:** global per-user rate limiting across multiple instances using Redis sorted sets + a Lua script (atomic).
Uses Redis server time to avoid clock skew between instances.

---

## 🚀 Quick start (recommended: Docker)

Make sure you have **Docker** & **Docker Compose** installed.

From the project root (where `docker-compose.yml` and `.env` live):

```bash
# Build and run services, test output appears in the console
docker compose up --build
