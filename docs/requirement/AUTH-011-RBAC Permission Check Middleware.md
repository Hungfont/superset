# AUTH-011 — RBAC Permission Check Middleware (Aligned to Current Backend)

**Project:** Auth Service (Go + Gin)  
**Type:** Backend Security Feature  
**Scope:** Align requirement with currently implemented methods, repositories, entities, and route protection model.

---

## 1. Current State Summary (As Implemented)

The backend currently enforces access in two layers:

1. Authentication layer:
- JWT middleware validates access token and sets authenticated user context into Gin context.

2. Authorization layer:
- Admin-only middleware checks whether current user has the `Admin` role.
- Admin routes are protected at group level, not by per-endpoint `(action, resource)` tuple middleware yet.
- I want to modifed that Admin roles will be bypass without verification.
This means the running model is role-gate authorization for admin endpoints, with role/permission management APIs available, but without route-level tuple checks per handler.

---

## 2. Existing Domain Model and Tables

Current auth entities and schema conventions align with Superset FAB-style data model:

- Users: `ab_user`
- Roles: `ab_role`
- User-role mapping: `ab_user_role`
- Permissions: `ab_permission`
- View menus/resources: `ab_view_menu`
- Permission-resource tuples: `ab_permission_view`
- Role-permission assignment: `ab_permission_view_role`
- Pending registration: `ab_register_user`

Logical tuple represented in DB:
- `(permission.name, view_menu.name)` via `ab_permission_view`

Examples:
- `can_read:Dashboard`
- `can_write:Chart`

---

## 3. Existing Repository Contracts (Auth Domain)

Current repository contracts already cover most RBAC persistence needs:

- RoleRepository:
  - admin-role check
  - role CRUD
  - list/replace/add/remove permission_view assignments
  - list/replace user-role assignments

- PermissionRepository:
  - permission CRUD
  - view_menu CRUD
  - permission_view CRUD
  - seed default permission-view tuples

- RoleCacheRepository:
  - `BustRBAC(ctx)` global RBAC cache bust
  - `BustRBACForUser(ctx, userID)` per-user RBAC cache bust

---

## 4. Current Cache and Invalidation Behavior

RBAC-related cache behavior currently implemented:

- Redis key pattern for RBAC invalidation support exists (prefix `rbac:*`).
- Global invalidation scans and deletes keys by prefix.
- User-specific invalidation deletes `rbac:{userID}` key.
- Invalidation is triggered when:
  - role permissions change
  - user roles change
  - user admin CRUD affects role assignment state

Not currently implemented:

- local in-process L1 cache
- Pub/Sub real-time cross-pod invalidation channel
- singleflight anti-thundering-herd for permission fetch path in middleware

---

## 5. Current Route Protection Pattern

Current router wiring pattern:

- Public auth endpoints remain open (`register`, `verify`, `login`, `refresh`, `logout`).
- Protected group uses JWT middleware.
- Admin group uses Admin-role authorization middleware.
- Admin resources under `/api/v1/admin/*` are guarded by role gate, not tuple middleware yet.

---

## 6. Gap to AUTH-011 Target

Target in original AUTH-011 expects per-route tuple permission middleware with multi-layer cache.  
Current implementation is not there yet. Main gaps:

1. Middleware model gap
- Missing `RequirePermission(action, resource)` middleware factory for route-level checks.

2. Decision granularity gap
- Current gate is role-level (`Admin`) at route group.
- Missing endpoint-specific checks such as:
  - `can_read:Dashboard`
  - `can_write:Dataset`
  - `can_delete:Chart`

3. Performance architecture gap
- Missing L1 in-process TTL cache for hot path.
- Missing singleflight collapse on cold misses.
- Missing Pub/Sub invalidation fan-out for multi-pod consistency.

4. Multi-tenant gap
- Current context/repo contracts are user-centric.
- Original requirement targets tenant-aware keys and DB predicates.

---

## 7. Revised AUTH-011 Scope (Implementation That Matches Current Project)

Phase this requirement to match existing architecture first:

### Phase A (Immediate, compatible)

- Introduce `RequirePermission(action, resource)` Gin middleware.
- Resolve user permission tuples from existing role/permission tables.
- Keep JWT context as identity source.
- Apply middleware per admin endpoint progressively.
- Keep existing `AuthorizeAdminRole` only where explicitly needed.

### Phase B (Cache hardening)

- Reuse existing Redis RBAC keyspace.
- Add permission set cache read-through:
  - miss -> DB join -> cache set -> evaluate
- Keep existing `BustRBAC` and `BustRBACForUser` as invalidation hooks.

### Phase C (Scale features)

- Add optional in-process hot cache with short TTL.
- Add singleflight around DB fallback.
- Add Redis Pub/Sub invalidation for multi-instance deployments.

---

## 8. Acceptance Criteria (Aligned With Current Backend)

1. Authorization behavior
- Protected admin endpoints return:
  - 401 when JWT missing/invalid
  - 403 when authenticated but lacking required permission
  - 2xx when permission exists

2. Route mapping
- At least one endpoint in each admin resource area uses tuple middleware:
  - users
  - roles
  - permissions/view-menus/permission-views

3. Repository compatibility
- No handler performs direct RBAC SQL.
- Middleware uses domain repository interfaces/service abstraction.

4. Cache invalidation integrity
- Role-permission mutation busts RBAC cache.
- User-role mutation busts user RBAC cache.

5. Test completeness
- Middleware tests include allow/deny/missing-user/repo-error branches.
- Integration test verifies one end-to-end tuple-protected route.

6. Tuple-readiness
- Middleware accepts or can be extended to accept object extractor function per route.
- Policy declaration supports relation plus object type/identifier mapping.
- Migration plan identifies zero-downtime path from role-permission checks to tuple checks.

---

## 9. Non-Goals (for this requirement version)

- Full ABAC/object-level policy engine
- Cross-service policy federation
- Tenant isolation redesign if tenant model is not yet present in auth domain
- Migration to external policy engine (OPA/Casbin) in this phase
