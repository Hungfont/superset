---
description: "Use when code touches authentication, user input, persistence, APIs, or sensitive data. Enforces secure defaults, secret hygiene, and incident response steps."
name: "Security Guidelines"
---
# Security Guidelines

- Never hardcode secrets, tokens, or credentials.
- Validate and sanitize all untrusted input.
- Use parameterized queries and avoid string-built SQL.
- Prevent XSS and CSRF in user-facing and state-changing flows.
- Enforce authentication and authorization checks consistently.
- Apply rate limiting on externally reachable endpoints.
- Ensure error messages do not expose sensitive internals.

# Security Response Protocol

1. Stop when a critical vulnerability is discovered.
2. Remediate the issue before continuing feature work.
3. Rotate exposed secrets immediately.
4. Audit adjacent code paths for similar weaknesses.
