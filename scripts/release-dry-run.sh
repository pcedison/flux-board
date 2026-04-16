#!/usr/bin/env sh
set -eu

timestamp=$(date +"%Y%m%d-%H%M%S")
output_dir=${RELEASE_OUTPUT_DIR:-"test-results/release/release-dry-run-$timestamp"}
binary_name=${RELEASE_BINARY_NAME:-"flux-board"}
binary_path="$output_dir/$binary_name"
checksum_path="$output_dir/$binary_name.sha256"
run_smoke=${RELEASE_RUN_SMOKE:-1}

mkdir -p "$output_dir"

echo "[1/3] go build -o $binary_path ."
go build -o "$binary_path" .

echo "[2/3] Generate checksum $checksum_path"
sha256sum "$binary_path" >"$checksum_path"

if [ "$run_smoke" != "0" ]; then
  echo "[3/3] Run smoke with release artifact"
  APP_BINARY="$binary_path" \
  VERIFY_SMOKE_BUILD=0 \
  TEST_RESULTS_DIR="${TEST_RESULTS_DIR:-$output_dir/smoke}" \
  sh "$(dirname "$0")/verify-smoke.sh"
else
  echo "[3/3] Skipping smoke because RELEASE_RUN_SMOKE=0"
fi

echo "Release dry run completed successfully. Output: $output_dir"
