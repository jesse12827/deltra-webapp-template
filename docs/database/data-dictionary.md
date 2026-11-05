# Data dictionary

Schema: `webapp`. Update this file in the same Pull Request as any migration that changes the schema
(Web-App Development Stack, Section 9.6).

## users

Application users. Microsoft Entra ID establishes identity; this table decides access and role.

| Column | Type | Null | Meaning |
|---|---|---|---|
| id | bigint identity | no | Primary key |
| email | text | no | Lower-case work email, for contact and display; unique. Never used to match an Entra ID sign-in |
| display_name | text | no | Name shown in the interface (1–200 characters) |
| role | text | no | `admin` or `tech` |
| active | boolean | no | Inactive users cannot sign in |
| created_at, updated_at | timestamptz | no | Provenance |
| entra_tid | uuid | yes | Entra ID tenant ID (`tid` claim) of the user's account |
| entra_oid | uuid | yes | Entra ID object ID (`oid` claim) of the user's account |

`entra_tid` and `entra_oid` are both set or both empty, and each pair belongs to at most one user.
Entra ID sign-in matches a user on this pair only, never on email (Stack Section 10.1).
A user with both empty, such as a synthetic development user, cannot sign in through Entra ID.

Privileges for `webapp_app`: SELECT, INSERT, UPDATE.

## sessions

Server-side sessions (Stack Section 10.2).

| Column | Type | Null | Meaning |
|---|---|---|---|
| token_hash | bytea (32 bytes) | no | SHA-256 of the session token; the token itself is never stored |
| user_id | bigint → users | no | Session owner; deleted with the user |
| created_at | timestamptz | no | Session start |
| last_seen_at | timestamptz | no | Last activity, for the idle timeout |
| expires_at | timestamptz | no | Absolute expiry |

Privileges for `webapp_app`: SELECT, INSERT, UPDATE, DELETE.

## audit_events

Append-only audit trail (Stack Section 12).

| Column | Type | Null | Meaning |
|---|---|---|---|
| id | bigint identity | no | Primary key |
| occurred_at | timestamptz | no | When the event happened |
| user_id | bigint → users | yes | Acting user; empty for events before sign-in |
| action | text | no | Dotted verb, e.g. `item.update`, `signin.denied`, `access.denied` |
| record_type | text | no | Kind of record affected, or empty |
| record_id | bigint | yes | Identifier of the affected record |
| changes | jsonb | yes | `{"changes": {field: {"old", "new"}}, "detail": {...}}`; never secrets |
| request_id | text | no | Request identifier, matching the logs |

Privileges for `webapp_app`: SELECT, INSERT only. The application cannot change or delete audit events.

## items (example domain)

| Column | Type | Null | Meaning |
|---|---|---|---|
| id | bigint identity | no | Primary key |
| title | text | no | 1–200 characters |
| description | text | no | Up to 4,000 characters; empty when not given |
| status | text | no | `open` or `closed` |
| due_on | date | yes | Date only |
| version | integer | no | Increments on every update; protects against concurrent edits |
| created_at, created_by, updated_at, updated_by | timestamptz / bigint → users | no | Provenance |

Privileges for `webapp_app`: SELECT, INSERT, UPDATE (no DELETE: items are closed, not removed).
