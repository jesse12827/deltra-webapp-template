# deploy/

Files that are installed on a server, and nothing else (Web-App Development Stack v2.0, Section 6).

| File | Installed as |
|---|---|
| `webapp.service` | `/etc/systemd/system/webapp.service` |
| `webapp.env.example` | `/etc/webapp/webapp.env`, after replacing the example values |
| `nginx-webapp.conf` | `/etc/nginx/sites-available/webapp` |

How and when to install them is in [`docs/operations/runbook.md`](../docs/operations/runbook.md).
Only the Approver deploys. Agents prepare these files and never install them.
