<!-- Generated: 2026-04-13 | Files scanned: ~60 | Token estimate: ~400 -->

# Architecture

## System Boundaries

```
Browser
  └── React/TS (Vite + Tailwind)
        └── REST → Go Auth Service (:PORT)
                      ├── PostgreSQL  (user storage)
                      └── Redis       (JWT blocklist, refresh tokens, rate limits)
```

## Request Flow

```
Client
  → POST /api/v1/auth/register  → RegisterHandler → RegisterService → PostgreSQL + SMTP
  → GET  /api/v1/auth/verify    → VerifyHandler   → VerifyService  → PostgreSQL
  → POST /api/v1/auth/login     → LoginHandler    → LoginService   → PostgreSQL + Redis
  → POST /api/v1/auth/refresh   → RefreshHandler  → RefreshService → Redis + PostgreSQL
  → [protected] /api/v1/*       → JWTMiddleware   → validates RS256 JWT via Redis blocklist
```

## Layer Stack (backend)

```
cmd/api/main.go          — wires all deps, starts server
delivery/http/router.go  — Gin route registration
delivery/http/auth/      — HTTP handlers (input parsing, response shaping)
delivery/http/middleware/ — JWT auth middleware
app/auth/                — business logic (services)
repository/postgres/     — GORM-backed DB access
repository/redis/        — Redis-backed token/rate stores
domain/auth/             — entities, interfaces, errors (no deps)
```

## Sequence Diagrams

See [docs/diagram/sequence/](../diagram/sequence/) for detailed flows:
- Authentication & Session Management
- Registration & Email Verification
