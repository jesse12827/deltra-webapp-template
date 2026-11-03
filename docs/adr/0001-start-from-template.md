# ADR 0001: Start from the Deltra web application template

- Status: Accepted
- Date: (date the application was created from the template)
- Decided by: Approver

## Context

Web-App Development Stack v2.0, Section 3, requires every new Deltra web application to start
as a copy of the reference template repository, so that applications built by agents are consistent.

## Decision

This application was created from `deltra-webapp-template` and renamed with `make init`.
It keeps the template's layout, middleware, session handling, database access, migrations,
test harness, quality gates, and agent files.

## Consequences

- Improvements made to the template are not applied to this application automatically. When the
  template changes, a work item decides whether to adopt the change here.
- Any departure from the template's patterns requires its own ADR.

---

## Template for new ADRs

Copy this section into `NNNN-short-title.md`.

    # ADR NNNN: Title

    - Status: Proposed | Accepted | Superseded by ADR NNNN
    - Date:
    - Decided by: Approver

    ## Context
    What problem or choice prompted this decision.

    ## Decision
    What was decided.

    ## Consequences
    What becomes easier or harder as a result.
