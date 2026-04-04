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

All agents use the same model — no exceptions:

| Agent Role | Model |
|---|---|
| **All agents** (maintainer, reviewers, writers) | `model: Claude Sonnet 4.6 (copilot)` |

## File Naming Convention (STRICT)

Agent filenames must follow the established `{category}-{specialty}.agent.md` pattern. This is non-negotiable — inconsistent naming breaks discoverability and team structure.

| Pattern | Examples |
|---|---|
| `maintainer.agent.md` | The single full-stack developer/maintainer agent |
| `{role}-{stack}.agent.md` | `sec-go.agent.md`, `sec-vue.agent.md`, `perf-go.agent.md`, `perf-vue.agent.md` |
| `writer-{scope}.agent.md` | `writer-api.agent.md`, `writer-arch.agent.md`, `writer-bench.agent.md`, `writer-readme.agent.md`, `writer-instructions.agent.md`, `writer-agents.agent.md` |
| `{role}.agent.md` (no suffix) | `pentester.agent.md` |

Rules:
- **Lowercase only**, words separated by hyphens.
- Category prefix groups related agents: `sec-`, `perf-`, `writer-`.
- Solo roles (no subcategory) use a single word: `maintainer`, `pentester`.
- The `name` frontmatter field uses **title case** for display (e.g., file `sec-go.agent.md` → `name: Go Sec`).
- Never invent new category prefixes without reviewing the existing team structure first.

## Frontmatter Rules

### The `agents` field

- **Never** specify `agents: []`. An empty array means "no agents allowed at all" — this blocks subagent invocation entirely.
- If an agent has no subagent restrictions, **omit the `agents` field entirely** (which means "all agents allowed by default").

### Other fields

- `description` — Must be keyword-rich with "Use when:" trigger phrases for subagent discovery.
- `tools` — Minimal set. Only include what the role actually needs.
- `user-invocable` — Only the Repo Maintainer is `true`. All other agents (reviewers, writers, pentester) are `false` — they are subagents only.
- `name` — Use title case for display (e.g., "Go Sec", "Vue Perf").

## Team Structure

The team is flat — no coordinators or leads. One entry point, specialized subagents:

```
Repo Maintainer (user-invocable, implements + orchestrates)
├── Security Reviewers: Go Sec, Vue Sec, Pentester
├── Performance Reviewers: Go Perf, Vue Perf
└── Writers: Writer API, Writer Arch, Writer Bench, Writer Readme, Writer Instructions, Writer Agents
```

- **Repo Maintainer** — Full-stack developer. Does all coding, testing, and validation. Calls reviewers to verify work, then writers to update docs.
- **Reviewers** — Read-only. Find issues and report them. Never modify code.
- **Writers** — Each owns one documentation file. Never modify source code.

## Procedure

1. Read existing agents in `.github/agents/` to understand the current team structure
2. Identify the agent's role: maintainer, reviewer, or writer
3. Assign the model: `Claude Sonnet 4.6 (copilot)` for all agents
4. Write a keyword-rich description with "Use when:" triggers
5. Define clear scope and constraints in the body
6. Verify: no `agents: []`, correct model, focused role, no duplicate responsibilities, only maintainer is user-invocable
