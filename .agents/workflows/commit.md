---
description: Commit and push local changes (with mandatory user review before any git write)
---

# Commit & Push

Commit and push local changes to a remote branch. This workflow enforces the mandatory review gate before any git write operation.

> **CRITICAL**: You MUST follow this workflow for ALL git commit and git push operations, even trivial one-line changes. There are NO exceptions. If you find yourself about to run `git commit` or `git push` outside this workflow, STOP and use this workflow instead.

## When to Use

- Whenever you need to commit and/or push changes
- For ad-hoc commits outside of `/prepare-pr`
- For amending existing commits
- For follow-up commits on an existing branch/PR

## Steps

### 1. Show the diff

Show the user what will be committed:

```bash
git add -A
git diff --cached --stat
```

If the user wants more detail on a specific file, show the full diff for that file:

```bash
git diff --cached -- <file>
```

**Present the diff to the user and STOP. Wait for explicit approval before proceeding.**

### 2. Propose a commit message

Draft a commit message following [Conventional Commits](https://www.conventionalcommits.org/) style:

```
<type>(<scope>): <short summary>

<body — what and why, not how>
```

If amending an existing commit, show the original message and propose the amended version.

**Present the commit message to the user. Do NOT run `git commit` until the user approves.**

### 3. Commit (after approval only)

Only after the user explicitly approves both the diff and the message:

```bash
git commit -m "<approved message>"
```

Or if amending:

```bash
git commit --amend --no-edit  # or with updated message
```

### 4. Push (after approval only)

Ask the user before pushing:

> Ready to push to `<remote>/<branch>`. Shall I proceed?

Only push after explicit confirmation:

```bash
git push origin <branch>
```

If force-pushing (e.g. after amend/rebase), warn the user explicitly:

> ⚠️ This requires a force-push (`--force-with-lease`) since the commit was amended/rebased. Shall I proceed?

## Important Notes

- **NEVER skip the review step** — no matter how small the change
- **NEVER combine commit + push in a single command** without separate approval for each
- If the user says "commit and push", still show the diff first, then push after commit approval
- This workflow is the **minimum required process** — `/prepare-pr` adds more steps on top of this
