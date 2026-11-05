**Database Safety Review — template-security-hardening — PASS WITH FINDINGS**

| # | Severity | File:line | Requirement or Stack section | Finding | Suggested fix |
|---|---|---|---|---|---|
| 1 | Low | /home/jesse100taylor/development/deltra-webapp-template/db/queries/users.sql:19-22 | Stack 9.1 (provenance columns); naming and tidiness | `SetUserEntraID` updates `users` and sets `updated_at`, but the `users` table has no `updated_by` column (this predates the change, in 00001). The query also overwrites an existing Entra binding without a guard. That is acceptable for a template, but a derived application should add a guard (`WHERE id = $1 AND entra_tid IS NULL`) or record who made the change. Assumption: the application's SRS decides whether re-linking is allowed. | Add a comment on the query that re-linking is deliberate, or add the guard. |

**Checked and passed:**
- Check 1 (reference data): no reference data is involved.
- Check 2 (tables and columns):
  - 00003 adds nullable `uuid` columns `entra_tid` and `entra_oid`.
  - The paired CHECK `users_entra_id_pair` and the UNIQUE pair `users_entra_id_key` are in place.
  - The names are snake_case, and the table still has its `bigint` identity key.
  - Neither query touches id, created_at, or any creator column.
  - No row-version requirement exists, because there is no SRS.
- Check 3 (integrity): the migration adds no cascading deletes from reference data. The existing `sessions` cascade is unchanged.
- Check 4 (transactions):
  - Sign-in (`completeSignin`, auth_handlers.go:99-111) deletes the old session, creates the new one, and inserts the `signin.success` audit event in a single `pgx.BeginFunc`.
  - Sign-out (auth_handlers.go:142-149) deletes the session and inserts its audit event in a single `pgx.BeginFunc`.
  - `Sessions.WithTx` and `Queries.WithTx` bind both to the same transaction.
  - The cookie is set only after commit.
  - `denySignin` writes one audit row only, so a transaction is not needed.
- Check 5 (privileges and migrations):
  - The existing grants in 00001 already cover users: SELECT, INSERT, UPDATE. The new columns need no extra grant.
  - Audit events stay insert-and-read only.
  - Migration 00003 has a working Down section: it drops the two constraints, then the columns, in a valid order.
  - 00001 was not edited.
  - `docs/database/data-dictionary.md` documents both columns, the pairing rule, and the uniqueness rule in the same change.
- Check 7 (data safety):
  - No production host, connection string, or credential was added.
  - `failAudit` (web_test.go:151-175) installs its trigger and function only as administrator, in the test's own disposable database. That database is created by `testdb.Empty` and dropped by `t.Cleanup`.
  - The trigger is in no migration, so it cannot reach the application's schema elsewhere.
  - `NewWithURL` returns the admin URL of that same disposable database only. `TEST_DATABASE_URL` is still required and must be a postgres URL.
  - The config test URL `postgres:///webapp_dev` is a local development database.

**Not applicable to this change:**
- Check 1 (reference data) and Check 6 (import tools): no SRS exists, so there are no reference tables or import requirements.
- Synthetic data generator coverage (part of check 7): `cmd/synthdata` is not in the changed files.

**Could not check:**
- I did not run the up/down/up migration test, sqlc generation consistency (whether `internal/dbgen` matches the queries and migration), or whether the failure-injection tests actually roll back. All of these need a running database.
- I read only the changed query, handler, test, and testdb files, plus migrations 00001 and 00003. I did not read 00002 or the dbgen output, so the check that the generated models match the migration rests on the migration text alone.
