# Flux Board

Flux Board is a single-user self-hosted task board built with Go, PostgreSQL, and React.
It is designed to be easy to fork, deploy, and operate as one board for one operator.

The intended product flow is simple:
- fork the repo
- deploy it locally or on a cloud host
- open the URL
- finish setup or sign in
- use one board as one operator

## Project Snapshot
- current maturity: strong single-user self-hosted beta
- canonical runtime: React app on `/`
- rollback shell: legacy HTML runtime on `/legacy/`

## Start Here
- deployment contracts and hosting notes: [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)
- architecture summary: [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md)
- operator checks and troubleshooting: [docs/OPERATIONS_RUNBOOK.md](docs/OPERATIONS_RUNBOOK.md)
- backup and restore drill: [docs/BACKUP_RESTORE_DRILL.md](docs/BACKUP_RESTORE_DRILL.md)

Contributor planning docs still live under `docs/`, but you do not need them to fork, deploy, or evaluate the project.

## What The App Does Today
- first-run setup at `/setup` if no admin password exists yet
- daily sign-in at `/login`
- board UI at `/board`
- operator status UI at `/status`
- settings UI at `/settings`
- create, edit, move, reorder, archive, restore, and permanently delete cards
- PostgreSQL-backed sessions
- password rotation and session revocation
- archive retention policy
- JSON export and import
- `GET /healthz`, `GET /readyz`, `GET /metrics`, and `GET /api/status`

## Supported Deployment Contracts
Two deployment paths are supported:

1. repo-root Docker image
2. self-contained root binary built from `go build .`

Both paths now use the same runtime assumptions:
- embedded migrations
- embedded rollback shell
- embedded React runtime
- optional bootstrap-only `APP_PASSWORD`

More detail: [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)

Tagged releases also publish a GHCR image for the same runtime contract:
- `ghcr.io/<owner>/flux-board:<tag>`

## Requirements
- Go `1.24+`
- PostgreSQL
- Node.js `24+` for frontend verification and release-style local builds
- Docker if you want the supported local hosted path via `docker compose`

## Environment
Copy [.env.example](.env.example) and set:
- `DATABASE_URL`
- `APP_ENV`
- `PORT`
- optional `APP_PASSWORD`
- optional `APP_VERSION`

The root binary reads `.env` from the current working directory and still lets exported environment variables override any value from that file.

Important:
- `APP_PASSWORD` is bootstrap-only
- leave it empty if you prefer to finish first-run setup in the browser
- tagged release artifacts and Docker images already embed the tracked `VERSION`
- hosted Docker builds can inject the same release label with `BUILD_VERSION`
- set `APP_VERSION` only if you intentionally want runtime status to announce a different operator-supplied label

## Quick Start
### Docker
```sh
docker compose up --build
```

For hosted Docker paths that build from source, keep release parity by setting
`BUILD_VERSION` to the same value as [`VERSION`](VERSION) before the deploy.

### Local binary
```sh
go build -o flux-board .
./flux-board
```

Open [http://localhost:8080](http://localhost:8080).

If you change files under `web/`, rebuild the embedded React runtime before shipping a new local binary:

```sh
./scripts/verify-web.sh
```

## Verification
### Go
```sh
./scripts/verify-go.sh
```

### Go race
```sh
./scripts/verify-go-race.sh
```

### Workflows
```sh
./scripts/verify-workflows.sh
```

### Frontend
```sh
./scripts/verify-web.sh
```

### Release dry run
```sh
RELEASE_RUN_SMOKE=0 ./scripts/release-dry-run.sh
```

### Browser smoke
```sh
npm ci
export FLUX_PASSWORD="your-password"
./scripts/verify-smoke.sh
```

Related smoke lanes:
- `./scripts/verify-setup-smoke.sh`
- `./scripts/verify-settings-smoke.sh`
- `./scripts/verify-dnd-smoke.sh`
- `./scripts/verify-board-keyboard-smoke.sh`
- `./scripts/verify-next-preview.sh`
- `./scripts/verify-docker-smoke.sh`

### Restore drill
```sh
npm ci
export FLUX_PASSWORD="your-password"
export RESTORE_DRILL_DUMP_PATH="$PWD/backups/flux-board-YYYYMMDD-HHMMSS.dump"
export RESTORE_DATABASE_URL='postgres://flux:flux@127.0.0.1:5432/flux_restore?sslmode=disable'
./scripts/verify-restore-drill.sh
```

Full operator guidance: [docs/BACKUP_RESTORE_DRILL.md](docs/BACKUP_RESTORE_DRILL.md)

### Hosted auth proof
```sh
BASE_URL=https://your-host.example \
./scripts/verify-hosted-auth-browser.sh
```

Use this on macOS when Chrome already has a valid hosted session and you want a repo-owned artifact proving `/board` and `/settings` open directly.

### Hosted deploy proof
Preferred: point the verifier at an explicit host.

```sh
BASE_URL=https://your-host.example \
EXPECT_NEEDS_SETUP=false \
EXPECT_ENVIRONMENT=production \
./scripts/verify-hosted-deploy.sh
```

If you intentionally want the script to discover the live deployment from GitHub metadata, opt in explicitly:

```sh
ALLOW_LIVE_DEPLOYMENT_DISCOVERY=1 \
HOSTED_ENVIRONMENT=production \
./scripts/verify-hosted-deploy.sh
```

The script is safe by default and will refuse to probe a hosted deployment unless one of those two inputs is set.

## Hosted Templates
- local Docker stack: [docker-compose.yml](docker-compose.yml)
- Render Docker template: [deploy/render.yaml](deploy/render.yaml)
- Railway and Zeabur should point at the repo root Dockerfile
- tagged releases publish a GHCR image from the same root Dockerfile

## Repository Layout
- [main.go](main.go): supported root runtime entrypoint for self-contained builds
- [cmd/flux-board/main.go](cmd/flux-board/main.go): modular entrypoint retained during the package transition
- [internal/config/](internal/config): env loading
- [internal/domain/](internal/domain): core types and contracts
- [internal/service/](internal/service): auth, settings, and task orchestration
- [internal/store/postgres/](internal/store/postgres): PostgreSQL repositories and migrations
- [internal/transport/http/](internal/transport/http): handlers, mux, runtime serving, and HTTP helpers
- [web/](web): React + TypeScript + Vite frontend
- [static/index.html](static/index.html): legacy rollback shell served on `/legacy/`
- [docs/DEPLOYMENT.md](docs/DEPLOYMENT.md): deployment contract
- [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md): current architecture

## Current Non-Goals
- multi-user accounts
- RBAC
- workspaces
- OIDC / SSO

Those are intentionally outside the current product target.
