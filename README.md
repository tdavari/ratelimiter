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

=== RUN   TestRateLimiter_SlidingWindow
--- PASS: TestRateLimiter_SlidingWindow (2.11s)
=== RUN   TestRateLimiter_ConcurrentMultipleUsers
--- PASS: TestRateLimiter_ConcurrentMultipleUsers (0.01s)
BenchmarkRateLimiter_ManyUsers-8    30000    35000 ns/op    960 B/op    29 allocs/op

How the limiter works (short) Each request is represented by a unique member (UUID) in a Redis sorted set ratelimiter:user:<id>. The Lua script (executed atomically) does: Reads Redis server time with TIME → builds a float timestamp now. ZREMRANGEBYSCORE key 0 now - window — removes old entries outside the sliding window. ZCARD key — counts current requests in the window. If count < limit → ZADD key now member and return 1; otherwise return 0. EXPIRE key 3600 — resets TTL on every request (so keys are auto-cleaned). Using Redis server time avoids clock skew between distributed instances.