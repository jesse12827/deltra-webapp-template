# Requirements

This folder holds **one file: the application's current, approved SRS**. Earlier versions live in Git history.
Supporting sources (user guide, project overview) may go in a `sources/` subfolder.

Every Deltra SRS follows the standard section structure (Web-App Development Stack Section 19), because the
Reviewer Agents find what they check by section: 1.3 builder rules, 1.4 scope, 1.5 source precedence,
1.7 agent roles, 3 business rules and reference data, 4 functional requirements, 5 data requirements
(5.3 synthetic data), 7 nonfunctional requirements, 8 data migration (when needed), 9 acceptance tests,
10 open decisions and development defaults, 11 milestone plan.

Rules (AI Policy Section 26A.4; Change Management Standard Section 134A):

- The SRS changes only at the Approver's instruction. Agents propose changes in
  `docs/decisions/decision-log.md`, never by editing the SRS.
- Decisions the Approver ratifies at each Acceptance Gate are folded into the next SRS version.
