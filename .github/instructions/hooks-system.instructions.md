---
description: "Use when defining or modifying automation hooks and tool-execution policies. Covers PreToolUse, PostToolUse, Stop hooks, and safe permission practices."
name: "Hooks System"
---
# Hooks System

- Use PreToolUse hooks for validation and parameter checks before execution.
- Use PostToolUse hooks for deterministic checks such as formatting or lint verification.
- Use Stop hooks for final verification when ending a session.
- Do not enable unsafe auto-accept behavior for exploratory or uncertain workflows.
- Prefer explicit allowedTools policies over broad permission bypasses.

# Task Tracking Guidance

- Keep todo items granular and action-oriented.
- Use todo tracking for multi-step tasks and progress visibility.
- Update task status in order to surface missing or out-of-order steps.
