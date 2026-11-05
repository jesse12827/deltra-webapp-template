**Security Review: template-security-hardening (final) — PASS WITH FINDINGS**

I read the code and the tests. I did not run the application, the tests or `make check`. The baseline is Stack Sections 10 and 12. There is no SRS. I re-read the sign-in path, the route registration, the middleware, sessions, config, the Entra code, the migration, the queries and the new test.

| # | Severity | File:line | Requirement or Stack section | Finding | Suggested fix |
|---|---|---|---|---|---|
| 1 | Low | /home/jesse100taylor/development/deltra-webapp-template/internal/auth/entra.go:97-98 | Stack 12 (no unneeded data in logs) | `Finish` copies the `error` query parameter into the error message with `%s`. `entraCallback` logs that message at `auth_handlers.go:70`. The callback route is public, so anyone can put a long string into the application log. `slog` quotes it, so log injection is not possible. This is a hardening point, not a defect. | Log only a fixed message, or cut the value to a short length and allow only the OAuth error-code characters. |

**Earlier Lows from the re-run report**
- **Low 1 (email in `signin.denied`):** fixed. `unlinkedReason` (`auth_handlers.go:91`) builds the reason from the tenant ID and object ID only. `TestUnlinkedReasonNamesIDsNotEmail` checks that both IDs are present and that the email and its domain are absent. Both IDs come from a verified token and are normalised to GUIDs by `parseGUID`, so the audit detail cannot hold arbitrary text.
- **Low 2 (`LinkUser` audit):** D-001 records the rule. `LinkUser` and `SetUserEntraID` have no production caller. The only callers are in tests, so there is no unaudited link today.
- **Low 3:** no action needed.

**Route inventory:** unchanged since the re-run, and re-confirmed in `server.go`.

| Route | Requires sign-in | Role |
|---|---|---|
| GET /healthz | No (public) | None; returns only "ok" or "database unavailable" |
| GET /static/ | No (public) | None |
| GET /signin | No (public) | None |
| POST /signin/dev | No (public) | None; registered only when AUTH_MODE=dev and APP_ENV=development, otherwise `web.New` returns an error |
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

**Checked and passed**
- **Check 1, route protection:** every route not on the public list uses `requireUser` or `requireRole`. The role check runs on the server and audits denials. It fails closed if the audit write fails.
- **Check 2, tampering:** CSRF protection is in place through `http.NewCrossOriginProtection`. The dev sign-in takes a numeric `user_id` and is reachable only in development. SQL is parameterised and lives in `db/queries`. Role and active state come from the database, not the request. I found no `template.HTML` conversion of user content in the files I read.
- **Check 3, sign-in and sessions:**
  - Users are matched on the tenant ID and object ID only, never on email. `identityFromClaims` rejects a token from another tenant and a token missing either claim.
  - The migration makes the pair all-or-nothing and unique.
  - An unlinked or inactive person is denied with a clear message, and the denial is audited.
  - Sessions store token hashes only, and the cookie uses the `__Host-` prefix, Secure, HttpOnly, Path=/ and SameSite=Lax.
  - `completeSignin` removes the old session, creates a new one and writes the audit event in one transaction.
  - Lookup enforces idle expiry, absolute expiry and the active flag.
  - Sign-out deletes the session on the server inside a transaction with its audit event.
  - The OIDC flow uses state, nonce and PKCE, and the flow cookie is cleared on use.
- **Check 4, administrator protection:** not applicable. There is no user administration and no SRS.
- **Check 5, audit:** `signin.success`, `signin.denied`, `signout` and `access.denied` are recorded. Each one either shares a transaction with the change or fails the request. No tokens, codes or secrets appear in audit details. The request log omits the query string and the body.
- **Check 6, secrets, headers and errors:**
  - A search for hard-coded secrets and connection strings found none outside the evidence reports.
  - The client secret is read from the systemd credentials directory. The environment-variable fallback is refused in production.
  - The CSP, nosniff, Referrer-Policy and `Cache-Control: no-store` headers are intact, and the static route overrides the cache header deliberately.
  - Config refuses `AUTH_MODE=dev` outside development, and the server checks again.
  - Error pages are generic.

**Could not check**
- **Runtime behaviour:** header presence on every response, transaction rollbacks and the cookie flow. The tests assert them but I did not run them.
- **The live Entra callback:** no test exercises it. The token-exchange and verification path is read-only reasoning.
- **Database grants on the new `entra_*` columns and the migration's operational safety:** these are db-safety scope.
- **Files I did not re-open this pass:** `internal/dbgen` (generated), `internal/testdb/testdb.go`, the docs and the other tests. I read the tests only where they bear on the findings above.
- **HSTS:** Nginx adds it, outside the repository.
