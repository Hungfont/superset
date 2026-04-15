<!-- Generated: 2026-04-14 | Files scanned: 120 | Token estimate: ~560 -->

# Data Codemap

## Postgres Entities (AutoMigrate)

Source: `backend/internal/domain/auth/entity.go`, bootstrapped in `backend/cmd/api/main.go`

### `ab_register_user`

- Purpose: pending registrations before email verification.
- Key columns: `id`, `first_name`, `last_name`, `username` (unique), `email` (unique), `password` (bcrypt), `registration_hash` (unique), `created_at`.

### `ab_user`

- Purpose: activated user accounts.
- Key columns: `id`, `first_name`, `last_name`, `username` (unique), `email` (unique), `password`, `active`, `login_count`, `last_login`, `created_on`, `changed_on`.

### `ab_role`

- Purpose: RBAC role catalog.
- Key columns: `id`, `name` (unique).

### `ab_permission`

- Purpose: RBAC action catalog (for example: `can_read`, `can_write`).
- Key columns: `id`, `name` (unique).

### `ab_view_menu`

- Purpose: RBAC resource/menu catalog (for example: `Dashboard`, `Chart`).
- Key columns: `id`, `name` (unique).

### `ab_permission_view`

- Purpose: permission-to-view mapping matrix used by role assignments.
- Key columns: `id`, `permission_id`, `view_menu_id`, unique composite (`permission_id`, `view_menu_id`).

### `ab_permission_view_role`

- Purpose: role-to-permission_view join table used for assignment checks.
- Key columns: `role_id`, `permission_view_id`.

## Redis Key Spaces

- `jwt:blacklist:<jti>`: revoked access-token JTIs.
- `refresh:<token>`: refresh token -> user ID mapping.
- `user_tokens:<userID>`: set of active refresh tokens for logout-all operations.
- `user:<userID>`: cached user context for JWT middleware hydration.
- `rate:login:<ip>`: short-window login attempt throttling counter.
- `failed_login:<username>`: failed login count for lockout policy.
- `lockout:<username>`: active lockout marker with TTL.
- `rbac:*`: RBAC cache namespace invalidated on role changes.

## Data Flow Summary

```
register -> ab_register_user
verify   -> move/activate into ab_user
login    -> read ab_user + write refresh/rate keys
logout   -> write jwt:blacklist + delete refresh session
roles    -> read/write ab_role + invalidate Redis rbac:* namespace
permissions/view-menus -> read/write ab_permission + ab_view_menu + invalidate Redis rbac:* namespace
permission-views -> read/write ab_permission_view, check ab_permission_view_role usage, invalidate Redis rbac:* namespace
```

## Domain Types Used in API

- `RegisterRequest`, `LoginRequest`, `RefreshRequest`, `LogoutRequest`
- `UserContext` (middleware-injected actor)
- `Role`, `UpsertRoleRequest`, `RoleListItem`
- `Permission`, `ViewMenu`, `PermissionView`
- `UpsertPermissionRequest`, `UpsertViewMenuRequest`, `CreatePermissionViewRequest`

## Extended Docs

- Service-level DB docs: `docs/db/`
- Backend repository map: `docs/CODEMAPS/backend.md`
