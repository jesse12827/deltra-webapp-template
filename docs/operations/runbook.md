# Operations runbook — webapp

Performed by the **Approver only**. Agents prepare and update this runbook; they never run it against
production (AI Policy Section 26A.9; Web-App Development Stack v2.0, Section 15).

Placeholders: `vX.Y.Z` is the release version. Commands run on the production server unless stated.

---

## 1. One-time server setup

### 1.1 Packages

```sh
# PostgreSQL 18 from the PGDG repository (Stack Section 4)
sudo apt install -y postgresql-common
sudo /usr/share/postgresql-common/pgdg/apt.postgresql.org.sh
sudo apt install -y postgresql-18 nginx
```

### 1.2 Service account and directories

```sh
sudo useradd --system --home /nonexistent --shell /usr/sbin/nologin webapp
sudo install -d -o root -g root  -m 0755 /opt/webapp/releases
sudo install -d -o root -g webapp -m 0750 /etc/webapp
sudo install -d -o root -g root  -m 0700 /etc/webapp/credentials
```

### 1.3 Database

Copy `db/bootstrap/` from the release to the server, then:

```sh
sudo -u postgres psql -v ON_ERROR_STOP=1 -f db/bootstrap/001_roles.sql
sudo -u postgres createdb -O webapp_owner webapp
sudo -u postgres psql -d webapp -v ON_ERROR_STOP=1 -f db/bootstrap/002_schema.sql
```

Let the `webapp` service account connect as `webapp_app` over the local socket, and nothing else:

```text
# /etc/postgresql/18/main/pg_ident.conf
webapp   webapp   webapp_app

# /etc/postgresql/18/main/pg_hba.conf — add ABOVE the general "local all all peer" line
local   webapp   webapp_app   peer map=webapp
```

```sh
sudo systemctl reload postgresql
```

### 1.4 Configuration and secrets

```sh
sudo install -o root -g webapp -m 0640 deploy/webapp.env.example /etc/webapp/webapp.env
sudo editor /etc/webapp/webapp.env        # set BASE_URL, ENTRA_TENANT_ID, ENTRA_CLIENT_ID

# Entra ID client secret, encrypted for this host. Type the secret, then Ctrl-D.
sudo systemd-creds encrypt --name=entra-client-secret - /etc/webapp/credentials/entra-client-secret.cred
```

Entra ID application registration (Microsoft Entra admin center): a single-tenant web application with
redirect URI `https://<host>/auth/callback`, and a client secret whose expiry is recorded in the
Deltra calendar.

### 1.5 systemd and Nginx

```sh
sudo install -m 0644 deploy/webapp.service /etc/systemd/system/webapp.service
sudo install -m 0644 deploy/nginx-webapp.conf /etc/nginx/sites-available/webapp
sudo editor /etc/nginx/sites-available/webapp   # server_name and certificate paths
sudo ln -s /etc/nginx/sites-available/webapp /etc/nginx/sites-enabled/webapp
sudo nginx -t && sudo systemctl reload nginx
sudo systemctl daemon-reload
sudo systemd-analyze security webapp.service    # review the exposure score
```

---

## 2. Deploying a release

Preconditions (Change Management Standard Sections 82 and 136): the Acceptance Gate and Release Gate
are recorded in the milestone evidence package, and no decision in the released scope is Assumed or Open.

```sh
# 1. Verify the artifact
sha256sum -c webapp.sha256

# 2. Back up the database (Section 4.1) and confirm the file is not empty
sudo -u postgres pg_dump -Fc webapp -f /var/backups/webapp/pre-vX.Y.Z.dump

# 3. Install the release
sudo install -d -m 0755 /opt/webapp/releases/vX.Y.Z
sudo install -m 0755 webapp /opt/webapp/releases/vX.Y.Z/webapp
/opt/webapp/releases/vX.Y.Z/webapp version

# 4. Apply migrations as the owner role
sudo -u postgres env DATABASE_URL='postgres:///webapp?host=/var/run/postgresql' DATABASE_ROLE=webapp_owner \
    /opt/webapp/releases/vX.Y.Z/webapp migrate status
sudo -u postgres env DATABASE_URL='postgres:///webapp?host=/var/run/postgresql' DATABASE_ROLE=webapp_owner \
    /opt/webapp/releases/vX.Y.Z/webapp migrate up

# 5. Switch to the new release and restart
sudo ln -sfn /opt/webapp/releases/vX.Y.Z /opt/webapp/current
sudo systemctl restart webapp

# 6. Smoke test
curl -fsS http://127.0.0.1:8080/healthz
sudo journalctl -u webapp -n 50 --no-pager
```

Then sign in through the browser, open the dashboard and one record, and record the deployment
(version, commit, time, result) in the release record.

---

## 3. Rollback

**Application only** (the release added no migrations):

```sh
sudo ln -sfn /opt/webapp/releases/<previous version> /opt/webapp/current
sudo systemctl restart webapp
curl -fsS http://127.0.0.1:8080/healthz
```

**With migrations:** follow the release's own rollback plan in its evidence package. Roll back one
migration at a time with `migrate down`, using the *new* release's binary (it contains the Down steps),
before switching the link back. If a migration's Down step would lose data, restore from the
pre-deployment backup instead (Section 4.2).

---

## 4. Backup and restore

### 4.1 Daily backup

```sh
sudo install -d -o postgres -g postgres -m 0700 /var/backups/webapp
# /etc/cron.d/webapp-backup
15 1 * * * postgres pg_dump -Fc webapp -f /var/backups/webapp/webapp-$(date +\%F).dump && find /var/backups/webapp -name 'webapp-*.dump' -mtime +30 -delete
```

Copy backups off the server according to Deltra's backup practice.

### 4.2 Restore test — into an isolated database, never over production

If the application's SRS has an acceptance test for restore, cite its ID here.

```sh
sudo -u postgres createdb -O webapp_owner webapp_restore_test
sudo -u postgres pg_restore -d webapp_restore_test /var/backups/webapp/webapp-YYYY-MM-DD.dump
sudo -u postgres psql -d webapp_restore_test -f db/scripts/verify_backup.sql
sudo -u postgres psql -d webapp            -f db/scripts/verify_backup.sql   # compare counts
sudo -u postgres dropdb webapp_restore_test
```

Record the result and the date of the test.

---

## 5. Routine operations

| Task | How |
|---|---|
| View logs | `sudo journalctl -u webapp --since today` |
| Check health | `curl -fsS http://127.0.0.1:8080/healthz` |
| Purge expired sessions manually | `sudo -u postgres psql -d webapp -f db/scripts/purge_expired_sessions.sql` (the application also does this hourly) |
| Rotate the Entra ID client secret | Create a new secret in Entra ID, re-run the `systemd-creds encrypt` step in 1.4, `sudo systemctl restart webapp`, then delete the old secret in Entra ID |
