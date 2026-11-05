**Document Consistency Review — template-security-hardening (template repo, no SRS; changed docs and code comments checked against CLAUDE.md, README files, docs/, deploy/ and the Stack in /home/jesse100taylor/development/deltra-standards) — PASS WITH FINDINGS**

| # | Severity | Type | Location A | Location B | Finding | Governing document | Owner |
|---|---|---|---|---|---|---|---|
| 1 | Medium | Stale reference | `docs/database/data-dictionary.md` line 22: "Entra ID sign-in matches a user on this pair only (Stack Section 10.1)" | `Web-App_Development_Stack.md` Section 10.1 | Section 10.1 says only that staff sign in through Entra ID over OpenID Connect. It does not say users are matched on tid+oid or that email is never used. The same citation is in `db/migrations/00003_user_entra_identity.sql` lines 1-6 and in `internal/web/auth_handlers.go` line 74. The cited section does not support the rule. This is my reading of the Stack text. | Stack (the rule is not in it today) | Builder (the data dictionary, migration comment and handler comment). Approver (shared standards) if the rule is meant to be a Stack rule. |
| 2 | Low | Stale reference | `internal/web/middleware.go` lines 63-66: "response headers required by ... Section 10.6", followed by a `Cache-Control: no-store` comment | Stack Section 10.6 | The code now sets `Cache-Control: no-store`, but Section 10.6 lists only CSP, nosniff, Referrer-Policy and HSTS. The comment is ambiguous: it could be read as Section 10.6 requiring no-store. The Stack (Section 5) does cover htmx history caching, but that is a different mechanism. | Stack | Builder (comment). Approver (shared standards) if no-store should be a Stack requirement. |
| 3 | Low | Stale reference | `internal/web/auth_handlers.go` lines 86-88: "(Stack Sections 10.2 and 12)" for the one-transaction audit | Stack Sections 10.2 and 12 | Section 10.2 supports replacing the token at sign-in. Section 12 says only that sign-in success and failure are audited. Neither says the audit must be in the same transaction or that a failed audit write must fail the request. The comment cites the Stack for a rule the Stack does not state. The rule comes from the security review, which is not a governing document. | Stack | Builder |

Checked with no finding:
- Runbook Section 5.1 matches migration 00003, `data-dictionary.md`, the `signin.denied` detail text and the `users` columns.
- `deploy/webapp.env.example` is consistent with the new config rules: `APP_ENV=production`, `AUTH_MODE=entra`, and a GUID-shaped placeholder tenant ID. Runbook 1.4 says to set `ENTRA_TENANT_ID`, but it does not say the value must be the tenant GUID (not a domain name), which is now enforced. This is an omission, not a conflict.
- `README.md` line 107 names `db/migrations/00002_items.sql`, which exists. `CLAUDE.md` and `.claude/agents/` contain no stale references to these changes.
- `docs/user-guide.md` is unaffected.
- The decision log is empty. I found no decision-log entry in the documents for the changes.

**Documents reviewed:**
- /home/jesse100taylor/development/deltra-webapp-template/CLAUDE.md
- /home/jesse100taylor/development/deltra-webapp-template/README.md
- /home/jesse100taylor/development/deltra-webapp-template/docs/operations/runbook.md
- /home/jesse100taylor/development/deltra-webapp-template/docs/database/data-dictionary.md
- /home/jesse100taylor/development/deltra-webapp-template/docs/user-guide.md
- /home/jesse100taylor/development/deltra-webapp-template/docs/decisions/decision-log.md
- /home/jesse100taylor/development/deltra-webapp-template/docs/evidence/phase0/reviews/security-reviewer-2026-11-05.md
- /home/jesse100taylor/development/deltra-webapp-template/deploy/webapp.env.example
- /home/jesse100taylor/development/deltra-webapp-template/db/migrations/00003_user_entra_identity.sql
- /home/jesse100taylor/development/deltra-webapp-template/.claude/agents/ (searched with grep, not read in full)
- Code comments in `internal/config/config.go`, `internal/web/auth_handlers.go` and `internal/web/middleware.go`
- /home/jesse100taylor/development/deltra-standards/Web-App_Development_Stack.md, Sections 10.1, 10.2 and 10.6, plus keyword searches

**Could not check:**
- I did not read Stack Section 12 in full, only its search hits.
- I did not read the other README files (db, web, internal, deploy, cmd) or the ADR in full. They matched none of the search terms.
- Neither deploy/webapp.service nor deploy/nginx-webapp.conf was read in full. I did not check whether Nginx sets its own Cache-Control.
- I did not check the AI Policy or Change Management Standard section numbers cited in the changed files. Only the Stack was checked.
