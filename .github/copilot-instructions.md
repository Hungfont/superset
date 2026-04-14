# Agent Orchestration

## Source of Truth

This file is the single source of truth for agent orchestration in this repository.

If codebase context is needed before planning or implementation, use docs/CODEMAPS/** as the source of truth for project structure and flow discovery.

Agent definitions are located in .github/agents/.

Codebase context discovery source of truth: docs/CODEMAPS/**
db schema discovery: docs/db/db-overview.md
## Available Agents

| Agent | Purpose | When to Use |
|-------|---------|-------------|
| planner | Implementation planning | Complex features, refactoring |
| architect | System design | Architectural decisions |
| tdd-guide | Test-driven development | New features, bug fixes |
| code-reviewer | Code review | After writing code |
| security-reviewer | Security review | Auth, input handling, APIs, persistence, sensitive data |
| go-reviewer | Go code review | Any Go code changes |
| typescript-reviewer | Typescript code review | frontend projects |
| integrate-reviewer | integrate frontend and backend code review | frontend and backend projects |
| doc-updater | Documentation | Updating docs |
| loop-operator | Autonomous loop operation | Running and monitoring autonomous loops |
| harness-optimizer | Harness tuning | Reliability, cost, and throughput tuning |

## Immediate Agent Usage

No user prompt needed:
1. Complex feature requests - Use **planner** agent
2. Bug fix or new feature - Use **tdd-guide** agent
3. Architectural decision - Use **architect** agent
4. Code just written/modified - Use **code-reviewer** agent
5. Security-sensitive code - Use **security-reviewer** agent
6. Harness reliability/cost tuning - Use **harness-optimizer** agent
7. Loop execution monitoring/stall control - Use **loop-operator** agent
8. Documentation update - Use **doc-updater** agent

## Requirement Delivery Flow

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