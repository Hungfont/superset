---
description: "Use when adding features, fixing bugs, or refactoring. Enforces test-first development, AAA test structure, descriptive naming, and minimum 80% coverage."
name: "Testing Requirements"
---
# Testing Requirements

- Follow TDD: write failing test first, implement minimal fix, then refactor.
- Cover unit, integration, and critical end-to-end behavior where applicable.
- Keep tests explicit with Arrange, Act, Assert structure.
- Use descriptive test names that express behavior and expected outcome.
- Maintain minimum 80% coverage for changed areas whenever feasible.

# Failure Triage

1. Confirm test isolation and deterministic setup.
2. Validate mocks and fixtures.
3. Fix implementation issues before modifying valid tests.
