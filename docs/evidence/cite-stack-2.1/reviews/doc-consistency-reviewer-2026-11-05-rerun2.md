**Document Consistency Review — milestone "cite Stack 2.1" (third review, after version numbers were removed from citations; 41 changed files checked against the whole set) — PASS WITH FINDINGS**

| # | Severity | Type | Location A | Location B | Finding | Governing document | Owner |
|---|---|---|---|---|---|---|---|
| 1 | Low | Stale reference | `/home/jesse100taylor/development/deltra-webapp-template/docs/adr/0001-start-from-template.md:9` ("Web-App Development Stack v2.0, Section 3") | — | The ADR still names Stack v2.0. Section 3 exists in 2.1 and says what the ADR describes, so only the version is stale. The ADR may record the standard in force when the application was created. That is a question of interpretation. This is the same finding as in the previous report. | Stack (Document Record: 2.1) | Builder |
| 2 | Low | Conflict | `/home/jesse100taylor/development/deltra-webapp-template/README.md:4` ("Web-App Development Stack v2.1, Part A") and `/home/jesse100taylor/development/deltra-webapp-template/.github/workflows/check.yml:1` ("Web-App Development Stack v2.1, Section 13.2") | Every other Stack citation in the repository (for example `README.md:72`, `Makefile:2`, and the Go, SQL and deploy files) | The last commit is titled "Cite the Stack without a version number". These two citations still carry a version. The values are not wrong, since 2.1 is in force. They do not follow the no-version convention, and they will go stale when the Stack moves on. Whether the convention applies to a README description or a workflow comment depends on interpretation. | Not stated in any document. The commit message is the only source of the convention. | Builder for `README.md`. Approver for `check.yml` (workflow edits are denied to agents in `.claude/settings.json`). |

**Checked and found consistent:**
- **Section numbers:** every Stack Section number cited in the repository exists in Stack 2.1 and matches how it is used. I checked them against the heading list. That covers 2, 3, 4, 6, 7, 8, 9 (9.1, 9.4, 9.5, 9.6), 10 (10.1 to 10.8), 11, 12, 13 (13.1, 13.2), 14, 15, 16, 18 and 19. Examples: 9.5 is Queries (`sqlc.yaml`), 10.5 is Input and output (`validate.go`), 10.7 is Errors (`render.go`) and 10.8 is Secrets (`config.go`).
- **Agent definitions:** the four files in `.claude/agents/` cite the Stack by section number only, with no version. I found no application content copied from an SRS.
- **`CLAUDE.md`:** it cites Stack Section 18 and Section 6, and both exist. Every `make` command it names is in the Makefile.
- **`README.md`:** every `make` command it names is in the Makefile: `tools`, `generate`, `check`, `db-dev`, `synth`, `run`, `build`, `init` and `standards`. `standards-pin` exists in the Makefile but is not named in `README.md`. That is not an inconsistency.
- **Settings:** `.claude/settings.json` agrees with `CLAUDE.md` on what the Builder may not edit: `CLAUDE.md`, `.claude/`, `docs/standards/`, and the workflow files.
- **Remaining "2.0" strings:** the Entra issuer URL in `internal/auth/entra.go:47`, the `GOLANGCI_LINT_VERSION` value in the Makefile, and the htmx vendor file are not Stack citations.

**Documents reviewed:**
- `/home/jesse100taylor/development/deltra-webapp-template/CLAUDE.md`
- `/home/jesse100taylor/development/deltra-webapp-template/README.md`
- `/home/jesse100taylor/development/deltra-webapp-template/Makefile`
- `/home/jesse100taylor/development/deltra-webapp-template/.claude/settings.json`
- `/home/jesse100taylor/development/deltra-webapp-template/docs/adr/0001-start-from-template.md`
- `/home/jesse100taylor/development/deltra-webapp-template/docs/evidence/cite-stack-2.1/reviews/doc-consistency-reviewer-2026-11-05-rerun.md`
- Stack citations in all four `.claude/agents/*.md` files, in the changed Go, SQL, YAML, service and conf files, and in the runbook, data dictionary, user guide and sub-folder READMEs. I read these by search, not in full.
- `/home/jesse100taylor/development/deltra-standards/Web-App_Development_Stack.md` (heading list)

**Could not check:**
- `docs/standards/` is not checked out in this template. I read the standards from `/home/jesse100taylor/development/deltra-standards`. I can't confirm which commit an application's submodule would pin.
- No SRS exists in the template, so SRS IDs and sections can't be resolved.
- I did not re-read the runbook, data dictionary and user guide in full. The previous report covered their content (Entra identity wording, D-001, file references). This review checked only their Stack citations.
- I did not verify code behaviour against the documents.
