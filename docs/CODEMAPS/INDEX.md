<!-- Generated: 2026-04-14 | Files scanned: ~60 | Token estimate: ~300 -->

# Codemaps Index

Full-stack web app — Go auth microservice + React/TypeScript frontend.  
HLD: [docs/diagram/HLD.md](../diagram/HLD.md)

## Maps

| File | Contents |
|------|----------|
| [architecture.md](architecture.md) | System boundaries, service flow, infra |
| [backend.md](backend.md) | Go auth service — routes, layers, DI wiring |
| [frontend.md](frontend.md) | React pages, hooks, stores, API client |
| [data.md](data.md) | DB tables, Redis keys, domain entities |
| [dependencies.md](dependencies.md) | Go modules, bun packages, external services |

## Current Scope

The **Auth Service** is implemented (Phase 1), along with Phase 1 **Database Connection Service** endpoints (create/list/get/update/delete + test connection + schema introspection flows).  
Schema introspection now includes `GET /api/v1/admin/databases/:id/schemas`, `GET /api/v1/admin/databases/:id/tables`, and `GET /api/v1/admin/databases/:id/columns` with Redis-backed metadata cache and force-refresh controls.
Future services tracked in [docs/requirement/](../requirement/).
