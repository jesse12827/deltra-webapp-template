# Decision log

Business decisions made by agents during development (AI Policy Section 26A.4).

When work touches an Open decision, the Builder Agent uses the SRS development default (or the
simplest option consistent with Confirmed and Recommended requirements), records the decision here
with status **Assumed**, and continues. Only the Approver changes a status, normally in a batch at
the Acceptance Gate. Recording an assumption is required; making one silently is prohibited.

Status values: **Assumed** · **Ratified** · **Changed** · **Rejected**

Add new entries at the top. Do not delete entries; a Changed or Rejected entry stays as history.

## Entry format

    ### D-NNN: Short title
    - Date: YYYY-MM-DD
    - Related: SRS requirement or Open-decision ID (e.g. OD-05)
    - Decision: what was decided
    - Alternatives considered:
    - Rationale:
    - Reversibility: easy | moderate | hard — and why
    - Status: Assumed
    - Approver note: (filled in by the Approver at the gate)

## Entries

### D-001: Every caller of LinkUser writes an audit event in the same transaction
- Date: 2026-11-05
- Related: No SRS (template). Security re-review 2026-11-05, finding 2
  (`docs/evidence/template-security-hardening/reviews/security-reviewer-2026-11-05-rerun.md`);
  Stack Section 12 (user and role administration is audited).
- Decision: `auth.LinkUser` decides which Entra ID account can sign in as an application user.
  Any code that calls it must, in the same transaction, write an audit event (for example
  `user.link_entra`) that records the user and the old and new tenant and object IDs. For now the
  rule is documented on `LinkUser` and `SetUserEntraID` only, because no application code calls
  `LinkUser` yet (the tests do). It is to
  be enforced in code, for example by having `LinkUser` take the transaction and write the audit
  event itself, when user administration is built. Until then the Approver links users with the
  SQL in runbook Section 5.1.
- Alternatives considered: make `LinkUser` write the audit event now (rejected: no application code
  calls `LinkUser` yet (the tests do), and no acting user exists to attribute the event to); remove `LinkUser` until it is needed
  (rejected: the tests use it to set up linked users).
- Rationale: directed by the Approver on 2026-11-05. Enforcing the rule belongs with the user
  administration feature that will call it.
- Reversibility: easy. No application code calls `LinkUser` yet (the tests do).
- Status: Ratified
- Approver note: Ratified by Jesse Garcia, CEO, 2026-11-05. Recorded as Assumed in e763536; ratified 2026-11-05.
