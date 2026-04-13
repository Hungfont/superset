<!-- Generated: 2026-04-13 | Files scanned: ~5 | Token estimate: ~350 -->

# Data Codemap

## PostgreSQL Tables

### `ab_register_user` — pending email verification

| Column | Type | Notes |
|--------|------|-------|
| id | uint PK | auto-increment |
| first_name | string | not null |
| last_name | string | not null |
| username | string | unique index |
| email | string | unique index |
| password | string | bcrypt hash |
| registration_hash | string | unique, used in verify link |
| created_at | timestamp | auto |

### `ab_user` — activated accounts

| Column | Type | Notes |
|--------|------|-------|
| id | uint PK | auto-increment |
| first_name | string | not null |
| last_name | string | not null |
| username | string | unique index |
| email | string | unique index |
| password | string | bcrypt hash |
| active | bool | default true |
| login_count | int | default 0 |
| last_login | timestamp | nullable |
| created_on | timestamp | auto |
| changed_on | timestamp | auto-update |

**Migration:** GORM AutoMigrate in `main.go` at startup.

## Redis Key Spaces

| Key pattern | Store | Purpose |
|-------------|-------|---------|
| `jwt:blocklist:<jti>` | `redis/jwt_repo.go` | Revoked access tokens |
| `refresh:<token>` | `redis/refresh_repo.go` | Active refresh tokens → user ID |
| `rate:<username>` | `redis/rate_repo.go` | Login attempt counter (rate limiting) |

## Domain Entities

Source: `backend/internal/domain/auth/entity.go`

- `RegisterUser` — pre-verification record
- `User` — active account
- `UserContext` — injected by JWT middleware into Gin context
- `LoginResponse` — `{access_token, refresh_token}`

## Extended Schema Docs

See [docs/db/](../db/) for full schema documentation across all planned services.

## Related

- [backend.md](backend.md) — repository implementations
- [architecture.md](architecture.md) — infra overview
