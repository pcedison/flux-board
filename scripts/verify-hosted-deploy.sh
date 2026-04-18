#!/usr/bin/env sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/.." && pwd)
timestamp=$(date +"%Y%m%d-%H%M%S")
results_dir=${TEST_RESULTS_DIR:-"test-results/hosted-deploy/verify-hosted-deploy-$timestamp"}
deployment_environment=${HOSTED_ENVIRONMENT:-production}
deployment_sha=${HOSTED_DEPLOY_SHA:-$(git -C "$repo_root" rev-parse HEAD)}
repository_full_name=${GITHUB_REPOSITORY:-}
base_url=${BASE_URL-}
smoke_script=${HOSTED_SMOKE_SCRIPT-}
summary_path="$results_dir/summary.json"
status_contract_results_dir="$results_dir/status-contract"

mkdir -p "$results_dir"

if ! command -v gh >/dev/null 2>&1; then
  echo "gh is required for verify-hosted-deploy.sh" >&2
  exit 1
fi

if ! command -v curl >/dev/null 2>&1; then
  echo "curl is required for verify-hosted-deploy.sh" >&2
  exit 1
fi

normalize_repository_full_name() {
  remote_url=$1
  case "$remote_url" in
    git@github.com:*)
      normalized=${remote_url#git@github.com:}
      ;;
    git+https://github.com/*)
      normalized=${remote_url#git+https://github.com/}
      ;;
    https://github.com/*)
      normalized=${remote_url#https://github.com/}
      ;;
    ssh://git@github.com/*)
      normalized=${remote_url#ssh://git@github.com/}
      ;;
    *)
      echo "unsupported GitHub remote URL: $remote_url" >&2
      return 1
      ;;
  esac

  normalized=${normalized%.git}
  printf '%s\n' "$normalized"
}

fetch_repo_full_name() {
  if [ -n "$repository_full_name" ]; then
    printf '%s\n' "$repository_full_name"
    return 0
  fi

  remote_url=$(git -C "$repo_root" config --get remote.origin.url || true)
  if [ -z "$remote_url" ]; then
    echo "could not determine repository from remote.origin.url" >&2
    return 1
  fi

  normalize_repository_full_name "$remote_url"
}

fetch_deployment_metadata() {
  repo_full_name=$1

  deployment_id=$(gh api "repos/$repo_full_name/deployments?environment=$deployment_environment&sha=$deployment_sha&per_page=20" --jq '.[0].id // ""')
  if [ -z "$deployment_id" ] || [ "$deployment_id" = "null" ]; then
    echo "no deployment found for $repo_full_name at sha $deployment_sha in environment $deployment_environment" >&2
    return 1
  fi

  environment_url=$(gh api "repos/$repo_full_name/deployments/$deployment_id/statuses" --jq 'map(select(.state == "success"))[0].environment_url // ""')
  if [ -z "$environment_url" ]; then
    echo "deployment $deployment_id has no successful status with environment_url" >&2
    return 1
  fi

  target_url=$(gh api "repos/$repo_full_name/deployments/$deployment_id/statuses" --jq 'map(select(.state == "success"))[0].target_url // ""')

  printf '%s\n%s\n%s\n' "$deployment_id" "$environment_url" "$target_url"
}

check_endpoint() {
  name=$1
  path=$2
  expect_code=${3:-200}

  body_path="$results_dir/$name.body"
  headers_path="$results_dir/$name.headers"
  code=$(curl -sS -D "$headers_path" -o "$body_path" -w "%{http_code}" "$base_url$path" || true)
  printf '%s\n' "$code" > "$results_dir/$name.status"

  if [ "$code" != "$expect_code" ]; then
    echo "expected $path to return $expect_code, got $code" >&2
    if [ -f "$headers_path" ]; then
      echo "===== $(basename "$headers_path") =====" >&2
      cat "$headers_path" >&2
    fi
    if [ -f "$body_path" ]; then
      echo "===== $(basename "$body_path") =====" >&2
      cat "$body_path" >&2
    fi
    return 1
  fi
}

run_optional_auth_smoke() {
  if [ -n "$smoke_script" ]; then
    :
  elif [ "$needs_setup" = "true" ] && [ -n "${FLUX_SETUP_PASSWORD-}" ]; then
    smoke_script="smoke:setup"
  elif [ "$needs_setup" = "false" ] && [ -n "${FLUX_PASSWORD-}" ]; then
    smoke_script="smoke:login"
  fi

  if [ -z "$smoke_script" ]; then
    auth_smoke_status="skipped"
    if [ "$needs_setup" = "true" ]; then
      auth_smoke_reason="set FLUX_SETUP_PASSWORD or HOSTED_SMOKE_SCRIPT to verify setup -> board on the hosted deployment"
    else
      auth_smoke_reason="set FLUX_PASSWORD or HOSTED_SMOKE_SCRIPT to verify login -> board on the hosted deployment"
    fi
    return 0
  fi

  auth_smoke_status="passed"
  auth_smoke_reason=""
  auth_results_dir="$results_dir/auth-smoke"
  mkdir -p "$auth_results_dir"

  echo "[auth] npm run $smoke_script against $base_url"
  if ! (
    export BASE_URL="$base_url"
    export TEST_RESULTS_DIR="$auth_results_dir"
    npm run "$smoke_script"
  ); then
    auth_smoke_status="failed"
    auth_smoke_reason="repo-owned auth smoke failed"
    return 1
  fi
}

repo_full_name=$(fetch_repo_full_name)

if [ -z "$base_url" ]; then
  metadata=$(fetch_deployment_metadata "$repo_full_name")
  deployment_id=$(printf '%s\n' "$metadata" | sed -n '1p')
  base_url=$(printf '%s\n' "$metadata" | sed -n '2p')
  target_url=$(printf '%s\n' "$metadata" | sed -n '3p')
else
  deployment_id=""
  target_url=""
fi

base_url=${base_url%/}
printf '%s\n' "$base_url" > "$results_dir/base-url.txt"

echo "[1/5] Verify status contract on $base_url"
BASE_URL="$base_url" \
TEST_RESULTS_DIR="$status_contract_results_dir" \
EXPECT_ENVIRONMENT="${EXPECT_ENVIRONMENT:-$deployment_environment}" \
sh "$script_dir/verify-status-contract.sh"

needs_setup=$(node --input-type=module -e '
  import fs from "node:fs";
  const statusArtifact = JSON.parse(fs.readFileSync(process.argv[1], "utf8"));
  process.stdout.write(statusArtifact.body.needsSetup ? "true" : "false");
' "$status_contract_results_dir/status.json")

primary_route="/login"
if [ "$needs_setup" = "true" ]; then
  primary_route="/setup"
fi

echo "[2/5] Verify route availability beyond the status contract"
check_endpoint "root" "/"
check_endpoint "primary-route" "$primary_route"
check_endpoint "legacy" "/legacy/"

echo "[3/5] Capture deployment metadata"
printf '%s\n' "$repo_full_name" > "$results_dir/repository.txt"
printf '%s\n' "$deployment_sha" > "$results_dir/head-sha.txt"
printf '%s\n' "$deployment_environment" > "$results_dir/environment.txt"
printf '%s\n' "${deployment_id:-}" > "$results_dir/deployment-id.txt"
printf '%s\n' "${target_url:-}" > "$results_dir/deployment-target-url.txt"

auth_smoke_status="skipped"
auth_smoke_reason=""

echo "[4/5] Run optional auth smoke when credentials are available"
run_optional_auth_smoke

echo "[5/5] Write hosted verification summary"
BASE_URL_VALUE=$base_url \
REPOSITORY_VALUE=$repo_full_name \
DEPLOYMENT_SHA_VALUE=$deployment_sha \
DEPLOYMENT_ENVIRONMENT_VALUE=$deployment_environment \
DEPLOYMENT_ID_VALUE=${deployment_id:-} \
TARGET_URL_VALUE=${target_url:-} \
PRIMARY_ROUTE_VALUE=$primary_route \
NEEDS_SETUP_VALUE=$needs_setup \
AUTH_SMOKE_STATUS_VALUE=$auth_smoke_status \
AUTH_SMOKE_REASON_VALUE=$auth_smoke_reason \
RESULTS_DIR_VALUE=$results_dir \
STATUS_CONTRACT_RESULTS_DIR_VALUE=$status_contract_results_dir \
SUMMARY_PATH_VALUE=$summary_path \
node --input-type=module -e '
  import fs from "node:fs";

  const resultsDir = process.env.RESULTS_DIR_VALUE;
  const readText = (name) => fs.readFileSync(`${resultsDir}/${name}`, "utf8").trimEnd();
  const statusContractSummary = JSON.parse(
    fs.readFileSync(`${process.env.STATUS_CONTRACT_RESULTS_DIR_VALUE}/summary.json`, "utf8")
  );

  const summary = {
    status: process.env.AUTH_SMOKE_STATUS_VALUE === "failed" ? "failed" : "passed",
    checkedAt: new Date().toISOString(),
    repository: process.env.REPOSITORY_VALUE,
    headSha: process.env.DEPLOYMENT_SHA_VALUE,
    environment: process.env.DEPLOYMENT_ENVIRONMENT_VALUE,
    deploymentId: process.env.DEPLOYMENT_ID_VALUE || null,
    baseURL: process.env.BASE_URL_VALUE,
    targetURL: process.env.TARGET_URL_VALUE || null,
    statusContractResultsDir: process.env.STATUS_CONTRACT_RESULTS_DIR_VALUE,
    needsSetup: process.env.NEEDS_SETUP_VALUE === "true",
    primaryRoute: process.env.PRIMARY_ROUTE_VALUE,
    statusContract: statusContractSummary,
    routeChecks: {
      root: { status: Number(readText("root.status")) },
      primaryRoute: { path: process.env.PRIMARY_ROUTE_VALUE, status: Number(readText("primary-route.status")) },
      legacy: { status: Number(readText("legacy.status")) },
    },
    authSmoke: {
      status: process.env.AUTH_SMOKE_STATUS_VALUE,
      reason: process.env.AUTH_SMOKE_REASON_VALUE || null,
    },
  };

  fs.writeFileSync(process.env.SUMMARY_PATH_VALUE, `${JSON.stringify(summary, null, 2)}\n`);
'

echo "Hosted deployment verification completed successfully. Summary: $summary_path"
