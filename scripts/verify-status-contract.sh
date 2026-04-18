#!/usr/bin/env sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
timestamp=$(date +"%Y%m%d-%H%M%S")

if [ -z "${TEST_RESULTS_DIR:-}" ]; then
  export TEST_RESULTS_DIR="test-results/status-contract/verify-status-contract-$timestamp"
fi

if [ -z "${BASE_URL:-}" ]; then
  export BASE_URL="http://127.0.0.1:8080"
else
  export BASE_URL=${BASE_URL%/}
fi

if ! command -v node >/dev/null 2>&1; then
  echo "node is required for verify-status-contract.sh" >&2
  exit 1
fi

node "$script_dir/verify-status-contract.mjs"
