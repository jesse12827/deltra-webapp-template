**Security Review — template-security-hardening (final re-run) — PASS WITH FINDINGS**

| # | Severity | File:line | Requirement or Stack section | Finding | Suggested fix |
|---|---|---|---|---|---|
| 1 | Low | /home/jesse100taylor/development/deltra-webapp-template/internal/auth/entra.go:107-110 | Stack 10.8 and 12 (nothing untrusted or sensitive in logs) | The authorization `code` is a public callback value, and it is sent to Entra ID. If the exchange fails, `oauth2` returns a `RetrieveError` that carries Entra's response body. `entraCallback` logs that error at auth_handlers.go:70. The body is Entra's text, not the caller's, so this is not the same flaw as the one you fixed. It is also not a way for the caller to inject text. I did not see Entra echo the code in its error text, but that rests on Entra's behavior, which I can't check from the code. | Optional. Log only a fixed message plus the HTTP status, or the `ErrorCode` from `*oauth2.RetrieveError`. Skip the full body. |

**Fix confirmed.** `Finish` returns the fixed `errEntraReturnedError` when the `error` parameter is present (entra.go:101-105). It does not read `error_description`. `TestFinishDoesNotCopyCallbackErrorText` sets both parameters to a marker string. It checks `errors.Is` against the sentinel and checks that the error text does not contain the marker. The previous Low is closed.

**Other untrusted request values on the sign-in path:**
- `entraCallback` log at auth_handlers.go:70: the other `Finish` errors are fixed strings. The ones that wrap a value are `%w` wraps of library errors or values from the verified ID token. `state` is only compared and never included in an error. The flow-cookie contents are not included either.
- Claim errors: `uuidParam` quotes the `tid` and `oid` claims (identity.go:67). Those claims come from an ID token that has already passed signature, audience and nonce checks. They are not request-controlled.
- `signin.denied` reasons: they are fixed strings, except `unlinkedReason`. That function uses `TenantID` and `ObjectID`, which `parseGUID` has already turned into canonical GUIDs. The email claim is not used. In dev mode, `user_id` is parsed to an integer, and its raw text is never stored.
- Request log (middleware.go:41): it logs method, path, status, duration and request ID. It does not log the query string, the cookies or the headers. The callback's `code` and `state` stay out of the log.
- `serverError` (render.go:106): it logs the error and `r.URL.Path`. The path is a request value, but `slog` structured output escapes it. It is not on the sign-in-specific path.
- `audit.Record` slog line (audit.go:58): it logs the action, record type, record ID, user ID and request ID. It logs no detail values, so no secrets or request text.
- `access.denied` audit (middleware.go:128): it stores `r.URL.Path`. Only signed-in users reach it, and it is stored as structured detail. This is acceptable. I note it for completeness.

**Route inventory:** I did not re-inventory routes. `server.go` changed in this branch and I did not re-read it. The routes are as in the earlier final report.

**Checked and passed:**
- Check 3 (sign-in and sessions): Entra sign-in matches on tenant ID and object ID only. A user with no linked record, or an inactive user, is denied with an audited `signin.denied`. Session replacement and the `signin.success` audit happen in one transaction.
- Check 5 (audit): secrets, tokens and codes are not in logs or audit details on this path.
- Check 6 (errors): the user-facing denial and error pages show fixed messages plus a request ID.
- Checks 1, 2 and 4: nothing changed in this fix, and nothing in the files I read contradicts the earlier reports.

**Could not check:**
- Real Entra error text and `RetrieveError` contents, which need a live tenant.
- Whether `Detail` JSON in audit rows is escaped when displayed. No audit viewer is in scope.
- I read only the files bearing on the sign-in-path log question. I did not re-read the migration, the queries, `session.go`, `config.go` or `server.go`, so I rely on the earlier reports for them.
