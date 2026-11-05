**Security Review — Phase 0 template first build — PASS WITH FINDINGS**

There are no High or Medium findings. There is no SRS to review against, so the application-specific checks are not covered (see "Could not check"). The review used Stack Sections 10 and 12, read from `/home/jesse100taylor/development/deltra-standards/Web-App_Development_Stack.md`. The `docs/standards/` submodule is not checked out in this working tree, so that file is the only copy I could read.

| # | Severity | File:line | Requirement or Stack section | Finding | Suggested fix |
|---|---|---|---|---|---|
| 1 | Low | `/home/jesse100taylor/development/deltra-webapp-template/internal/web/auth_handlers.go:85-101` | Stack 10.2 (token replaced at sign-in) | `completeSignin` issues a new token but never deletes the session the browser already presented. The old session stays valid until it expires. There is no fixation risk, because the new token is fresh. | If the request carries a session cookie, call `Sessions.Delete` on it before `Create`. |
| 2 | Low | `/home/jesse100taylor/development/deltra-webapp-template/internal/config/config.go:106-109` | Stack 10.1, 11 | `AUTH_MODE=dev` is refused only when `APP_ENV=production`. Acceptance with dev mode lets anyone who can reach the host sign in as any listed user, including the administrator, with no authentication. This assumes acceptance is network-reachable or holds anything beyond synthetic data. | Refuse dev mode when `APP_ENV` is not `development`, or require an explicit opt-in. |
| 3 | Low | `/home/jesse100taylor/development/deltra-webapp-template/internal/auth/entra.go:105-119` and `/home/jesse100taylor/development/deltra-webapp-template/internal/web/auth_handlers.go:73` | Stack 10.1 | The application user is matched on the `email` claim, falling back to `preferred_username`. Both can be changed by directory administrators, and `preferred_username` is not an email-verification claim. This assumes a single-tenant issuer, which `NewEntra` does build. | Match on a stable subject (`oid` plus `tid`) after the first email match. Otherwise record this as an accepted risk in the decision log. |
| 4 | Low | `/home/jesse100taylor/development/deltra-webapp-template/internal/web/auth_handlers.go:90-98` | Stack 12 (audit with the change) | `signin.success` is audited after the session row is created, and not in the same transaction. If the audit insert fails, the user sees an error and an unusable session row remains until it expires. | Create the session and the audit event in one transaction, or delete the session when the audit insert fails. |
| 5 | Low | `/home/jesse100taylor/development/deltra-webapp-template/internal/web/middleware.go:120-125` and `/home/jesse100taylor/development/deltra-webapp-template/internal/web/auth_handlers.go:126-128` | Stack 12 | If the audit insert fails, `access.denied` and `signout` events are only logged and the request goes on. The denial itself still happens. | Accept this as intended, or fail closed for authorization denials. |
| 6 | Low | `/home/jesse100taylor/development/deltra-webapp-template/internal/web/server.go:81-88` | Stack 10.6 (hardening) | Signed-in pages carry no `Cache-Control: no-store`. Confidential HTML could be kept by shared caches or the browser back-cache. | Set `Cache-Control: no-store` on every response except `/static/`. |

**Route inventory:**

| Route | Sign-in required | Role |
|---|---|---|
| `GET /healthz` | No | None. It returns only `ok` or `database unavailable`. |
| `GET /static/` | No | None |
| `GET /signin` | No | None |
| `POST /signin/dev` | No | None. Registered only when `AUTH_MODE=dev`. |
| `GET /signin/entra` | No | None. Registered only when `AUTH_MODE=entra`. |
| `GET /auth/callback` | No | None. Registered only when `AUTH_MODE=entra`. |
| `POST /signout` | Yes | Any |
| `GET /{$}` | Yes | Any |
| `GET /items` | Yes | Any |
| `GET /items/{id}` | Yes | Any |
| `GET /items/new` | Yes | tech (admin also passes) |
| `POST /items` | Yes | tech (admin also passes) |
| `GET /items/{id}/edit` | Yes | tech (admin also passes) |
| `POST /items/{id}` | Yes | tech (admin also passes) |
| `GET /admin/audit` | Yes | admin |
| `/` (catch-all) | No | None. It renders a generic not-found page with no data. |

**Checked and passed:**
- **Check 1, route protection:**
  - Every non-public route goes through `requireUser` or `requireRole`, and role checks run on the server.
  - Role denials return 403 and are audited.
  - The health endpoint reveals nothing beyond up or down.
- **Check 2, tampering:**
  - Role, creator, updater and timestamps come from the session and the database, not the form.
  - The `version` field is only an optimistic-lock check.
  - Sort and page size are normalized from an allow-list.
  - All SQL is parameterized and lives in `db/queries` and `internal/dbgen`.
  - `http.CrossOriginProtection` wraps all routes, and `TestCrossOriginPostIsRejected` covers it.
  - No `template.HTML` is used. I found no inline scripts, styles or handlers in the templates.
- **Check 3, sign-in and sessions (everything not listed in findings 1 and 3):**
  - Sign-in is Entra with OIDC, PKCE, state and nonce, and no passwords are stored anywhere.
  - A person with no active user record is denied with a clear message, and the denial is audited.
  - The token is 32 random bytes and only its hash is stored (CHECK constraint of 32 bytes).
  - The cookie is `__Host-session` with Secure, HttpOnly and SameSite=Lax.
  - Sign-out deletes the session on the server.
  - Idle and absolute expiry are enforced, and an inactive user's session is deleted at the next lookup.
  - Creation and update are attributed to the session user.
- **Check 5, audit (except finding 4):**
  - Item create and update write their audit event in the same transaction as the change, with old and new values.
  - The `audit_events` grant is SELECT and INSERT only.
  - Audit details and logs hold no tokens, codes, secrets or connection strings.
  - `logRequests` omits query strings and bodies.
- **Check 6, secrets, headers, errors:**
  - Client secrets are read only from systemd credentials. The environment fallback is refused in production.
  - I found no committed credentials; the only password in the repo is the throwaway CI Postgres service.
  - The CSP and the other required headers match Stack 10.6.
  - `serverError` shows a generic message and a request ID only. Panic stacks go to the log.
  - The session cookie and the sign-in flow cookie are scoped as required.

**Could not check:**
- **No SRS:** `docs/requirements/` holds only the placeholder README. I could not check SRS-specific requirements, the Section 1.7 requirement IDs, the SRS's list of audited actions or its sign-in method.
- **Administrator protection (check 4):** there is no user or role administration in this build, so the last-administrator rule has nothing to apply to. It needs review once user administration exists. `Sessions.DeleteForUser` exists, but nothing calls it yet.
- **Record-level access:** any signed-in user can view any item and its audit history. The template has no per-record ownership, and I am assuming that is intended for the example domain.
- **Behavior that needs a running server:**
  - The real Entra flow against a tenant.
  - Nginx and HSTS.
  - The behavior of `spa-shell.js`, which is unchanged.
- **Generated code:** I read the sqlc output for sessions and users only. I did not read the `internal/dbgen` audit, items, models or db files. Also not read: `internal/database/database.go`, the `README.md` and `Makefile` diffs, `db/bootstrap` and `deploy/`.
- **`.claude/settings.json`:** the permissions look appropriately restrictive. Whether they are correct is an Approver matter.
