# Security Policy

## Reporting
If you discover a security issue, please report it privately to the repository owner before opening a public issue.

Include:
- impact summary
- reproduction steps
- affected files or routes
- suggested mitigation if available

## Current Security Posture
The current main branch is still in transition from MVP to hardened baseline.

Known active gaps being addressed by the master plan:
- bootstrap-password setup still leads to a single-admin auth model
- DB-backed sessions and auth audit logging exist, but full database-backed auth/session verification and broader auth evolution are still incomplete
- HTTP hardening is implemented but still needs stronger verification and clearer abuse-control trust assumptions
- no formal migration framework yet

## Deployment Guidance
Until the corrected `W1-W4` and `W5` gates in [docs/MASTER_PLAN.md](docs/MASTER_PLAN.md) are completed:
- do not treat this project as production-ready for open public internet exposure
- prefer private or development-only deployment
- apply your own infrastructure protections if you deploy it externally

## Security Principles For This Repo
- security and data integrity take priority over feature speed
- auth, session, schema, reorder logic, and CI are treated as high-risk areas
- fixes should prefer explicit, testable controls over implicit assumptions
- current login throttling prefers `X-Forwarded-For` / `X-Real-IP`; only trust those headers when your reverse proxy is the sole writer and strips client-supplied values
