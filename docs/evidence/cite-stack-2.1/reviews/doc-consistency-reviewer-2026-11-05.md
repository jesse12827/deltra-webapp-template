**Document Consistency Review — milestone "cite Stack 2.1" (11 changed files checked against the whole set) — FAIL**

| # | Severity | Type | Location A | Location B | Finding | Governing document | Owner |
|---|---|---|---|---|---|---|---|
| 1 | High | Conflict | `/home/jesse100taylor/development/deltra-webapp-template/README.md:4` ("**Web-App Development Stack v2.0, Part A**") | `/home/jesse100taylor/development/deltra-standards/Web-App_Development_Stack.md` Document Record ("Version 2.1"); `internal/web/middleware.go:64` and `internal/auth/identity.go:17`, which cite "v2.1" | The README names the version the template follows as v2.0. The standards in force are 2.1, and the milestone's own files now cite v2.1. The README is the document a person reads to learn which version the template implements. | Stack 2.1 (tag v2026.11.2) | Builder |
| 2 | Medium | Stale reference | `/home/jesse100taylor/development/deltra-webapp-template/docs/database/data-dictionary.md:4` ("Stack v2.0, Section 9.6") | — | This file is in the changed set. It still cites v2.0, while line 22 of the same file and the new migration cite Stack 2.1 Section 10.1. Section 9.6 exists in 2.1, so the section number is correct and only the version is stale. | Stack 2.1 | Builder |
| 3 | Medium | Stale reference | `/home/jesse100taylor/development/deltra-webapp-template/internal/web/middleware.go:120`; `internal/web/web_test.go:6` | `internal/web/middleware.go:64` ("v2.1") | Both files are in the changed set. Each still cites "v2.0" for Section 12 or 13.1, and middleware.go cites two versions in one file. The section numbers exist in 2.1. | Stack 2.1 | Builder |
| 4 | Medium | Stale reference | `/home/jesse100taylor/development/deltra-webapp-template/internal/auth/entra.go:27` ("v2.0, Section 10.1") | `internal/auth/identity.go:17` ("v2.1, Section 10.1") | entra.go is in the changed set. It cites Section 10.1 at v2.0, but 10.1 was revised in 2.1 (tid/oid identification, and the audited link). The two files in the same package now cite different versions for the same section. | Stack 2.1 | Builder |
| 5 | Low | Stale reference | v2.0 citations outside the changed set: `Makefile:2`, `sqlc.yaml:2`, `.github/workflows/check.yml:1`, `cmd/webapp/main.go:2,142`, `db/embed.go:4`, `web/embed.go:4`, `db/migrations/00001_core.sql:3`, `db/bootstrap/001_roles.sql:4`, `db/bootstrap/002_schema.sql:5`, `deploy/webapp.service:2`, `deploy/nginx-webapp.conf:2`, `docs/operations/runbook.md:4`, `docs/user-guide.md:4`, `docs/adr/0001-start-from-template.md:9`, and about 15 Go files under `internal/` | — | These all say "v2.0". Each section number cited exists in 2.1. If the milestone's aim is to cite 2.1 everywhere, these are incomplete. If the aim is only to cite 2.1 where 2.1 changed the text, the split is intentional but unexplained. That is a question of interpretation. The README, `docs/operations/runbook.md` and `docs/user-guide.md` are Builder-owned documents; the rest are code comments. | Stack 2.1 | Builder |

Everything else I checked agrees with Stack 2.1.
- **Section 10.1:** the claims in `db/migrations/00003_user_entra_identity.sql:1-6` and `docs/database/data-dictionary.md:18-23` (tid and oid matching, never email) match it.
- **Section 10.2:** the session citations match it.
- **Section 10.6:** `middleware.go:64-68` matches it, including Nginx adding Strict-Transport-Security and static assets being exempt from `no-store`.
- **Section 12:** the citations in `entra.go`, `entra_test.go` and `web_test.go` resolve.
- **Agent definitions and `CLAUDE.md`:** neither states a Stack version. I found no conflict there.

**Documents reviewed:**
- `/home/jesse100taylor/development/deltra-webapp-template/CLAUDE.md`
- `/home/jesse100taylor/development/deltra-webapp-template/README.md`
- `/home/jesse100taylor/development/deltra-webapp-template/docs/database/data-dictionary.md`
- `/home/jesse100taylor/development/deltra-webapp-template/db/migrations/00003_user_entra_identity.sql`
- `/home/jesse100taylor/development/deltra-standards/Web-App_Development_Stack.md` (Sections 10.1 to 10.8 read in full; the heading list checked)
- A search of the whole repository for version and section citations. This covered the changed `internal/auth` and `internal/web` files and the other Go files that cite the Stack.
- By file list only, not read in full:
  - the sub-folder `README.md` files
  - `docs/requirements/README.md`, `docs/operations/runbook.md`, `docs/user-guide.md`, `docs/adr/0001-start-from-template.md`
  - the four `.claude/agents/` definitions and `.claude/settings.json`

**Could not check:**
- `docs/standards/` is not checked out in this template. I read the standards from `/home/jesse100taylor/development/deltra-standards` instead. I can't confirm which commit an application's submodule would pin.
- No SRS exists in the template, so SRS IDs and sections can't be resolved.
- I did not read the agent definitions in full or compare them with the Stack's Section 18, so I make no claim about conflicts or duplication there.
- I did not diff the Stack's 2.0 and 2.1 text to confirm which sections 2.1 changed. I inferred that 10.1 changed from its content.
- I did not verify the `make` targets in `CLAUDE.md` against the Makefile.
