# Agent Orchestration

## Source of Truth

This file is the single source of truth for agent orchestration in this repository.

If codebase context is needed before planning or implementation, use docs/CODEMAPS/** as the source of truth for project structure and flow discovery.

Agent definitions are located in .github/agents/.

Codebase context discovery source of truth: docs/CODEMAPS/**

## Available Agents

| Agent | Purpose | When to Use |
|-------|---------|-------------|
| planner | Implementation planning | Complex features, refactoring |
| architect | System design | Architectural decisions |
| tdd-guide | Test-driven development | New features, bug fixes |
| code-reviewer | Code review | After writing code |
| security-reviewer | Security review | Auth, input handling, APIs, persistence, sensitive data |
| go-reviewer | Go code review | Any Go code changes |
| doc-updater | Documentation | Updating docs |
| loop-operator | Autonomous loop operation | Running and monitoring autonomous loops |
| harness-optimizer | Harness tuning | Reliability, cost, and throughput tuning |

## Immediate Agent Usage

No user prompt needed:
1. Complex feature requests - Use **planner** agent
2. Code just written/modified - Use **code-reviewer** agent
3. Bug fix or new feature - Use **tdd-guide** agent
4. Architectural decision - Use **architect** agent
5. Security-sensitive code - Use **security-reviewer** agent
6. Go code changes - Use **go-reviewer** agent
7. Documentation update - Use **doc-updater** agent

## Requirement Delivery Flow

Use this sequence when implementing requirements.

### Phase 1: Intake Requirement
- Main agent reads requirement files in docs/requirement/ and current code context.
- Call **planner** to produce an implementation plan by phase and dependency.

### Phase 2: Architecture Gate
- Call **architect** to validate the plan against HLD and sequence docs in docs/diagram/.

### Phase 3: TDD Gate
- Call **tdd-guide** to produce RED-GREEN-IMPROVE test planning for backend/frontend scope.
- Run tests and coverage for changed areas.

### Phase 4: Implement
- Main agent executes code changes according to the locked plan.

### Phase 5: Parallel Review
- Run in parallel where independent:
	- **code-reviewer** for quality and regression risk.
	- **security-reviewer** for trust boundaries and vulnerabilities.
	- **go-reviewer** when Go files are modified.

### Phase 6: Docs and Codemap
- Call **doc-updater** to update impacted codemaps, README sections, and sequence documentation.

### Phase 7: Final Verification
- Run final validation and close merge-readiness checklist.

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