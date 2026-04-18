#!/usr/bin/env sh
set -eu

run_npm() {
  label=$1
  retries=$2
  shift 2

  attempt=1
  while [ "$attempt" -le "$retries" ]; do
    echo "$label"
    if npm "$@"; then
      return 0
    fi
    if [ "$attempt" -lt "$retries" ]; then
      echo "npm $* failed, retrying ($attempt/$retries)..." >&2
      sleep 2
    fi
    attempt=$((attempt + 1))
  done

  echo "npm $* failed after $retries attempts" >&2
  return 1
}

run_npm "[1/5] npm --prefix web ci --no-fund --no-audit" 3 --prefix web ci --no-fund --no-audit

run_npm "[2/5] npm --prefix web run typecheck" 1 --prefix web run typecheck

run_npm "[3/5] npm --prefix web run lint" 1 --prefix web run lint

run_npm "[4/5] npm --prefix web run test:run" 1 --prefix web run test:run

run_npm "[5/5] npm --prefix web run build" 1 --prefix web run build

echo "Web verification completed successfully."
