# Contributing

## Working Model
- Follow the wave structure in [docs/MASTER_PLAN.md](docs/MASTER_PLAN.md).
- Do not mix unrelated waves in one change unless the dependency is explicit.
- Security, data integrity, and CI changes are high-risk and should stay narrowly scoped.

## Before You Change Code
- Read the relevant wave and work package first.
- Check current progress in the `Execution Log`.
- Confirm whether your change affects auth, schema, reorder logic, CI, or release flow.

## Local Checks
```powershell
./scripts/verify-go.ps1
```

If your change affects frontend assets, also verify the browser behavior with the tracked smoke tooling and record any manual browser verification you performed if the smoke path is unavailable.

Current local smoke command for the canonical React runtime:
```powershell
npm ci
$env:FLUX_PASSWORD="your-password"
npm run smoke:login
```

## Documentation Rule
- Update docs when behavior, setup, or architecture changes.
- Append progress to [docs/MASTER_PLAN.md](docs/MASTER_PLAN.md); do not overwrite older log entries unless fixing factual mistakes.

## Pull Request Expectations
- Keep changes scoped to one wave or one tightly related package.
- Explain what changed, how it was verified, and what risks remain.
- Do not introduce hidden setup assumptions.

## Current Transition Note
Flux Board is still moving from MVP to enterprise-grade structure. Some cleanup work is intentionally staged by wave; if a dependency is not ready yet, record it rather than improvising around it.
