# deltra-webapp-template

The reference template for Deltra web applications: the working implementation of
**Web-App Development Stack v2.0, Part A**. Every new Deltra web application starts as a copy of
this repository (Stack Section 3).

It is a small but complete application. It has Entra ID sign-in (with development sign-in for
synthetic users), server-side sessions, roles, an append-only audit log, the Deltra application shell
(sidebar navigation for desktop and tablet, a bottom tab bar on phones, htmx navigation), four Reviewer
Agents, and one **example domain**, "items". Items demonstrates every standard pattern: search,
filtering, sorting, and pagination; validated forms with field-level errors; concurrent-edit
protection; status counts on a dashboard; and per-record history.

## Start a new application

```sh
git clone <template repository> procurement && cd procurement
rm -rf .git && git init -b main && git add -A && git commit -m "Start from deltra-webapp-template"
make init NAME=procurement MODULE=github.com/<org>/procurement
make standards STANDARDS_URL=git@github.com:<org>/deltra-standards.git STANDARDS_REF=<tag>
```

Then, as the Approver:

1. Put the approved SRS in `docs/requirements/`.
2. Fill in the "What this application is" and "own additions" sections of `CLAUDE.md`.
3. Commit.

The Builder Agent then replaces the example domain (`internal/items`, `db/migrations/00002_items.sql`,
`db/queries/items.sql`, the item templates, and the Items entry in `web/templates/partials/nav.html`)
with the application's own, guided by the SRS.

When cloning an existing application, fetch the standards too: `git clone --recurse-submodules <url>`.

## Prerequisites (development machine)

- Debian 13 (or any Linux) with **Go 1.27** and **PostgreSQL 18** installed locally
- Your PostgreSQL role is a superuser on the local server (for creating disposable test databases)
- `make tools` installs sqlc, golangci-lint, govulncheck, and gitleaks

## Everyday commands

| Command | Purpose |
|---|---|
| `make check` | Every quality gate: format, vet, lint, vulnerabilities, secrets, generated code, tests |
| `make generate` | Regenerate `internal/dbgen` from `db/` |
| `make db-dev` | Create or migrate the local development database |
| `make synth` | Load synthetic data |
| `make run` | Run at http://localhost:8080 with development sign-in |
| `make build` | Build the release binary and its checksum into `bin/` |

## First build of the template itself

The template was written without network access to the Go module proxy, so its first build also
pins its dependencies and generates code:

```sh
make tools          # then pin the four tool versions in the Makefile
make generate       # creates internal/dbgen
go mod tidy         # adds and pins pgx, goose, go-oidc, and oauth2
make check
```

Fix anything `make check` reports, and commit the result, including `go.sum` and `internal/dbgen`.

## Layout

The folder layout is defined once, in Section 6 of the Web-App Development Stack
(`docs/standards/Web-App_Development_Stack.md` in an application; `deltra-standards` for the template itself).
Each top-level folder also has a short README saying what belongs in it.

Agent instructions are in [`CLAUDE.md`](CLAUDE.md).

**Working on the template itself:** the template does not carry the `docs/standards/` submodule, because
each application attaches its own with `make standards`. When improving the template, start Claude Code
with the standards repository alongside it: `claude --add-dir ../deltra-standards`.
Deployment is in [`docs/operations/runbook.md`](docs/operations/runbook.md) and is performed only by the Approver.
