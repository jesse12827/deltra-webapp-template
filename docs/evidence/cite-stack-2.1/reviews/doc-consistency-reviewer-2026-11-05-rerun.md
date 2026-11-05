**Document Consistency Review — milestone "cite Stack 2.1" (re-review after fixes; 40 changed files checked against the whole set) — PASS WITH FINDINGS**

| # | Severity | Type | Location A | Location B | Finding | Governing document | Owner |
|---|---|---|---|---|---|---|---|
| 1 | Low | Stale reference | `/home/jesse100taylor/development/deltra-webapp-template/docs/adr/0001-start-from-template.md:9` ("Web-App Development Stack v2.0, Section 3") | — | This is the only remaining "v2.0" Stack citation. The file is not in the changed set. Section 3 ("Reference template repository") exists in 2.1 and still says what the ADR describes, so only the version is stale. The ADR is the record of a decision made against the standard in force when the application was created. The version may be intentional. That is a question of interpretation. | Stack 2.1 | Builder |

The four findings from the previous report (`docs/evidence/cite-stack-2.1/reviews/doc-consistency-reviewer-2026-11-05.md`) are resolved:
- **README.md:4** now says "v2.1, Part A". This was the previous High.
- **`docs/database/data-dictionary.md:4`, `internal/web/middleware.go:120`, `internal/web/web_test.go:6` and `internal/auth/entra.go:27`** all cite v2.1 now.
- **Remaining citations:** a search of the repository found every other Stack citation at v2.1. That covers the Makefile, `sqlc.yaml`, the workflow, `cmd/`, `db/`, `deploy/`, `web/`, `internal/` and `go.mod`. The `v2.0` strings left in `internal/auth/entra.go:47` (the Entra issuer URL), `web/static/vendor/htmx-2.0.11.min.js` and the Makefile's `GOLANGCI_LINT_VERSION` are not Stack citations.

What I checked and found consistent:
- **Section numbers:** every Section number cited in the changed files exists in Stack 2.1 (3, 4, 5, 6, 7, 9.1–9.6, 10.1–10.8, 11, 12, 13.1, 13.2, 14, 15, 16, 19). I checked this against the heading list of `Web-App_Development_Stack.md`. The previous report also confirmed that 10.1, 10.2 and 10.6 match the claims made about them.
- **Change Management Standard:** Sections 82, 136 and 134C exist, and the runbook and README use them as the headings describe.
- **Decision and file references:**
  - `docs/operations/runbook.md` cites decision D-001 and runbook 5.1. D-001 exists in `docs/decisions/decision-log.md:28` ("Every caller of LinkUser writes an audit event in the same transaction").
  - The runbook's file references exist: `db/bootstrap/001_roles.sql`, `002_schema.sql`, `deploy/webapp.env.example`, `deploy/webapp.service`, `deploy/nginx-webapp.conf`, `db/scripts/verify_backup.sql` and `purge_expired_sessions.sql`.
- **Entra identity wording:** `docs/database/data-dictionary.md`, the runbook (Section 5.1) and the user guide agree that Entra ID sign-in matches a user on tenant ID and object ID, never on email. They also agree on the `signin.denied` event.
- **Make targets:** every `make` command named in `CLAUDE.md` and `README.md` is in the Makefile: `check`, `generate`, `db-dev`, `synth`, `run`, `build`, `tools`, `init` and `standards`.
- **Agent definitions:** the four files cite the Stack only by section number (6, 9, 10, 12, 18), with no version. These sections exist. I found no application content copied from an SRS.

**Documents reviewed:**
- `/home/jesse100taylor/development/deltra-webapp-template/CLAUDE.md`
- `/home/jesse100taylor/development/deltra-webapp-template/README.md`
- `/home/jesse100taylor/development/deltra-webapp-template/docs/database/data-dictionary.md`
- `/home/jesse100taylor/development/deltra-webapp-template/docs/operations/runbook.md`
- `/home/jesse100taylor/development/deltra-webapp-template/docs/user-guide.md`
- `/home/jesse100taylor/development/deltra-webapp-template/docs/adr/0001-start-from-template.md`
- `/home/jesse100taylor/development/deltra-webapp-template/docs/evidence/cite-stack-2.1/reviews/doc-consistency-reviewer-2026-11-05.md`
- `/home/jesse100taylor/development/deltra-webapp-template/docs/decisions/decision-log.md` (D-001 only)
- Citation lines and Stack-reference lines in the four `.claude/agents/*.md` files, by search
- Makefile targets, by search
- `/home/jesse100taylor/development/deltra-standards/Web-App_Development_Stack.md` (Document Record, history and heading list)
- `/home/jesse100taylor/development/deltra-standards/deltra-source-control-code-review-and-change-management-standard.md` (heading search for Sections 82, 134C and 136)
- A search of the whole repository for Stack version citations
- By file list only, not read in full:
  - `.claude/settings.json`
  - the sub-folder `README.md` files
  - `docs/requirements/README.md`
  - `docs/evidence/README.md`
  - `docs/evidence/TEMPLATE.md`

**Could not check:**
- `docs/standards/` is not checked out in this template. I read the standards from `/home/jesse100taylor/development/deltra-standards`, and I can't confirm which commit an application's submodule would pin.
- No SRS exists in the template, so SRS IDs and sections can't be resolved.
- I did not read the agent definitions in full against Stack Section 18, so I make no claim about duplication there.
- I did not diff Stack 2.0 against 2.1. I did read the 2.1 history row. It lists 10.1, 10.2, 10.6 and 12 as the changed sections, and the changed files agree with it on those.
- I did not verify code behaviour against the documents. This review covers only whether paths, commands, versions and section numbers resolve.
