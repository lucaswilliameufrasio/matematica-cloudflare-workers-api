**Overview**

Production-ready Golang backend for a real-time math game with operator and expressions modes, adaptive difficulty, ratings, leaderboards, drills, AI coach scaffolding, and multiplayer foundation.

**Tech Stack**
- Language: Go 1.22+
- Web: Chi router
- WebSocket: nhooyr.io/websocket
- DB: PostgreSQL (pgx v5), migrations with goose (library)
- Cache/RT: Valkey/Redis (go-redis v9), Pub/Sub ready
- In-memory cache: ristretto + singleflight via reusable driver
- Auth: bcrypt + JWT (scaffold)
- Config: envconfig + .env via godotenv
- Docs: OpenAPI 3.1 at `/api/openapi.yaml`, Redoc at `/docs`
- Logging: zerolog; request IDs
- Metrics: Prometheus at `/metrics`
- Rate limit: Sliding window per IP
- Testing: `go test` with unit tests
- Makefile: `make dev`, `make test`, `make migrate`, `make seed`
- Docker: Multi-stage Dockerfile; docker-compose with Postgres & Valkey

**Run locally**
- Create `.env` (see `.env.example`).
- `make dev` to run the server.
- `curl http://localhost:8080/docs` for docs.
- `curl -X POST localhost:8080/v1/expressions/next -d '{"mode":"mixed","max_depth":4,"min_binary_ops":1,"allow_unary_start":true}' -H 'Content-Type: application/json'`

**Docker Compose**
- `docker-compose up --build`
- API at `http://localhost:8080` (auto-migrates on start)

**Environment**
- `DATABASE_URL` (required)
- `REDIS_ADDR` (default `localhost:6379`)
- `HTTP_ADDR` (default `:8080`)
- `AUTO_MIGRATE` (default `true`)
- `JWT_SECRET` (default dev secret; change in prod)

**OpenAPI**
- Spec served at `/api/openapi.yaml`
- Redoc served at `/docs`

**Notes**
- Expressions endpoints are fully implemented with server-side deterministic evaluation.
- Additional domains (duels, operator practice, leaderboards, AI coach) are scaffolded via schema and infra; extend handlers as needed.

