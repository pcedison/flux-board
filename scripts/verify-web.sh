#!/usr/bin/env sh
set -eu

echo "[1/4] npm --prefix web ci --no-fund --no-audit"
npm --prefix web ci --no-fund --no-audit

echo "[2/4] npm --prefix web run typecheck"
npm --prefix web run typecheck

echo "[3/4] npm --prefix web run test:run"
npm --prefix web run test:run

echo "[4/4] npm --prefix web run build"
npm --prefix web run build

echo "Web verification completed successfully."
