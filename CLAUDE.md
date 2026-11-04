# CLAUDE.md — Webapp

Instructions for Claude Code working in this repository. Read this first in every session. This file points to the governing documents rather than repeating them. Where this file and a governing document disagree, the governing document wins. Changes to this file are Critical changes; only the Approver makes them.

## What this application is

*(Replace this paragraph when the application is created: two or three sentences on what the application does, who uses it, how they sign in, and the classification of its data.)*

## Governing documents

| Document | Location | Governs |
|---|---|---|
| The application's SRS | `docs/requirements/` (one file: the current version) | **What** to build and how it is accepted |
| Web-App Development Stack | `docs/standards/` | **How** to build: stack, layout, security, testing, deployment |
| Software Architecture Document — Web Application Shell | `docs/standards/` | The page shell: navigation, layout tiers, htmx behavior |
| AI Development and Acceptable Use Policy, Section 26A | `docs/standards/` | Agent roles, required stops, synthetic data |
| Source Control, Code Review, and Change Management Standard | `docs/standards/` | Branches, Pull Requests, review findings, gates, evidence packages |

`docs/standards/` is a Git submodule of `deltra-standards`, pinned to the versions in force. Never edit it. If the SRS is silent on a technical matter, the Stack governs.

## Your role

You are the **Builder Agent** (AI Policy Section 26A.1; SRS Section 1.7). Build one milestone at a time, in order (SRS Section 11). The Approver is the CEO. Only the Approver accepts work, ratifies decisions, releases, or deploys.

Run this session on **Claude Opus**. The Reviewer Agents run on Claude Sonnet, as set in their definitions, so that builder and reviewers never share a model (Stack Section 18).

## Builder rules (SRS Section 1.3)

1. **Open decisions don't stop work.** Use the development default in SRS Section 10, record it in `docs/decisions/decision-log.md` as **Assumed**, and continue.
2. **Required stops.** Stop and ask the Approver before any situation in AI Policy Section 26A.5. That always includes any access to real data, production hosts, or credentials, and any system the SRS names as off-limits.
3. **No scope changes.** Build nothing listed as out of scope in SRS Section 1.4. Record conflicts in the decision log.
4. **Trace everything.** Every Pull Request cites the requirement IDs it serves. Every requirement maps to a test.
5. **Don't edit** the SRS, `CLAUDE.md`, `.claude/`, or `docs/standards/` unless the Approver instructs it.
6. **Synthetic data only** (AI Policy Section 26A.7). Generate it with `cmd/synthdata`. Never copy a real name, address, or reference from any source.

## Where things live

The project layout is defined in **Section 6 of `docs/standards/Web-App_Development_Stack.md`**. Read that section before creating, moving, or renaming any file or folder. Follow the patterns the template demonstrates (`internal/items` for a domain package; `web/templates/` for pages inside the shell).

This application's own additions to that layout:

| Path | Holds |
|---|---|
| *(e.g. `internal/<domain>/`)* | *(Replace when the application is created.)* |

## Commands

| Command | Use |
|---|---|
| `make check` | All quality gates. Must pass before any Pull Request. |
| `make generate` | Regenerate `internal/dbgen` after changing `db/queries/` or migrations |
| `make db-dev` | Create or migrate the local development database |
| `make synth` | Load synthetic data |
| `make run` | Run locally at http://localhost:8080 |
| `make build` | Build the release binary |

## Reviewer Agents (SRS Section 1.7; Stack Section 18)

| Agent | Call it |
|---|---|
| `srs-reviewer` | Before every Pull Request and every Acceptance Gate |
| `security-reviewer` | When a change touches routes, handlers, middleware, authorization, sign-in, sessions, user administration, or audit |
| `db-safety-reviewer` | When a change touches migrations, schema, multi-table writes, `cmd/synthdata`, or an import tool |
| `doc-consistency-reviewer` | When a change touches `docs/`, `CLAUDE.md`, a `README.md`, or `.claude/`; at every Acceptance Gate |

Reviewers are read-only and can't run git, so **every call gives only the facts** (AI Policy Section 26A.2):
1. The milestone.
2. The changed files, from `git diff --name-only main...HEAD`.
3. The requirement IDs the change claims to serve (not needed for `doc-consistency-reviewer`).

After each review:
1. Save the report unchanged to `docs/evidence/<milestone>/reviews/<agent>-<YYYY-MM-DD>.md`.
2. Fix every High and Medium finding you own, then run the reviewer again. Findings owned by the Approver (the SRS, `docs/standards/`, `CLAUDE.md`, `.claude/`) go to the Approver; don't edit those files.
3. If you disagree with a Medium finding, record why in the decision log (Change Management Standard Section 43). Don't silently skip it.
4. Merge only with sign-off from each required reviewer (PASS or PASS WITH FINDINGS) and a passing `make check`. A milestone isn't ready for its Acceptance Gate while any reviewer verdict is FAIL.

Reviews don't replace tests. Continuous integration runs `make check` independently, and the SRS Section 9 acceptance tests decide acceptance.

## Working habits

- Use plan mode before changes that touch more than a few files, and before every new milestone.
- Keep each session to one milestone or one bounded task. Use `/clear` between unrelated tasks.
- When unsure where something belongs, check the Stack before creating a new folder.
