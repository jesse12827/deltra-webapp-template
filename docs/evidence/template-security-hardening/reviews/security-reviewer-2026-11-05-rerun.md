**Security Review — template-security-hardening (re-run) — PASS WITH FINDINGS**

I read the code and the tests. I did not run the application, the tests or `make check`. The baseline is the Stack at /home/jesse100taylor/development/deltra-standards. There is no SRS. This pass covers the three fixes to the earlier Lows and the six Phase 0 findings, and I re-read `server.go`, `auth_handlers.go`, `identity.go` and `middleware.go` for the whole route and sign-in path. I did not re-open `session.go`, `entra.go`, `config.go`, the migration, the dbgen code, `testdb.go` or the docs in this pass. My judgement of them rests on the earlier report (which read the first four) and on the tests.

| # | Severity | File:line | Requirement | Finding | Suggested fix |
|---|---|---|---|---|---|
| 1 | Low | /home/jesse100taylor/development/deltra-webapp-template/internal/web/auth_handlers.go:78 | Stack 12 (tests) | The `signin.denied` event for an Entra person with no linked user now holds only the tenant and object IDs. No test covers this callback path. The web tests use dev sign-in only, and `entraCallback` needs a live OIDC exchange. The "no email in the audit detail" fix therefore has no regression test. | Factor the denial reason into a small function and unit-test it. Or accept the gap and note it in the decision log. |
| 2 | Low | /home/jesse100taylor/development/deltra-webapp-template/internal/auth/identity.go:35 | Stack 10.1 and 12 | The `LinkUser` audit requirement is documentation only. The function still writes without an audit event, and it replaces an earlier link silently. This is acceptable while it has no caller. Each caller must follow the comment. | When a caller is added, require an audit event (`user.link_entra`, with the old and new IDs) in the same transaction. Optionally make the function take a `pgx.Tx`. |
| 3 | Low | /home/jesse100taylor/development/deltra-webapp-template/internal/web/web_test.go:444 | Stack 10.2 (tests) | `TestServerRefusesDevSigninOutsideDevelopment` covers Acceptance and Production. It does not cover a Development run, where registration must still succeed. The other web tests do sign in through dev mode, so that case is covered indirectly. | None needed. |

**Route inventory:** this is unchanged from the earlier report. `server.go` is the same except that `web.New` now returns an error when `AUTH_MODE=dev` and `Env` is not Development. The `/healthz` handler still returns only "ok" or "database unavailable".

| Route | Requires sign-in | Role |
|---|---|---|
| GET /healthz | No (public) | None |
| GET /static/ | No (public) | None |
| GET /signin | No (public) | None |
| POST /signin/dev | No (public) | None; registered only when AUTH_MODE=dev and APP_ENV=development |
| GET /signin/entra | No (public) | None; registered only when AUTH_MODE is not dev |
| GET /auth/callback | No (public) | None; registered only when AUTH_MODE is not dev |
| POST /signout | Yes | Any role |
| GET / | Yes | Any role |
| GET /items | Yes | Any role |
| GET /items/{id} | Yes | Any role |
| GET /items/new | Yes | Tech |
| POST /items | Yes | Tech |
| GET /items/{id}/edit | Yes | Tech |
| POST /items/{id} | Yes | Tech |
| GET /admin/audit | Yes | Admin |
| / (catch-all) | No | None; renders the not-found page |

**Checked and passed:**
- **Earlier Low 1 (email in `signin.denied`):** fixed. The detail is now "no application user for Entra ID tenant X, object Y". It holds GUIDs only, no email and no secret.
- **Earlier Low 2 (`LinkUser` audit):** addressed as documentation. The function has no caller (see finding 2).
- **Earlier Low 3 (dev sign-in registration):** fixed. `server.go:58-63` re-checks `Env == Development` and returns an error. The test covers Acceptance and Production.
- **Phase 0 finding 1 (new token at sign-in):** fixed and tested. `TestSignInReplacesExistingSession` checks that the new token differs, that the old session row is gone, and that replaying the old cookie redirects to sign-in.
- **Phase 0 finding 2 (dev mode only in development):** fixed. `TestLoadRejectsDevAuthOutsideDevelopment` covers the config check. `TestServerRefusesDevSigninOutsideDevelopment` covers the server check.
- **Phase 0 finding 3 (match on tenant and object ID, not email):** fixed and tested. `FindUser` queries `entra_tid` and `entra_oid` only. `identity_test.go` checks the wrong-email, unlinked-email and same-oid-in-another-tenant cases. `TestEntraIdentityConstraints` checks the database constraints. `TestIdentityFromClaims` rejects tokens missing the tid or oid claim.
- **Phase 0 finding 4 (`signin.success` audited with the session):** fixed and tested. The session and the audit event commit in one transaction. `TestSigninAuditFailureLeavesNoSession` checks that no cookie and no session row remain.
- **Phase 0 finding 5 (audit failure fails the request):** fixed and tested for `signin.denied`, `access.denied` and `signout`, and the signout test checks the session is unchanged. Access stays refused in each case.
- **Phase 0 finding 6 (`Cache-Control: no-store`):** fixed and tested. `TestSignedInPagesAreNotCached` checks `/`, `/items` and `/signin`, and checks the static override. The CSP, nosniff and Referrer-Policy headers are intact, and no inline script or style was added.
- **Route protection and tampering:** every route is protected as listed above, and `requireRole` checks roles on the server. CSRF protection (`http.NewCrossOriginProtection`) is in place and `TestCrossOriginPostIsRejected` covers it. The dev sign-in takes an integer `user_id`, which is reachable only in development.
- **Secrets, audit content and errors:** no tokens, codes or secrets appear in audit details or logs. The request log omits the query string and the body, and the user-facing errors are generic.
- **Administrator protection:** not applicable, because there is no user administration and no SRS.

**Could not check:**
- Runtime behavior, namely the header on every response, the transaction rollbacks and the cookie flow. The tests assert them, but I did not run them.
- The live Entra callback, since no test exercises it.
- Whether the database role has grants on the new `entra_*` columns, and the contents of the migration, `dbgen`, `testdb.go` and the two doc edits. Those are in db-safety and doc-consistency scope.
- Nginx-added HSTS, which is outside the repository.
- Committed secrets in files I did not open.
