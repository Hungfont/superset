---
description: "Use when designing architecture, APIs, or data access layers. Applies repository pattern, response-envelope consistency, SOLID principles, and dependency injection guidance."
name: "Common Patterns"
---
# Common Patterns

- For new systems, prefer proven project skeletons and evaluate options in parallel.
- Encapsulate persistence behind repository interfaces.
- Keep business logic dependent on abstractions, not storage implementations.
- Standardize API responses with status, data, error, and metadata fields.
- Apply SOLID principles when defining module boundaries.
- Use dependency injection and avoid constructing hard dependencies inside business logic.

# DI Guidance

- Prefer constructor injection for required dependencies.
- Inject interfaces rather than concrete implementations.
- Keep wiring explicit and testable.
