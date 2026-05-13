<!-- Generated: 2026-05-13 | Files scanned: ~250 | Token estimate: ~700 -->

# Dependencies Codemap

## Backend (`backend/go.mod`)

Core runtime deps:

- `github.com/gin-gonic/gin v1.12.0`: HTTP server/router.
- `github.com/gorilla/websocket` (indirect): WebSocket support (QE-005).
- `gorm.io/gorm v1.25.10` + `gorm.io/driver/postgres v1.5.7`: ORM + Postgres driver.
- `github.com/jackc/pgx/v5 v5.5.5`: direct Postgres driver for pool connections.
- `github.com/redis/go-redis/v9 v9.18.0`: Redis client (cache, pub/sub, queues).
- `github.com/golang-jwt/jwt/v5 v5.3.1`: JWT parsing/signing (RS256).
- `github.com/google/uuid v1.6.0`: UUID generation.
- `golang.org/x/crypto v0.48.0`: bcrypt password hashing.
- `golang.org/x/sync v0.20.0`: concurrency primitives.
- `github.com/joho/godotenv v1.5.1`: local env loading.

Testing:

- `github.com/stretchr/testify v1.11.1`: assertions and mocking.
- `github.com/go-redis/redismock/v9 v9.0.0`: Redis mocking for tests.

## Frontend (`frontend/package.json`)

App/runtime deps:

- `react ^18.3.1`, `react-dom ^18.3.1`.
- `react-router-dom ^6.26.2`.
- `@tanstack/react-query ^5.56.2`.
- `@tanstack/react-table ^8.21.3`.
- `zustand ^5.0.0`: state management (authStore, sqlLabStore, wsStore).
- `react-hook-form ^7.72.1` + `@hookform/resolvers ^3.10.0` + `zod ^3.25.76`.
- `lucide-react ^0.462.0`: icon library.
- `cmdk ^1.1.1` (command palette primitives used by admin matrix search).
- `sonner ^2.0.7` (toast notifications).
- Radix primitives (`@radix-ui/*`): alert-dialog, avatar, checkbox, dialog, dropdown-menu,
  label, popover, progress, scroll-area, select, separator, slot, switch, tabs, tooltip.
- Styling helpers: `class-variance-authority ^0.7.1`, `clsx ^2.1.1`, `tailwind-merge ^2.5.2`, `tailwindcss-animate ^1.0.7`.

Tooling/test deps:

- `vite ^5.4.8`, `@vitejs/plugin-react-swc ^3.5.0`, `typescript ^5.5.3`.
- `vitest ^2.1.1`, `@vitest/coverage-v8 ^2.1.1`, `jsdom ^25.0.1`, Testing Library packages (`@testing-library/react`, `@testing-library/jest-dom`, `@testing-library/user-event`).
- `tailwindcss ^3.4.13`, `postcss ^8.4.47`, `autoprefixer ^10.4.20`.

## External Services and Infra

- PostgreSQL: primary relational store (15 tables after auto-migrate), `pg_trgm` extension for GIN index (QE-007).
- Redis: blocklist, refresh sessions, role cache, rate limiting, dataset queues, query result cache, async query queues, pub/sub channels (QE-004/005/006).
- External databases: connection via pool manager (max 10 open, 3 idle, 30min lifetime), query execution, schema introspection.
- SMTP server: verification email delivery.

## Environment Contract

Backend (`backend/.env`):
- `DATABASE_URL`
- `REDIS_URL`
- `JWT_PRIVATE_KEY`, `JWT_PUBLIC_KEY`
- `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM`
- `APP_BASE_URL`, `APP_PORT`
- `DB_CREDENTIALS_ENCRYPTION_KEY`

Frontend (`frontend/.env`):
- `VITE_API_URL` (defaults to `http://localhost:8080`)

## Dependency Risk Notes

- JWT correctness depends on valid RSA PEM env values at startup.
- WebSocket connections (QE-005) require Redis pub/sub functioning for real-time updates; fallback to polling available.
- Redis availability impacts refresh/logout/rate-limiting, role caching, async query queues, and WS streaming.
- External DB pools require careful lifecycle management to prevent connection leaks; graceful shutdown in main.go.
- SMTP failures affect registration completion path.
- `pg_trgm` extension required for query history ILIKE search performance (QE-007); installed at startup via `CREATE EXTENSION IF NOT EXISTS`.
