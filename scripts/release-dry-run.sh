#!/usr/bin/env sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
timestamp=$(date +"%Y%m%d-%H%M%S")
version=$(sh "$script_dir/validate-release.sh")
host_goos=$(go env GOHOSTOS)
host_goarch=$(go env GOHOSTARCH)
target_goos=${RELEASE_GOOS:-"$host_goos"}
target_goarch=${RELEASE_GOARCH:-"$host_goarch"}
artifact_stem="${RELEASE_BINARY_BASENAME:-flux-board}-v$version-$target_goos-$target_goarch"
binary_name=$artifact_stem
if [ "$target_goos" = "windows" ]; then
  binary_name="$binary_name.exe"
fi
output_dir=${RELEASE_OUTPUT_DIR:-"test-results/release/release-dry-run-v$version-$target_goos-$target_goarch-$timestamp"}
binary_path="$output_dir/$binary_name"
checksum_name="$binary_name.sha256"
checksum_path="$output_dir/$checksum_name"
run_smoke=${RELEASE_RUN_SMOKE:-1}

mkdir -p "$output_dir"

if [ "${RELEASE_WEB_BUILD:-1}" != "0" ] && [ ! -f "web/dist/index.html" ]; then
  echo "[prep] web/dist is missing; building the React runtime first"
  sh "$script_dir/verify-web.sh"
fi

echo "[1/4] Validate VERSION and CHANGELOG for v$version"

echo "[2/4] go build -o $binary_path ."
CGO_ENABLED=${CGO_ENABLED:-0} GOOS="$target_goos" GOARCH="$target_goarch" go build -trimpath -o "$binary_path" .

echo "[3/4] Generate checksums"
(
  cd "$output_dir"
  sha256sum "$binary_name" >"$checksum_name"
  cp "$checksum_name" "SHA256SUMS"
)

if [ "$run_smoke" != "0" ]; then
  if [ "$target_goos" != "$host_goos" ] || [ "$target_goarch" != "$host_goarch" ]; then
    echo "[4/4] Skipping smoke because target $target_goos/$target_goarch does not match host $host_goos/$host_goarch"
  else
    echo "[4/4] Run smoke with release artifact"
    APP_BINARY="$binary_path" \
    VERIFY_SMOKE_BUILD=0 \
    TEST_RESULTS_DIR="${TEST_RESULTS_DIR:-$output_dir/smoke}" \
    sh "$script_dir/verify-smoke.sh"
  fi
else
  echo "[4/4] Skipping smoke because RELEASE_RUN_SMOKE=0"
fi

echo "Release dry run completed successfully. Output: $output_dir"
