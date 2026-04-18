# Security Policy

## Reporting
If you discover a security issue, please report it privately to the repository owner before opening a public issue.

Include:
- impact summary
- reproduction steps
- affected files or routes
- suggested mitigation if available

## Current Security Posture
- Flux Board now ships a hardened single-user baseline with bootstrap-only password seeding, PostgreSQL-backed sessions, auth audit logging, request limits, operator status endpoints, and repo-owned restore guidance.
- The security model is intentionally scoped to one operator per instance rather than shared multi-user collaboration.
- Deployment and release verification should rely on the repo-owned CI, hosted checks, and operator runbooks rather than ad hoc manual steps.

## Deployment Guidance
- Treat the official Docker image and root binary as the supported deployment contracts.
- Use PostgreSQL, HTTPS termination, and a reverse proxy that you control if you expose the app externally.
- Run the repo-owned post-deploy checks in [docs/OPERATIONS_RUNBOOK.md](docs/OPERATIONS_RUNBOOK.md) before calling a new hosted release healthy.

## Security Principles For This Repo
- security and data integrity take priority over feature speed
- auth, session, schema, reorder logic, and CI are treated as high-risk areas
- fixes should prefer explicit, testable controls over implicit assumptions
- current login throttling prefers `X-Forwarded-For` / `X-Real-IP`; only trust those headers when your reverse proxy is the sole writer and strips client-supplied values
