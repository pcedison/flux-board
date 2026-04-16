#!/usr/bin/env sh
set -eu

echo "[1/3] npm --prefix web ci --no-fund --no-audit"
npm --prefix web ci --no-fund --no-audit

echo "[2/3] npm --prefix web run typecheck"
npm --prefix web run typecheck

echo "[3/3] npm --prefix web run build"
npm --prefix web run build

echo "Web verification completed successfully."
