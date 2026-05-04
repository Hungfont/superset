# Everything Claude Code (ECC) — Agent Instructions

This is a **production-ready AI coding plugin** providing specialized agents, skills, commands, and automated hook workflows for software development. Agents generate queries and answer are concise and not too long.

**Version:** 1.10.0

Behavioral guidelines to reduce common LLM coding mistakes, derived from Andrej Karpathy's observations on LLM coding pitfalls.

Tradeoff: These guidelines bias toward caution over speed. For trivial tasks, use judgment.

## Think Before Coding
Don't assume. Don't hide confusion. Surface tradeoffs.

Before implementing:

State your assumptions explicitly. If uncertain, ask.
If multiple interpretations exist, present them - don't pick silently.
If a simpler approach exists, say so. Push back when warranted.
If something is unclear, stop. Name what's confusing. Ask.
## Simplicity First
Minimum code that solves the problem. Nothing speculative.

No features beyond what was asked.
No abstractions for single-use code.
No "flexibility" or "configurability" that wasn't requested.
No error handling for impossible scenarios.
If you write 200 lines and it could be 50, rewrite it.
Ask yourself: "Would a senior engineer say this is overcomplicated?" If yes, simplify.

## Surgical Changes
Touch only what you must. Clean up only your own mess.

When editing existing code:

Don't "improve" adjacent code, comments, or formatting.
Don't refactor things that aren't broken.
Match existing style, even if you'd do it differently.
If you notice unrelated dead code, mention it - don't delete it.
When your changes create orphans:

Remove imports/variables/functions that YOUR changes made unused.
Don't remove pre-existing dead code unless asked.
The test: Every changed line should trace directly to the user's request.


## Core Principles

1. **Agent-First** — Delegate to specialized agents for domain tasks
2. **Test-Driven** — Write tests before implementation, 80%+ coverage required
3. **Security-First** — Never compromise on security; validate all inputs
4. **Immutability** — Always create new objects, never mutate existing ones
5. **Plan Before Execute** — Plan complex features before writing code

## Available Agents

| Agent | Purpose | When to Use |
|-------|---------|-------------|
| planner | Implementation planning | Complex features, refactoring |
| architect | System design and scalability | Architectural decisions |
| doc-updater | Documentation and codemaps | Updating docs |
| tdd-guide | Test-driven development | New features, bug fixes |
| code-reviewer | Code quality and maintainability | After writing/modifying code |
| go-reviewer | Go code review | Go projects |
| typescript-reviewer | Typescript code review | frontend projects |
| integrate-reviewer | integrate frontend and backend code review | frontend and backend projects |
| loop-operator | Autonomous loop execution | Run loops safely, monitor stalls, intervene |
| harness-optimizer | Harness config tuning | Reliability, cost, throughput |

## Agent Orchestration
Use this sequence when implementing requirements.

### Phase 0: Harness Optimization Gate
- Call **harness-optimizer** before execution.
- Tune model routing, step prompts, eval rules, and context handoff between stages.
- Add quality checks for weak spots (example: ensure TDD tests fail for the right reason before implementation).

### Phase 1: Intake Requirement
- Main agent reads requirement files in docs/requirement/ and current code context.
- Call **planner** to produce an implementation plan by phase and dependency.

### Phase 2: Architecture Gate
- Call **architect** to validate the plan against HLD and sequence docs in docs/diagram/.

### Phase 3: Runtime Loop Setup
- Call **loop-operator** to wrap runtime execution.
- Run the delivery pipeline as a monitored loop with checkpoints and end-of-loop eval.

### Phase 4: TDD Gate
- Call **tdd-guide** to produce RED-GREEN-IMPROVE test planning for backend/frontend scope.
- Run tests and coverage for changed areas.

### Phase 5: Implement
- Main agent executes code changes according to the locked plan.

### Phase 6: Parallel Review
- Run in parallel where independent:
	- **code-reviewer** for quality and regression risk.
	- **security-reviewer** for trust boundaries and vulnerabilities.
	- **go-reviewer** when Go files are modified.

### Phase 7: Docs and Codemap
- Call **doc-updater** to update impacted codemaps, README sections, and sequence documentation.

### Phase 8: Eval and Loop Decision
- Run final validation and close merge-readiness checklist.
- **loop-operator** determines whether to continue the loop or stop.

## Runtime Checkpoints

Enforce checkpoint validation between stages:
- plan checkpoint: requirement clarity and dependency completeness
- tdd checkpoint: tests are meaningful and valid RED state is observed
- code review checkpoint: issue severity/count improves between loops
- docs checkpoint: docs/codemaps are consistent with actual behavior

If repeated failures are detected:
1. **loop-operator** pauses the loop.
2. Call **harness-optimizer** to adjust harness-level config (routing, eval, prompts, context).
3. Resume only after optimizer changes are applied.

## Orchestration Rule

Do not insert **harness-optimizer** or **loop-operator** as normal middle steps inside plan/tdd/review/docs.

Use this structure:

```text
[harness-optimizer]
	↓
optimized pipeline
	↓
[loop-operator]
	↓
plan → tdd → review → docs → eval
	↓
    loop or stop
```

## Parallel Task Execution

ALWAYS use parallel Task execution for independent operations:

```markdown
# GOOD: Parallel execution
Launch 3 agents in parallel:
1. Agent 1: Security analysis of auth module
2. Agent 2: Performance review of cache system
3. Agent 3: Type checking of utilities

# BAD: Sequential when unnecessary
First agent 1, then agent 2, then agent 3
```

## Multi-Perspective Analysis

For complex problems, use split role sub-agents:
- Factual reviewer
- Senior engineer
- Security expert
- Consistency reviewer
- Redundancy checker
- Doc updater

## Security Guidelines

**Before ANY commit:**
- No hardcoded secrets (API keys, passwords, tokens)
- All user inputs validated
- SQL injection prevention (parameterized queries)
- XSS prevention (sanitized HTML)
- CSRF protection enabled
- Authentication/authorization verified
- Rate limiting on all endpoints
- Error messages don't leak sensitive data

**Secret management:** NEVER hardcode secrets. Use environment variables or a secret manager. Validate required secrets at startup. Rotate any exposed secrets immediately.

**If security issue found:** STOP → use security-reviewer agent → fix CRITICAL issues → rotate exposed secrets → review codebase for similar issues.

## Coding Style

**Immutability (CRITICAL):** Always create new objects, never mutate. Return new copies with changes applied.

**File organization:** Many small files over few large ones. 200-400 lines typical, 800 max. Organize by feature/domain, not by type. High cohesion, low coupling.

**Error handling:** Handle errors at every level. Provide user-friendly messages in UI code. Log detailed context server-side. Never silently swallow errors.

**Input validation:** Validate all user input at system boundaries. Use schema-based validation. Fail fast with clear messages. Never trust external data.

**Code quality checklist:**
- Functions small (<50 lines), files focused (<800 lines)
- No deep nesting (>4 levels)
- Proper error handling, no hardcoded values
- Readable, well-named identifiers

## Testing Requirements

**Minimum coverage: 80%**

Test types (all required):
1. **Unit tests** — Individual functions, utilities, components
2. **Integration tests** — API endpoints, database operations
3. **E2E tests** — Critical user flows

**TDD workflow (mandatory):**
1. Write test first (RED) — test should FAIL
2. Write minimal implementation (GREEN) — test should PASS
3. Refactor (IMPROVE) — verify coverage 80%+

Troubleshoot failures: check test isolation → verify mocks → fix implementation (not tests, unless tests are wrong).

## Workflow Surface Policy

- `skills/` is the canonical workflow surface.
- New workflow contributions should land in `skills/` first.
- `commands/` is a legacy slash-entry compatibility surface and should only be added or updated when a shim is still required for migration or cross-harness parity.

<!-- ## Git Workflow

**Commit format:** `<type>: <description>` — Types: feat, fix, refactor, docs, test, chore, perf, ci

**PR workflow:** Analyze full commit history → draft comprehensive summary → include test plan → push with `-u` flag. -->

## Architecture Patterns

**API response format:** Consistent envelope with success indicator, data payload, error message, and pagination metadata.

**Repository pattern:** Encapsulate data access behind standard interface (findAll, findById, create, update, delete). Business logic depends on abstract interface, not storage mechanism.

**Skeleton projects:** Search for battle-tested templates, evaluate with parallel agents (security, extensibility, relevance), clone best match, iterate within proven structure.

## Performance

**Context management:** Avoid last 20% of context window for large refactoring and multi-file features. Lower-sensitivity tasks (single edits, docs, simple fixes) tolerate higher utilization.

**Build troubleshooting:** Use build-error-resolver agent → analyze errors → fix incrementally → verify after each fix.

## Project Structure

```
agents/          — specialized subagents
skills/          — workflow skills and domain knowledge
commands/        — slash commands
hooks/           — Trigger-based automations
rules/           — Always-follow guidelines (common + per-language)
scripts/         — Cross-platform Node.js utilities
mcp-configs/     — MCP server configurations
tests/           — Test suite
backend/         _ Bakend project
frontend/        _ Frontend project
```

`commands/` remains in the repo for compatibility, but the long-term direction is skills-first.

### Success Metrics

- All tests pass with 80%+ coverage
- No security vulnerabilities
- Code is readable and maintainable
- Performance is acceptable
- User requirements are met