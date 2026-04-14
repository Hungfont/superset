<!-- Generated: 2026-04-14 | Files scanned: 120 | Token estimate: ~610 -->

# Dependencies Codemap

## Backend (`backend/go.mod`)

Core runtime deps:

- `github.com/gin-gonic/gin v1.12.0`: HTTP server/router.
- `gorm.io/gorm v1.25.10` + `gorm.io/driver/postgres v1.5.7`: ORM + Postgres driver.
- `github.com/redis/go-redis/v9 v9.18.0`: Redis client.
- `github.com/golang-jwt/jwt/v5 v5.3.1`: JWT parsing/signing.
- `github.com/google/uuid v1.6.0`: UUID generation.
- `golang.org/x/crypto v0.48.0`: bcrypt.
- `github.com/joho/godotenv v1.5.1`: local env loading.

Testing:

- `github.com/stretchr/testify v1.11.1`.

## Frontend (`frontend/package.json`)

App/runtime deps:

- `react`, `react-dom`, `react-router-dom`.
- `@tanstack/react-query`.
- `zustand`.
- `react-hook-form` + `@hookform/resolvers` + `zod`.
- `@tanstack/react-table`.
- `lucide-react`.
- Radix primitives (`@radix-ui/*`) with `class-variance-authority`, `clsx`, `tailwind-merge`, `tailwindcss-animate`.

Tooling/test deps:

- `vite`, `@vitejs/plugin-react-swc`, `typescript`.
- `vitest`, `@vitest/coverage-v8`, `jsdom`, Testing Library packages.
- `tailwindcss`, `postcss`, `autoprefixer`.

## External Services and Infra

- PostgreSQL: primary relational store.
- Redis: blocklist, refresh sessions, role cache, rate limiting.
- SMTP server: verification email delivery.

## Environment Contract

- `DB_DSN`
- `REDIS_URL`
- `JWT_PRIVATE_KEY`, `JWT_PUBLIC_KEY`
- `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM`
- `APP_BASE_URL`, `APP_PORT`

## Dependency Risk Notes

- JWT correctness depends on valid RSA PEM env values at startup.
- Redis availability impacts refresh/logout/rate-limiting and role caching.
- SMTP failures affect registration completion path.
