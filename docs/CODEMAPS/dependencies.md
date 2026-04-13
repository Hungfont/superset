<!-- Generated: 2026-04-13 | Files scanned: ~3 | Token estimate: ~350 -->

# Dependencies Codemap

## Go (backend/go.mod)

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/gin-gonic/gin` | v1.12.0 | HTTP router & framework |
| `github.com/golang-jwt/jwt/v5` | v5.3.1 | RS256 JWT sign & verify |
| `github.com/google/uuid` | v1.6.0 | UUID generation |
| `github.com/joho/godotenv` | v1.5.1 | `.env` file loading |
| `github.com/redis/go-redis/v9` | v9.18.0 | Redis client |
| `github.com/stretchr/testify` | v1.11.1 | Test assertions |
| `golang.org/x/crypto` | v0.48.0 | bcrypt password hashing |
| `gorm.io/driver/postgres` | v1.5.7 | PostgreSQL driver for GORM |
| `gorm.io/gorm` | v1.25.10 | ORM (AutoMigrate, queries) |

## npm (frontend/package.json)

| Package | Purpose |
|---------|---------|
| React + TypeScript | UI framework |
| Vite | Build tool & dev server |
| Tailwind CSS | Utility-first styling |
| shadcn/ui | Component library (Radix-based) |
| Zustand | Lightweight state management |
| Zod | Schema validation (forms) |
| Vitest | Unit test runner |
| `components.json` | shadcn registry config |

## External Services

| Service | Used By | Purpose |
|---------|---------|---------|
| PostgreSQL | backend | Primary data store |
| Redis | backend | Token store, rate limiting, JWT blocklist |
| SMTP server | backend (`pkg/email`) | Email verification links |

## Config & Secrets (env vars)

| Var | Purpose |
|-----|---------|
| `DB_DSN` | PostgreSQL connection string |
| `REDIS_URL` | Redis connection URL |
| `JWT_PRIVATE_KEY` | RSA private key PEM (signing) |
| `JWT_PUBLIC_KEY` | RSA public key PEM (verification) |
| `SMTP_HOST/PORT/USERNAME/PASSWORD/FROM` | SMTP credentials |
| `APP_BASE_URL` | Base URL for email verification links |
| `APP_PORT` | Server listen port |

## Related

- [backend.md](backend.md) — how packages are used
- [data.md](data.md) — Redis key patterns
