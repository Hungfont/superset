---
description: "Use when reviewing code changes, preparing merge requests, or validating readiness to merge. Enforces severity-based findings, security-first checks, and minimum testing expectations."
name: "Code Review Standards"
---
# Code Review Standards

- Perform code review after any code modification and before commit or merge.
- Check security concerns first, then correctness, maintainability, and performance.
- Classify findings by severity: CRITICAL, HIGH, MEDIUM, LOW.
- Block approval when any CRITICAL issue exists.
- Require tests for new behavior and verify minimum 80% coverage when relevant.
- Prefer explicit references to changed files and concrete remediation steps.

# Review Flow

1. Inspect current diff and understand behavior changes.
2. Identify security risks in auth, input handling, data access, filesystem, crypto, and external calls.
3. Validate code quality: naming, function size, file cohesion, nesting depth, and error handling.
4. Verify tests and coverage expectations.
5. Summarize findings ordered by severity with actionable fixes.
