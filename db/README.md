# db/

All SQL for the application (Web-App Development Stack v2.0, Sections 6 and 9).

| Folder | What it holds | Who runs it |
|---|---|---|
| `bootstrap/` | Creates the roles (`001_roles.sql`) and the schema and database privileges (`002_schema.sql`). Runs before any migration. | Approver, once per server, as the PostgreSQL administrator |
| `migrations/` | Numbered schema changes, applied in order by `webapp migrate up`. Each has `Up` and `Down` sections. | Agents on disposable databases; the Approver in production, as `webapp_owner` |
| `queries/` | The application's SQL, compiled into Go by sqlc (`make generate`) into `internal/dbgen/`. | Compiled into the binary |
| `scripts/` | Operational SQL, such as backup verification. Each script states its purpose, its role, and whether it changes data. | Approver |

`embed.go` includes the migrations and bootstrap scripts in the application binary.

Rules:

- Synthetic data never appears in `migrations/`; it comes from `cmd/synthdata`.
- Table privileges are granted table by table in migrations, never by default.
- Every migration that changes the schema also updates `docs/database/data-dictionary.md` in the same Pull Request.
