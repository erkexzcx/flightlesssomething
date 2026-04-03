---
name: agent-team-management
description: 'Manage the custom agent team (.agent.md files). Use when: creating, updating, reviewing, or deleting agent definitions under .github/agents/. Covers model assignment rules, frontmatter conventions, tool selection, and team structure.'
---

# Agent Team Management

## When to Use

- Creating a new custom agent
- Updating an existing agent's frontmatter, tools, or instructions
- Reviewing agents for correctness
- Restructuring the agent team (adding/removing roles, changing delegation)

## Model Assignment (STRICT)

This is a non-negotiable rule — always verify the `model` field matches the agent's role:

| Agent Role | Model |
|---|---|
| **Coordinators and leads** (delegate work, never implement) | `model: Claude Opus 4.6 (copilot)` |
| **Workers** (developers, reviewers, writers, QA, pentesters) | `model: Claude Sonnet 4.6 (copilot)` |

- Coordinators/leads use Opus because they make complex delegation and synthesis decisions.
- Workers use Sonnet — no fallback array, just the single model.

## File Naming Convention (STRICT)

Agent filenames must follow the established `{category}-{specialty}.agent.md` pattern. This is non-negotiable — inconsistent naming breaks discoverability and team structure.

| Pattern | Examples |
|---|---|
| `{role}-{stack}.agent.md` | `dev-go.agent.md`, `dev-vue.agent.md`, `sec-go.agent.md`, `perf-vue.agent.md`, `qa-go.agent.md`, `qa-vue.agent.md` |
| `lead-{domain}.agent.md` | `lead-dev.agent.md`, `lead-docs.agent.md`, `lead-perf.agent.md`, `lead-security.agent.md` |
| `writer-{scope}.agent.md` | `writer-api.agent.md`, `writer-arch.agent.md`, `writer-bench.agent.md`, `writer-readme.agent.md`, `writer-instructions.agent.md`, `writer-agents.agent.md` |
| `{role}.agent.md` (no suffix) | `coordinator.agent.md`, `pentester.agent.md`, `qa.agent.md`, `writer.agent.md` |
| `{category}-{specialty}.agent.md` | `infra-dev.agent.md` |

Rules:
- **Lowercase only**, words separated by hyphens.
- Category prefix groups related agents: `dev-`, `sec-`, `perf-`, `qa-`, `lead-`, `writer-`.
- Solo roles (no subcategory) use a single word: `coordinator`, `pentester`, `qa`, `writer`.
- The `name` frontmatter field uses **title case** for display (e.g., file `dev-go.agent.md` → `name: Go Dev`).
- Never invent new category prefixes without reviewing the existing team structure first.

## Frontmatter Rules

### The `agents` field

- **Never** specify `agents: []`. An empty array means "no agents allowed at all" — this blocks subagent invocation entirely.
- If an agent has no subagent restrictions, **omit the `agents` field entirely** (which means "all agents allowed by default").

### Other fields

- `description` — Must be keyword-rich with "Use when:" trigger phrases for subagent discovery.
- `tools` — Minimal set. Only include what the role actually needs.
- `user-invocable` — Set to `false` for agents that should only be called as subagents.
- `name` — Use title case for display (e.g., "Go Dev", "Security Lead").

## Procedure

1. Read existing agents in `.github/agents/` to understand the current team structure
2. Identify the agent's role: coordinator/lead vs. worker
3. Assign the correct model per the strict rule above
4. Write a keyword-rich description with "Use when:" triggers
5. Define clear scope and constraints in the body
6. Verify: no `agents: []`, correct model, focused role, no duplicate responsibilities
