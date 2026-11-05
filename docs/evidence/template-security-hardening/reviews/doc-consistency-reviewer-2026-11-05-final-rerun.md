**Document Consistency Review — template-security-hardening, re-run after runbook 5.1 change — PASS WITH FINDINGS**

| # | Severity | Type | Location A | Location B | Finding | Governing document | Owner |
|---|---|---|---|---|---|---|---|
| 1 | Low (carried from earlier report, Low 3) | Conflict | `docs/decisions/decision-log.md` D-001: "because nothing calls them" | `docs/operations/runbook.md` 5.1, which now writes `user.link_entra` itself | The wording of D-001 may now be misread as saying that no linking path exists. D-001 does say that "Until then the Approver links users with the SQL in runbook Section 5.1". The two documents are therefore compatible, and this is an interpretation call. Left for the Approver because D-001 is ratified. | D-001 (Approver-ratified) | Approver |
| 2 | Low (carried from earlier report, Low 2) | Conflict | `docs/decisions/decision-log.md`, entry format ("Status: Assumed") and the log header ("Only the Approver changes a status, normally in a batch at the Acceptance Gate") | D-001 "Status: Ratified", created already Ratified | The entry was written as Ratified rather than starting as Assumed. The Approver's note records the ratification, so this looks like a process-record difference, not a content conflict. Left for the Approver. | Decision log header and AI Policy Section 26A.4 | Approver |

**Checks that now pass**
- **D-001 against runbook 5.1:** The Medium finding is closed. The statement inserts the user and writes a `user.link_entra` audit event in one statement, so the two are saved together or not at all. D-001 says the event is "for example `user.link_entra`" and that the Approver links users with this SQL until the enforcing code exists.
- **Old and new IDs:** D-001 asks for "old and new" tenant and object IDs. For a first-time insert there is no old value, so recording only the new values is consistent.
- **`audit_events` columns:** `action`, `record_type`, `record_id` and `changes` exist in migration 00001 and the data dictionary. `user_id` and `request_id` are omitted. `user_id` is nullable and `request_id` defaults to `''`, which fits "no acting user".
- **`changes` shape:** The runbook writes `{"detail": {...}}`. The data dictionary shape is `{"changes": ..., "detail": ...}`, and `internal/audit/audit.go` omits empty parts, so the shape matches. All detail values are strings, matching `map[string]string`.
- **`users` columns:** `entra_tid` and `entra_oid` are uuid columns from migration 00003. `users_entra_id_pair` and `users_entra_id_key` are satisfied by the insert. The role value `tech` and the lowercase email are valid under the `users_role_valid` and `users_email_lowercase` constraints from 00001. The data dictionary describes the same pair rule.
- **Schema name:** `webapp.users` and `webapp.audit_events` match `CREATE SCHEMA webapp` in `db/bootstrap/002_schema.sql`.
- **Last paragraph of 5.1:** "`signin.denied` ... names the tenant ID and object ID" matches `unlinkedReason` in `internal/web/auth_handlers.go`.
- **Stack Section 12:** I did not re-read it for this run. The citation is the same as in D-001, which the previous review accepted.

**Documents reviewed:**
- `/home/jesse100taylor/development/deltra-webapp-template/CLAUDE.md` (from the supplied context)
- `/home/jesse100taylor/development/deltra-webapp-template/docs/operations/runbook.md` (Sections 5 and 5.1)
- `/home/jesse100taylor/development/deltra-webapp-template/docs/decisions/decision-log.md`
- `/home/jesse100taylor/development/deltra-webapp-template/docs/database/data-dictionary.md` (users and `audit_events`)
- `/home/jesse100taylor/development/deltra-webapp-template/db/migrations/00001_core.sql`
- `/home/jesse100taylor/development/deltra-webapp-template/db/migrations/00003_user_entra_identity.sql`
- `/home/jesse100taylor/development/deltra-webapp-template/db/bootstrap/002_schema.sql` (schema name only)
- `/home/jesse100taylor/development/deltra-webapp-template/internal/web/auth_handlers.go` and `/home/jesse100taylor/development/deltra-webapp-template/internal/audit/audit.go` (only to confirm what the documents say)

**Could not check:**
- I did not re-read the rest of the document set (other READMEs, `.claude/agents/`, the other `docs/` files, the evidence reports). This review was scoped to the runbook 5.1 change.
- I did not re-read the governing documents in `/home/jesse100taylor/development/deltra-standards/`, including Stack Section 12. The Stack Section 12 citation and the Stack Section 6 layout were accepted in the earlier reviews of this milestone.
- I did not run the runbook SQL. The checks above compare it against the migrations on paper.
