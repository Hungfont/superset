# Common Patterns

## Skeleton Projects

When implementing new functionality:
1. Search for battle-tested skeleton projects
2. Use parallel agents to evaluate options:
   - Security assessment
   - Extensibility analysis
   - Relevance scoring
   - Implementation planning
3. Clone best match as foundation
4. Iterate within proven structure

## Design Patterns

### Repository Pattern

Encapsulate data access behind a consistent interface:
- Define standard operations: findAll, findById, create, update, delete
- Concrete implementations handle storage details (database, API, file, etc.)
- Business logic depends on the abstract interface, not the storage mechanism
- Enables easy swapping of data sources and simplifies testing with mocks

### API Response Format

Use a consistent envelope for all API responses:
- Include a success/status indicator
- Include the data payload (nullable on error)
- Include an error message field (nullable on success)
- Include metadata for paginated responses (total, page, limit)

### SOLID Principles

Apply all five principles consistently:

- **S — Single Responsibility**: Each module/class/function has exactly one reason to change. Split when a unit handles multiple concerns (e.g., business logic + persistence + formatting).
- **O — Open/Closed**: Open for extension, closed for modification. Add behavior by extending (new implementations, decorators, plugins) rather than editing existing code.
- **L — Liskov Substitution**: Subtypes must be substitutable for their base types without altering correctness. Avoid overrides that tighten preconditions or weaken postconditions.
- **I — Interface Segregation**: Prefer narrow, focused interfaces over fat ones. Clients should not depend on methods they don't use — split large interfaces into role-specific ones.
- **D — Dependency Inversion**: High-level modules depend on abstractions, not concretions. Both high-level and low-level modules depend on the same interface; low-level modules implement it.

### Dependency Injection (DI)

Pass dependencies in rather than constructing them internally:

- **Constructor injection** (preferred): declare all required dependencies as constructor parameters — makes them explicit and testable
- **Interface-based**: inject the abstraction, not the concrete type — aligns with the D in SOLID
- **No hidden coupling**: a module must never `new` up its own collaborators or reach into global singletons for dependencies
- **Testing benefit**: swap real implementations for fakes/mocks at the injection site without touching business logic
- **DI containers**: use a container (e.g., tsyringe, inversify, wire) only when manual wiring becomes unwieldy; prefer manual wiring for small services

```
// Pseudocode — constructor injection
WRONG:  Service() { this.repo = new PostgresRepo() }
CORRECT: Service(repo: IRepo) { this.repo = repo }
```