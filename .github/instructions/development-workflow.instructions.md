---
description: "Use when implementing features or larger fixes. Enforces plan-first delivery, TDD cycle, post-change code review, and documentation updates."
name: "Development Workflow"
---
# Development Workflow

The Feature Implementation Workflow describes the development pipeline: research, planning, TDD, code review, and then committing to git.

## Feature Implementation Workflow
1. **Plan First**
   - Use **planner** agent to create implementation plan
   - Generate planning docs before coding: PRD, architecture, system_design, tech_doc, task_list
   - Identify dependencies and risks
   - Break down into phases

2. **TDD Approach**
   - Use **tdd-guide** agent
   - Write tests first (RED)
   - Implement to pass tests (GREEN)
   - Refactor (IMPROVE)
   - Verify 80%+ coverage

3. **Code Review**
   - Use **code-reviewer** agent immediately after writing code
   - Address CRITICAL and HIGH issues
   - Fix MEDIUM issues when possible

4. **Documentation Update**
   - Use **doc-updater** agent after code is reviewed and stable
   - Run `/update-codemaps` to regenerate `docs/CODEMAPS/*`
   - Update relevant READMEs and guides affected by the change
   - Keep sequence diagrams in `docs/sequences/*` in sync with new flows