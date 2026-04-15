---
description: "Use when implementing features or larger fixes. Enforces plan-first delivery, TDD cycle, post-change code review, and documentation updates."
name: "Development Workflow"
---
# Development Workflow

The Feature Implementation Workflow uses meta-agent orchestration around the delivery pipeline.

Meta-agent model:
- **harness-optimizer** runs before execution to tune prompts, routing, eval checks, and context handoff.
- **loop-operator** wraps runtime execution to run monitored loops, checkpoints, and stop/resume control.

## Feature Implementation Workflow
1. **Harness Optimization (Pre-Flow)**
   - Use **harness-optimizer** before running implementation stages
   - Optimize model routing per stage (example: stronger model for planning, cheaper model for docs)
   - Define eval checks between stages (example: verify RED test fails for correct reason)

2. **Runtime Loop Control**
   - Use **loop-operator** to run full delivery loops
   - Loop shape: `plan -> tdd -> code review -> docs -> eval`
   - Add checkpoint validation after each stage
   - Stop loop when no progress or goals are met

3. **Plan First**
   - Use **planner** agent to create implementation plan
   - Generate planning docs before coding: PRD, architecture, system_design, tech_doc, task_list
   - Identify dependencies and risks
   - Break down into phases

4. **TDD Approach**
   - Use **tdd-guide** agent
   - Write tests first (RED)
   - Implement to pass tests (GREEN)
   - Refactor (IMPROVE)
   - Verify 100%+ coverage

5. **Code Review**
   - Use **code-reviewer** agent immediately after writing code
   - Address CRITICAL and HIGH issues
   - Fix MEDIUM issues when possible

6. **Documentation Update**
   - Use **doc-updater** agent after code is reviewed and stable
   - Run `/update-codemaps` to regenerate `docs/CODEMAPS/*`
   - Update relevant READMEs and guides affected by the change
   - Keep sequence diagrams in `docs/diagram/sequence/*` in sync with new flows

## Checkpoint Policy

- plan checkpoint: requirement clarity and dependency coverage
- tdd checkpoint: tests are meaningful and RED state is valid
- code review checkpoint: repeated issues trend down per loop
- docs checkpoint: docs and codemaps match code behavior

If failure repeats with the same pattern:
1. **loop-operator** pauses execution.
2. Re-run **harness-optimizer** to fix harness-level causes (prompting, eval, routing, context boundaries).
3. Resume loop after config changes.

Do not place **harness-optimizer** or **loop-operator** as normal in-between pipeline steps.