# Ophelia

Semantic search engine for the Arch Linux Wiki. Submit an error message or a technical query and get back ranked, relevant Wiki pages — no keyword guessing required.

## How it works

1. The query goes through a preprocessing pipeline that strips stop words to improve signal before hitting the search API.
2. The cleaned query is sent to the Alchemyst Context API, which runs vector similarity scoring against indexed Arch Wiki content.
3. Results are bucketed into high / medium / low relevance tiers and mapped back to real Arch Wiki URLs.
4. Responses are cached in Redis by MD5-keyed query for 5 minutes. Analytics tracking runs asynchronously in goroutines so it never blocks the response path.

## Stack

**Backend (Go 1.24)**
- [Gin](https://github.com/gin-gonic/gin) — HTTP router
- [GORM](https://gorm.io) — ORM with PostgreSQL driver
- [go-redis](https://github.com/redis/go-redis) — caching layer
- [gocolly](https://github.com/gocolly/colly) + [goquery](https://github.com/PuerkitoBio/goquery) — Arch Wiki content ingestion
- [Logrus](https://github.com/sirupsen/logrus) — structured logging
- [Viper](https://github.com/spf13/viper) — configuration

**Infrastructure**
- PostgreSQL — primary database
- Redis — search result cache (5-min TTL)
- Alchemyst Context API — external vector search service

**Frontend**
- Next.js 15 / React 19
- Radix UI + Tailwind CSS v4

## Features

- **Semantic search** — results ranked by vector similarity with configurable thresholds (0.7 high, 0.3 floor)
- **Query preprocessing** — stop word stripping before the vector call; falls back to the raw query if preprocessing removes too much content
- **Redis cache-aside** — check cache first, fall back to live API, write result back
- **Exponential backoff** — up to 4 retries, 1.5x backoff, 2s base delay, 15s cap, context-cancellation aware
- **Async analytics** — query logging, popular query tracking, and stats updates run in goroutines
- **Health endpoints** — `/health` (cached) and `/health/detailed` (live); returns `503` automatically if any dependency is unhealthy
- **Rate limiting** — 100 req/min per client, security headers, request ID injection
- **Graceful shutdown** — catches `SIGINT`/`SIGTERM`, allows 30 seconds for in-flight requests before forcing exit
- **Wiki ingestion** — `alchemyst` service scrapes and uploads Arch Wiki pages as context documents, with source-keyed deduplication before re-upload

## Architecture

```
cmd/server          entry point, wires dependencies
internal/
  api/              Gin router, handlers
  services/         business logic
  repository/       GORM-backed data layer
  alchemyst/        adapter for the external vector search API
  middleware/        rate limiting, auth, request ID
  models/           SearchQuery, UserFeedback, PopularQuery, SystemHealth, ...
```

All packages live under `internal/` with no cross-layer imports. The `alchemyst` package is a fully isolated integration adapter — business logic has no direct dependency on the external API shape.

## Running locally

```bash
# Copy and fill in environment variables
cp .env.example .env

# Start dependencies
docker compose up -d

# Run the backend
go run ./cmd/server

# Run the frontend
cd frontend && npm install && npm run dev
```

## Environment variables

| Variable | Description |
|---|---|
| `DATABASE_URL` | PostgreSQL connection string |
| `REDIS_URL` | Redis connection string |
| `ALCHEMYST_API_KEY` | API key for the Alchemyst Context API |
| `ALCHEMYST_BASE_URL` | Base URL for the Alchemyst API |
| `PORT` | Server port (default `8080`) |
