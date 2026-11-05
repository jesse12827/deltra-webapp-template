**Security Review — template-security-hardening — PASS WITH FINDINGS**

I read the code only. I did not run the application, the tests or `make check`. The `docs/standards` submodule was not consulted, and the baseline is the Stack at `/home/jesse100taylor/development/deltra-standards`. No SRS exists, so I checked only against the Stack and the six prior findings. I reviewed these files: `auth_handlers.go`, `middleware.go`, `server.go`, `identity.go`, `entra.go`, `session.go`, `config.go`, the migration and `users.sql`, and `audit.Record`. I did not open the test files, the generated `dbgen` code, `testdb.go` or the two docs.

| # | Severity | File:line | Requirement | Finding | Suggested fix |
|---|---|---|---|---|---|
| 1 | Low | /home/jesse100taylor/development/deltra-webapp-template/internal/web/auth_handlers.go:77 | Stack 12 (audit content) | The `signin.denied` audit reason includes the email taken from the ID token, which a directory administrator controls. The length is unbounded. This is personal data, but not a secret. The Stack says the email is informational only. | Record only the oid (and tid), or truncate the email. |
| 2 | Low | /home/jesse100taylor/development/deltra-webapp-template/internal/auth/identity.go:32 | Stack 10.1 and 12 | `LinkUser` has no caller or audit event. It is an unaudited write to a sign-in identity link. The application should call it from an audited transaction (for example `user.link_entra`), and nothing here does. This is an assumption about future use. | Document the audit requirement on `LinkUser`, or take a transaction and require an audit event. |
| 3 | Low | /home/jesse100taylor/development/deltra-webapp-template/internal/web/auth_handlers.go:41 | Stack 10.2 (hardening) | `devSignin` accepts any `user_id` with no authentication. It is registered only when `AUTH_MODE=dev`, and `config.Load` allows that mode only when `APP_ENV=development`. The control is sound. The route's registration still depends on `Config` alone, with no second check in `server.go`. | Optionally re-check `Env == Development` at registration (defense in depth). |

**Route inventory:**

| Route | Requires sign-in | Role |
|---|---|---|
| GET /healthz | No (public) | None. It returns only "ok" or "database unavailable". |
| GET /static/ | No (public) | None |
| GET /signin | No (public) | None. It redirects away if already signed in. |
| POST /signin/dev | No (public; registered only when AUTH_MODE=dev) | None |
| GET /signin/entra | No (public; registered only when AUTH_MODE is not dev) | None |
| GET /auth/callback | No (public; registered only when AUTH_MODE is not dev) | None |
| POST /signout | Yes | Any role |
| GET / | Yes | Any role |
| GET /items | Yes | Any role |
| GET /items/{id} | Yes | Any role |
| GET /items/new | Yes | Tech |
| POST /items | Yes | Tech |
| GET /items/{id}/edit | Yes | Tech |
| POST /items/{id} | Yes | Tech |
| GET /admin/audit | Yes | Admin |
| / (catch-all) | No | None. It renders the not-found page. |

**Checked and passed (the six fixes):**
1. Sign-in replaces any existing session. `completeSignin` deletes the cookie's old session, creates a new token and writes the audit event in one transaction. The cookie is set only after commit.
2. `AUTH_MODE=dev` is rejected in `config.Load` unless `APP_ENV=development`. The dev route exists only in dev mode. Entra mode requires the tenant GUID and a client secret. In production the secret comes only from the systemd credential, not from an environment variable.
3. Entra users are matched on `entra_tid` and `entra_oid` only, through a parameterized query (`GetUserByEntraID`). The migration adds the columns, a both-or-neither CHECK, and a UNIQUE constraint. `identityFromClaims` requires GUID claims and rejects a token from another tenant. An Entra person with no linked active user is denied and the denial is audited. I found no email lookup on the sign-in path. `GetUserByEmail` still exists in `users.sql`, but nothing in the web path calls it.
4. `signin.success` is audited in the same transaction as the session create, so a failed audit write leaves no session.
5. Audit write failures now fail the request in `denySignin`, `requireRole` (`access.denied`) and `signout`. Access stays refused in each case. `signout` deletes the session and writes its audit event in one transaction.
6. `securityHeaders` sets `Cache-Control: no-store` on every response. `cacheStatic` overrides it only for `/static/`. The CSP, nosniff and Referrer-Policy headers are intact, and no inline scripts or styles were added.

Other checks, one line each:
- **Route protection:** every route is protected as listed above. The server enforces roles, and `requireRole` checks them server-side.
- **Tampering and CSRF:** state-changing routes sit behind `http.NewCrossOriginProtection`. The SQL I read uses parameters. I saw no `template.HTML` in the changed files.
- **Sessions and cookies:** the session cookie is `__Host-`, Secure, HttpOnly and SameSite=Lax. `Lookup` ends sessions that are inactive, expired or idle. Sessions are stored as token hashes. The Entra flow uses state, nonce and PKCE, with a `__Host-` cookie.
- **Audit content:** I saw no tokens, codes or secrets in the audit details. User-facing denial messages are generic.
- **Secrets and errors:** I found no committed credentials in the files I read. The error pages are generic, and the panic stack goes to the log only.
- **Administrator protection:** no last-administrator logic is in scope, because there is no user administration and no SRS.

**Could not check:**
- Runtime behavior (header presence on every response, transaction rollback on audit failure, the cookie flow). These need a running server and database. I did not read `web_test.go`, `entra_test.go`, `identity_test.go` or `config_test.go`, so I can't say whether they cover the six fixes.
- The generated `internal/dbgen` code, `internal/testdb/testdb.go`, and the data-dictionary and runbook edits, which I did not read.
- Whether the database role has privileges on the new `entra_*` columns. That depends on the grants, which I did not inspect.
- Nginx-added HSTS, which is outside the repository.
- The claim that no secret has been committed in files I did not open.
