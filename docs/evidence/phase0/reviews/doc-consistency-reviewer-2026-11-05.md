**Document Consistency Review — Phase 0 template first build (changed documents `.claude/settings.json`, `README.md`, and `Makefile` where it names documents, checked against the whole set) — PASS WITH FINDINGS**

| # | Severity | Type | Location A | Location B | Finding | Governing document | Owner |
|---|---|---|---|---|---|---|---|
| 1 | Medium | Stale reference | `/home/jesse100taylor/development/deltra-webapp-template/docs/operations/runbook.md` Section 4.2: "Restore test (SRS AT-17 pattern)" | — | The runbook cites an SRS acceptance test ID, but `docs/requirements/` holds only a README and no SRS. The ID cannot be resolved in the template. It will only resolve in an application whose SRS happens to number its restore test AT-17. This depends on interpretation: it may be an intended placeholder. Not High, because no agent file relies on it. | The application's SRS (none present in the template); Stack Section 15 and runbook content | Builder |

**Notes (not findings):**
- `Makefile` `init` ends with "Next: make standards ...; fill in CLAUDE.md and README.md; then make generate && go mod tidy && make check". README Section 3 steps 2 and 3 run `make init`, then `make standards`, then `make check`, with no `make generate` or `go mod tidy`. The Makefile is code, not a document in the set, so I did not rate this as a Conflict. The Approver may want the two sequences aligned. Owner: Builder (README, Makefile).
- Every other cross-reference I checked resolves. These were the Stack sections (2, 3, 4, 6, 7, 8, 9.x, 10.x, 12, 13, 15, 18, 19), Change Management Standard sections (42, 43, 44, 82, 127, 134A–C, 136), and AI Policy 26A.1–26A.9. The Stack version "v2.0" matches the Document Record in `deltra-standards`.
- The `make` targets in `CLAUDE.md` and README (`check`, `generate`, `db-dev`, `synth`, `run`, `build`) all exist in the Makefile. The paths named in the runbook, the folder READMEs, and README Section 3 all exist: `db/scripts/*.sql`, `db/bootstrap`, the `deploy/` files, `nav.html`, and the `items` files.
- The data dictionary's roles, privileges, and cascade behavior match `db/migrations/00001_core.sql`.
- The credential name `entra-client-secret` and the paths `/etc/webapp` and `/opt/webapp` agree between the runbook and `deploy/webapp.service`.
- The four agent definitions cite Stack sections correctly. None copies application content from an SRS.
- The restatements in `CLAUDE.md` (Builder rules, reviewer table) name their sources, so they are sanctioned.

**Documents reviewed:**
- `/home/jesse100taylor/development/deltra-webapp-template/CLAUDE.md`
- `/home/jesse100taylor/development/deltra-webapp-template/README.md`
- `/home/jesse100taylor/development/deltra-webapp-template/Makefile`
- `/home/jesse100taylor/development/deltra-webapp-template/.claude/settings.json`
- `/home/jesse100taylor/development/deltra-webapp-template/.claude/agents/srs-reviewer.md`
- `/home/jesse100taylor/development/deltra-webapp-template/.claude/agents/security-reviewer.md`
- `/home/jesse100taylor/development/deltra-webapp-template/.claude/agents/db-safety-reviewer.md`
- `/home/jesse100taylor/development/deltra-webapp-template/.claude/agents/doc-consistency-reviewer.md` (lines 1–12 and grep only)
- `/home/jesse100taylor/development/deltra-webapp-template/docs/requirements/README.md`
- `/home/jesse100taylor/development/deltra-webapp-template/docs/decisions/decision-log.md`
- `/home/jesse100taylor/development/deltra-webapp-template/docs/adr/0001-start-from-template.md`
- `/home/jesse100taylor/development/deltra-webapp-template/docs/database/data-dictionary.md`
- `/home/jesse100taylor/development/deltra-webapp-template/docs/operations/runbook.md`
- `/home/jesse100taylor/development/deltra-webapp-template/docs/evidence/README.md`
- `/home/jesse100taylor/development/deltra-webapp-template/docs/evidence/TEMPLATE.md`
- `/home/jesse100taylor/development/deltra-webapp-template/docs/user-guide.md`
- `/home/jesse100taylor/development/deltra-webapp-template/db/README.md`, `web/README.md`, `internal/README.md`, `deploy/README.md`, `cmd/README.md`
- Governing documents, read at `/home/jesse100taylor/development/deltra-standards/` (not through a submodule): `Web-App_Development_Stack.md`, the AI Policy (26A), the Change Management Standard (headings and the cited sections), and the Software Architecture Document (headings).

**Could not check:**
- **`docs/standards/` and the SRS:** neither exists in the template, which is by design (README "Working on the template itself"). `CLAUDE.md`, the agent definitions, and the runbook point at `docs/standards/...` paths and at SRS IDs that cannot be resolved here. I checked those references against the sibling `deltra-standards` checkout instead. I cannot confirm that the pinned submodule commit in an application matches v2.0 of the Stack or tag `v2026.11.1`.
- **Section numbers inside the SAD:** I verified only the headings ("Sections 2 and 11"), not their content.
- **Go code and SQL:** I did not review them, per my scope. In the changed Go files I did not check whether comments name stale sections.
- **The Stack's "Migrations: up, down, up" gate (Section 13.2):** I could not confirm it is run by `make check`. The Makefile has no separate target for it, so it may live inside `go test`. That is a code question, so I did not rate it.
