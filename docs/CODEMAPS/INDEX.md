<!-- Generated: 2026-05-05 | Files scanned: ~180 | Token estimate: ~350 -->

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
| [dependencies.md](dependencies.md) | Go modules, npm packages, external services |

## Current Scope

The **Auth Service** (Phase 1) is complete, along with **Database Connection Service** (Phase 1) endpoints.

Additional features implemented:
- **Dataset Service**: virtual/physical datasets, columns, metrics, cache management
- **Query Execution Service**: SQL query execution with RLS injection and result caching
- **RLS (Row-Level Security)**: filter management for data access control

See individual codemaps for detailed route maps and domain entities. Future services tracked in [docs/requirement/](../requirement/).
