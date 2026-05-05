<!-- Generated: 2026-05-05 | Files scanned: 180 | Token estimate: ~650 -->

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

Additional runtime:

- `database/sql`: built-in SQL driver interface.
- `github.com/lib/pq`: Postgres driver (used via GORM).

## Frontend (`frontend/package.json`)

App/runtime deps:

- `react`, `react-dom`, `react-router-dom`.
- `@tanstack/react-query`.
- `zustand`.
- `react-hook-form` + `@hookform/resolvers` + `zod`.
- `@tanstack/react-table`.
- `lucide-react`.
- `cmdk` (command palette primitives used by admin matrix search).
- `sonner` (toast notifications).
- Radix primitives (`@radix-ui/*`) including alert-dialog, avatar, checkbox, dialog, dropdown-menu,
  label, progress, scroll-area, separator, slot, tabs, tooltip.
- Styling helpers: `class-variance-authority`, `clsx`, `tailwind-merge`, `tailwindcss-animate`.

Tooling/test deps:

- `vite`, `@vitejs/plugin-react-swc`, `typescript`.
- `vitest`, `@vitest/coverage-v8`, `jsdom`, Testing Library packages.
- `tailwindcss`, `postcss`, `autoprefixer`.

## External Services and Infra

- PostgreSQL: primary relational store.
- Redis: blocklist, refresh sessions, role cache, rate limiting, dataset queues, query result cache.
- External databases: connection via pool manager, query execution, schema introspection.
- SMTP server: verification email delivery.

## Environment Contract

- `DATABASE_URL`
- `REDIS_URL`
- `JWT_PRIVATE_KEY`, `JWT_PUBLIC_KEY`
- `SMTP_HOST`, `SMTP_PORT`, `SMTP_USERNAME`, `SMTP_PASSWORD`, `SMTP_FROM`
- `APP_BASE_URL`, `APP_PORT`

## Dependency Risk Notes

- JWT correctness depends on valid RSA PEM env values at startup.
- Redis availability impacts refresh/logout/rate-limiting and role caching.
- External DB pools require careful lifecycle management to prevent connection leaks.
- SMTP failures affect registration completion path.
