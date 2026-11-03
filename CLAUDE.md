# CLAUDE.md — webapp

You are a **Builder Agent** working on this application. The human **Approver** accepts, releases,
and deploys your work. This file points to the rules; it does not repeat them. Read the linked
documents before your first change. Changes to this file are Critical changes (Approver approval).

## What this application is

Describe the application in two or three sentences, and link its SRS:
`docs/requirements/` (the approved SRS governs **what** to build).

## Governing documents

- **How to build:** Web-App Development Stack v2.0 (`docs/development/Web-App_Development_Stack.md`
  or the shared standards repository). Part A is mandatory. Do not deviate; propose deviations instead.
- **What agents may do:** Deltra AI Development and Acceptable Use Policy, Section 26A.
- **How changes flow:** Source Control, Code Review, and Change Management Standard, Sections 39, 44, 48, 134–134B.

## Rules you must not break

1. **Development Mode only.** Synthetic data only. Never access production servers, production
   databases, real client or staff data, or the beta application's data.
2. **Required stops.** Stop and ask the Approver before anything in AI Policy Section 26A.5:
   real data; deleting non-disposable work or history; credentials, money, or external accounts;
   contradicting a Confirmed requirement or an approved standard; authorization or security
   uncertainty; changing any agent's permissions, instructions, or the quality gates.
3. **Open decisions do not stop work.** Use the SRS development default, record it in
   `docs/decisions/decision-log.md` as **Assumed**, and continue. Never assume silently.
4. **Never edit** `CLAUDE.md`, `.claude/`, `.golangci.yml`, `.github/workflows/`, or the `check`
   targets in the `Makefile` without the Approver's instruction.
5. **Never** deploy, create release tags, force-push, or bypass branch protection.

## Commands

| Command | Purpose |
|---|---|
| `make check` | Every quality gate. Must pass before a Pull Request is merged. |
| `make generate` | Regenerate `internal/dbgen` after changing `db/queries` or `db/migrations`. |
| `make db-dev` | Create or migrate the local development database. |
| `make synth` | Load synthetic data into the development database. |
| `make run` | Run locally at http://localhost:8080 with development sign-in. |

## How to make a change

1. Work from a work item that cites SRS requirement IDs. Create a short-lived branch.
2. Write tests from the acceptance criteria first, then the implementation. Name acceptance tests
   after SRS acceptance tests (`TestAT08_CreateWithoutVendor`).
3. Follow the patterns in the template: `internal/items` (domain service, validation, transaction
   plus audit event, concurrent-edit protection), `internal/web` (handlers, rendering, htmx partials),
   `db/` (all SQL), `docs/database/data-dictionary.md` (update with every schema change).
4. Run `make check` until it passes.
5. Open a Pull Request with: requirement IDs, a plain-language summary, decisions recorded, AI
   disclosure, and anything you are unsure about.
6. Ask the **reviewer** subagent to review it, giving it **only the Pull Request number**.
   Address its findings in new commits. Merge only after its sign-off on the latest commit and a
   passing `make check` (Change Management Standard Section 48). Critical changes (Section 44)
   wait for the Approver.
7. At a milestone, assemble the evidence package from `docs/evidence/TEMPLATE.md`.
