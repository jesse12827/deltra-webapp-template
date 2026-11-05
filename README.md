# deltra-webapp-template

The reference template for Deltra web applications: the working implementation of
**Web-App Development Stack v2.1, Part A**. Every new Deltra web application starts as a copy of
this repository (Stack Section 3).

It is a small but complete application. It has Entra ID sign-in (with development sign-in for
synthetic users), server-side sessions, roles, an append-only audit log, the Deltra application shell
(sidebar navigation for desktop and tablet, a bottom tab bar on phones, htmx navigation), four Reviewer
Agents, and one **example domain**, "items". Items demonstrates every standard pattern: search,
filtering, sorting, and pagination; validated forms with field-level errors; concurrent-edit
protection; status counts on a dashboard; and per-record history.

Read the sections below in order: **1** once per machine, **2** once for the template, **3** for each new application.

## 1. Prerequisites (development machine)

- Debian 13 (or any Linux) with **Go 1.27** and **PostgreSQL 18** installed locally
- Your PostgreSQL role is a superuser on the local server (for creating disposable test databases)
- Go's `bin` folder is on your `PATH`, so the tools `make tools` installs can be found
- A C compiler (`gcc`), needed by `go test -race` in `make check`: `sudo apt install gcc`
- Claude Code is installed: `curl -fsSL https://claude.ai/install.sh | bash` (installs to `~/.local/bin`; if
  `claude` is not found, add `export PATH="$HOME/.local/bin:$PATH"` to `~/.bashrc` and open a new terminal),
  then sign in once with `claude auth login`
- Your Git name and email are set (`git config --global user.name` / `user.email`)

Check:

```sh
go version                                                                      # go1.27 or later
psql --version                                                                  # 18.x
psql -d postgres -tAc "select rolsuper from pg_roles where rolname = current_user"   # t
echo "$PATH" | grep -q "$(go env GOPATH)/bin" && echo "PATH ok"                 # PATH ok
gcc --version                                                                   # prints the gcc version
claude --version                                                                # prints the Claude Code version
```

If the superuser check fails: `sudo -u postgres createuser --superuser $USER && sudo -u postgres createdb $USER`.
If the PATH check prints nothing: add `export PATH="$PATH:$(go env GOPATH)/bin"` to `~/.bashrc` and open a new terminal.

## 2. First build of the template itself (once, before any application)

The template was written without network access to the Go module proxy, so its first build downloads and
pins its dependencies and generates the database code. It is done once. It is already done if
`ls go.sum internal/dbgen` lists both; then skip to Section 3.

Claude Code does the work, on a branch (Change Management Standard Section 134C):

```sh
cd ~/development/deltra-webapp-template
git switch -c phase0-first-build
claude --model opus --add-dir ../deltra-standards
```

Give Claude Code this instruction:

> Do the template's first build: run `make tools`, pin the four tool versions in the Makefile to the
> versions installed, run `make generate`, run `go mod tidy`, then `make check`. Fix whatever
> `make check` reports, keeping to the Web-App Development Stack. Commit the result on this branch,
> including `go.sum` and `internal/dbgen`. Don't push.

When Claude Code has finished, the Approver pushes the branch, opens a Pull Request to `main` on GitHub,
waits for the check to pass, and squash-merges it. Then, on the development machine:

```sh
git switch main && git pull
ls go.sum internal/dbgen     # both listed: the template is ready to copy
```

## 3. Start a new application

Conventions (Web-App Development Stack Section 3): the folder, the GitHub repository, and the `make init`
name are one lowercase name, never tied to a host; `deltra-standards`, this template, and each application
are sibling folders under one parent folder; the layout comes only from copying this template.

The Approver performs these steps. The example uses `procurement` and the parent folder `~/development`.
Section 2 must be done first; `make init` refuses to run until it is. Replace every `<...>` placeholder
with its real value, angle brackets included.

```sh
cd ~/development                                  # holds deltra-standards/ and deltra-webapp-template/

# 1. One copy of the template, with a fresh history (cp creates the procurement folder)
cp -r deltra-webapp-template procurement
cd procurement
rm -rf .git && git init -b main
git add -A && git commit -m "Start from deltra-webapp-template"

# 2. Adapt the copy: rename it, then link the governing documents (deltra-standards must be on GitHub, tagged)
make init NAME=procurement MODULE=github.com/<org>/procurement
make standards STANDARDS_URL=git@github.com:<org>/deltra-standards.git STANDARDS_REF=<tag>
make check                # the renamed copy must still pass every quality gate

# 3. Add the application's own files: its approved SRS, and its CLAUDE.md
#    (the skeleton with "What this application is" and "own additions" filled in)
cp <path>/<Application>_Software_Requirements_Specification_v<n>.md docs/requirements/
editor CLAUDE.md          # nano, vim, ...: fill in the two sections, or copy a prepared CLAUDE.md over the skeleton
git add -A && git commit -m "Retool for <application>: SRS, CLAUDE.md, standards <tag>"
```

Then create the application's GitHub repository (empty: no README or license), push `main`, and turn on
branch protection (Change Management Standard Section 134C). In that repository's settings, add the secret
`STANDARDS_READ_TOKEN`: a fine-grained, read-only GitHub token for `deltra-standards`, which lets continuous
integration fetch the private `docs/standards/` submodule. From here on, every change goes through a branch
and a Pull Request.

The Builder Agent then replaces the example domain (`internal/items`, `db/migrations/00002_items.sql`,
`db/queries/items.sql`, the item templates, and the Items entry in `web/templates/partials/nav.html`)
with the application's own, guided by the SRS.

When cloning an existing application, fetch the standards too: `git clone --recurse-submodules <url>`.

## Everyday commands

| Command | Purpose |
|---|---|
| `make check` | Every quality gate: format, vet, lint, vulnerabilities, secrets, generated code, tests |
| `make generate` | Regenerate `internal/dbgen` from `db/` |
| `make db-dev` | Create or migrate the local development database |
| `make synth` | Load synthetic data |
| `make run` | Run at http://localhost:8080 with development sign-in |
| `make build` | Build the release binary and its checksum into `bin/` |

## Layout

The folder layout is defined once, in Section 6 of the Web-App Development Stack
(`docs/standards/Web-App_Development_Stack.md` in an application; `deltra-standards` for the template itself).
Each top-level folder also has a short README saying what belongs in it.

Agent instructions are in [`CLAUDE.md`](CLAUDE.md).

**Working on the template itself:** the template does not carry the `docs/standards/` submodule, because
each application attaches its own with `make standards`. When improving the template, start Claude Code
with the standards repository alongside it: `claude --add-dir ../deltra-standards`.
Deployment is in [`docs/operations/runbook.md`](docs/operations/runbook.md) and is performed only by the Approver.
