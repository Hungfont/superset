<!-- Generated: 2026-04-13 | Files scanned: ~35 | Token estimate: ~500 -->

# Backend Codemap

**Entry Point:** `backend/cmd/api/main.go`  
**Module:** `superset/auth-service`  
**Runtime:** Go 1.25, Gin framework

## Routes

```
POST /api/v1/auth/register   → RegisterHandler.Register
GET  /api/v1/auth/verify     → VerifyHandler.Verify
POST /api/v1/auth/login      → LoginHandler.Login
POST /api/v1/auth/refresh    → RefreshHandler.Refresh
POST /api/v1/auth/logout     → LogoutHandler.Logout
[protected group]  /api/v1/  → middleware.JWTMiddleware (RS256 + Redis blocklist)
```

## Key Files

| File | Purpose |
|------|---------|
| `cmd/api/main.go` | Bootstrap: DB, Redis, RSA keys, DI wiring, server start |
| `internal/delivery/http/router.go` | Gin engine, route groups, middleware attachment |
| `internal/delivery/http/auth/register_handler.go` | Parse RegisterRequest, call RegisterService |
| `internal/delivery/http/auth/verify_handler.go` | Email verification via hash token |
| `internal/delivery/http/auth/login_handler.go` | Credential check, issue JWT + refresh cookie |
| `internal/delivery/http/auth/refresh_handler.go` | Rotate refresh token, issue new access JWT |
| `internal/delivery/http/auth/logout_handler.go` | Revoke access/refresh tokens and clear refresh cookie |
| `internal/delivery/http/middleware/jwt.go` | RS256 JWT validation, blocklist check, inject UserContext |
| `internal/app/auth/register_service.go` | Hash password (bcrypt), persist RegisterUser, send email |
| `internal/app/auth/verify_service.go` | Promote RegisterUser → User on hash match |
| `internal/app/auth/login_service.go` | Authenticate, rate-limit, sign RS256 JWT |
| `internal/app/auth/refresh_service.go` | Validate refresh token, rotate, re-sign JWT |
| `internal/app/auth/logout_service.go` | Blacklist access token jti and revoke refresh sessions |
| `internal/domain/auth/entity.go` | RegisterUser, User, request/response types, UserContext |
| `internal/domain/auth/repository.go` | Repository interfaces (UserRepository, JWTRepository, …) |
| `internal/domain/auth/errors.go` | Sentinel errors |
| `internal/repository/postgres/` | GORM implementations: register, verify, login, user repos |
| `internal/repository/redis/` | Redis implementations: JWT blocklist, refresh tokens, rate limits |
| `internal/pkg/email/sender.go` | SMTP email sender |
| `internal/pkg/validator/password.go` | Password strength validation |
| `configs/` | Config loader (env vars → struct) |

## Dependency Injection

All wiring happens in `main.go` — no DI framework. Constructor injection throughout:
```
NewRegisterService(registerRepo, mailer, baseURL)
NewLoginService(loginRepo, rateRepo, refreshRepo, privKey)
NewRefreshService(refreshRepo, userRepo, privKey)
```

## Related

- [architecture.md](architecture.md) — layer diagram
- [data.md](data.md) — DB tables & Redis keys
- [dependencies.md](dependencies.md) — Go modules
