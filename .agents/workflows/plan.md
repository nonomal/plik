---
description: Create an implementation plan for a feature or change (research → design → user approval)
---

# Create an Implementation Plan

A structured planning workflow that gathers full context before proposing changes — producing a reviewable implementation plan for user approval.

## When to Use

- Before starting any non-trivial feature or refactoring
- When the user asks to plan, design, or propose an approach
- Invoked via `/plan`

## Steps

### 1. Understand the request

Clarify the user's intent. Ask follow-up questions if the scope or requirements are ambiguous. Don't start researching until the goal is clear.

### 2. Gather architectural context

// turbo-all

Read AGENTS.md if it exists at the project root.
Read the ARCHITECTURE.md files relevant to the change.

These files are specifically aimed at AI Coding Assistants and contain critical context about:
- Package structure and patterns
- Migration procedures
- Testing strategies and commands
- Design decisions and gotchas

> **Rule**: Never skip this step. Even for "simple" changes, the architecture docs often reveal constraints or patterns that affect the implementation.

### 3. Research the codebase

Study the specific files and patterns relevant to the change:

- Grep for related functions, types, routes, and tests
- Read existing implementations of similar features (follow the existing patterns)
- Check the test files to understand coverage expectations
- Note any existing fixtures, utilities, or helpers that can be reused

### 4. Write the implementation plan

Create an implementation plan artifact with the following structure:

```markdown
# [Goal Description]

Brief description of the problem and what the change accomplishes.

## Design

High-level approach and key design decisions. Explain the "why" not just the "what".

## User Review Required (if applicable)

> [!IMPORTANT]
> Highlight breaking changes, trade-offs, or decisions that need user input.

## Proposed Changes

### [Component Name]

#### [MODIFY/NEW/DELETE] [filename](file:///absolute/path)

What changes and why. Include code snippets for non-obvious changes.

---

## Mandatory Checklist

Every plan MUST include these items — omit only if genuinely not applicable:

- [ ] **linting** Run the necessary linters
- [ ] **Tests**: Add/update unit tests, E2E tests (go/vitest/backend e2e/frontend e2e/...)
- [ ] **User-facing docs** (`docs/`): Update if behavior visible to users changes
- [ ] **Architecture docs**: Update `ARCHITECTURE.md` file(s) if patterns or structures change
- [ ] **AGENTS.md**: Update if dependencies, processes, or logic shift
- [ ] **Migration dumps**: If adding a DB migration, run `make test` + `make test-backends` and commit dump files

## Verification Plan

### Automated Tests
- Exact commands to run

### Manual Verification
- Steps for the user to verify
```

### 5. Cross-check completeness

Before presenting the plan, verify:

- [ ] Does the plan cover ALL files that need changing? (source, tests, docs, config)
- [ ] Does it follow the patterns documented in ARCHITECTURE.md?
- [ ] Are migration dump file requirements addressed (if DB changes)?
- [ ] Is the mandatory checklist filled in?
- [ ] Are there any circular dependencies or import issues?
- [ ] Is the verification plan actionable and specific?

### 6. Present for review

Use `notify_user` with the implementation plan path in `PathsToReview` and `BlockedOnUser: true`.

Do NOT proceed to implementation until the user explicitly approves the plan. If the user requests changes, update the plan and re-submit for review.

## Important Notes

- **Context first, code second**: The quality of the plan depends on understanding existing patterns. Rushing to write the plan without reading ARCHITECTURE.md leads to plans that fight the codebase.
- **Be opinionated**: Recommend a specific approach rather than listing alternatives. If there's a trade-off, state your preference and why.
- **Include the "boring" stuff**: Tests, docs, and architecture updates are not optional extras — they're part of the plan.
- **Scope honestly**: If the change is bigger than it seems, say so. Don't hide complexity.