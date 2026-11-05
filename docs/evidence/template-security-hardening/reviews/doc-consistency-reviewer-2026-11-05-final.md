**Document Consistency Review — template-security-hardening (final, before Pull Request) — PASS WITH FINDINGS**

| # | Severity | Type | Location A | Location B | Finding | Governing document | Owner |
|---|---|---|---|---|---|---|---|
| 1 | Medium | Conflict | `docs/decisions/decision-log.md` D-001 Decision: "Until then the Approver links users with the SQL in runbook Section 5.1." | `docs/operations/runbook.md` Section 5.1: the `INSERT INTO webapp.users ... entra_tid, entra_oid` block, which says nothing about an audit event. Stack Section 12 lists "user and role administration" among audited events. | D-001 names the runbook SQL as the interim way to link users. D-001 says linking needs an audit event in the same transaction, but the runbook procedure writes none and does not say to. The runbook is the only document an operator follows, so the rule does not reach the one place where linking happens today. Whether D-001's rule covers manual SQL is not stated, so this depends on interpretation. | Stack Section 12; D-001 as ratified | Builder (runbook). The Approver decides whether the manual path needs an audit step. |
| 2 | Low | Conflict | D-001 `Status: Ratified`, with the Approver note filled in | AI Policy 26A.4: "Every entry begins as **Assumed**." The log's own Entry format says `Status: Assumed`. | The entry was created already Ratified, not recorded as Assumed and then changed by the Approver. It was directed and ratified by the Approver on the same day, which may justify it. The log's rules do not describe that case. Other entry fields match the format and the six items 26A.4 requires. | AI Policy 26A.4 and the decision-log header | Builder (log). The Approver says whether Approver-directed entries may start Ratified. |
| 3 | Low | Conflict | D-001: "Any code that calls it must, in the same transaction, write an audit event"; "nothing calls them." | `internal/auth/identity_test.go:36`, `:70`, `:73` call `auth.LinkUser` with no audit event. | D-001 and the `LinkUser` comment say "nothing calls" the function. Tests do call it, and the Alternatives section admits it ("the tests use it"). The rule is worded for all code, but the "nothing calls them" and "Nothing calls `LinkUser` yet" statements are true only for non-test code. | D-001 | Builder |
| 4 | Low | Conflict | `docs/operations/runbook.md` line 190: the `signin.denied` event "names the object ID they signed in with." | `internal/web/auth_handlers.go:91` `unlinkedReason` returns "...tenant `&lt;tid&gt;`, object `&lt;oid&gt;`". | The runbook is incomplete: the event names the tenant ID and the object ID. | Code, which is current | Builder |

All other checks passed:
- The security re-review citations in D-001, `unlinkedReason` and `FindUser` match the rerun report. Finding 1 is the audit-detail email fix and finding 2 is the `LinkUser` audit requirement. The report path exists.
- The Stack references are correct: Section 10.2 is Sessions and Section 12 is Logging and audit.
- The `LinkUser` comment in `internal/auth/identity.go`, the sqlc comment in `db/queries/users.sql` and the matching comment in `internal/dbgen/users.sql.go` agree with D-001. They use the same event name, `user.link_entra`.
- The `unlinkedReason` comment matches its behavior (tenant and object IDs, no email).

**Documents reviewed:**
- `/home/jesse100taylor/development/deltra-webapp-template/CLAUDE.md`
- `/home/jesse100taylor/development/deltra-webapp-template/docs/decisions/decision-log.md`
- `/home/jesse100taylor/development/deltra-webapp-template/docs/operations/runbook.md` (Sections 5 and 5.1)
- `/home/jesse100taylor/development/deltra-webapp-template/docs/evidence/template-security-hardening/reviews/security-reviewer-2026-11-05-rerun.md`
- Code comments in `internal/auth/identity.go`, `internal/web/auth_handlers.go` and `internal/dbgen/users.sql.go`
- `db/queries/users.sql` (comment only)
- `/home/jesse100taylor/development/deltra-standards/deltra-AI-development-and-acceptable-use-policy.md` (Section 26A.4)
- `/home/jesse100taylor/development/deltra-standards/Web-App_Development_Stack.md` (Sections 10.2 and 12, and the layout references to the decision log)

**Could not check:**
- I did not re-read every other changed file. This review covered only what changed since the previous PASS: D-001 and the `unlinkedReason` comment.
- `docs/database/data-dictionary.md` and the older evidence reports were not re-read.
- `.claude/` and `README.md` files were not re-read, because the changes since the previous PASS do not touch them.
- Tests were not run.
