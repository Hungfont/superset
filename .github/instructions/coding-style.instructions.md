---
description: "Use when writing or refactoring code. Enforces immutability, clear naming, small focused functions, explicit error handling, and input validation boundaries."
name: "Coding Style"
---
# Coding Style

- Prefer immutable updates; avoid mutating input objects or shared state directly.
- Follow KISS, DRY, and YAGNI; optimize for clarity over cleverness.
- Keep functions focused and short, and keep files cohesive.
- Use early returns to avoid deep nesting.
- Replace magic numbers with named constants.
- Handle errors explicitly and never swallow failures silently.
- Validate all external or user-provided input at boundaries.

# Naming Rules

- Use descriptive camelCase for variables and functions.
- Use PascalCase for types, interfaces, and components.
- Use UPPER_SNAKE_CASE for true constants.
- Use boolean names with is, has, should, or can prefixes.
