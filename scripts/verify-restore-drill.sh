#!/usr/bin/env sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/.." && pwd)
timestamp=$(date +"%Y%m%d-%H%M%S")

results_dir=${TEST_RESULTS_DIR:-"test-results/restore-drill/verify-restore-drill-$timestamp"}
base_url=${BASE_URL:-"http://127.0.0.1:8080"}
base_url=${base_url%/}
app_binary=${APP_BINARY:-"./flux-board"}
restore_dump_path=${RESTORE_DRILL_DUMP_PATH:-}
restore_database_url=${RESTORE_DATABASE_URL:-}
pg_restore_bin=${RESTORE_DRILL_PG_RESTORE_BIN:-pg_restore}
ready_attempts=${RESTORE_READY_ATTEMPTS:-60}
ready_delay_seconds=${RESTORE_READY_DELAY_SECONDS:-2}
playwright_browser=${PLAYWRIGHT_BROWSER:-"${SMOKE_BROWSER:-chromium}"}
status_results_dir="$results_dir/status-contract"
browser_results_dir="$results_dir/browser"
restore_stdout_log="$results_dir/pg-restore.stdout.log"
restore_stderr_log="$results_dir/pg-restore.stderr.log"
stdout_log="$results_dir/server.stdout.log"
stderr_log="$results_dir/server.stderr.log"
server_pid=""
forced_app_env=0

extract_port() {
  host_port=${1#http://}
  host_port=${host_port#https://}
  host_port=${host_port%%/*}

  case "$host_port" in
    *:*)
      printf '%s\n' "${host_port##*:}"
      ;;
    *)
      case "$1" in
        https://*)
          printf '443\n'
          ;;
        *)
          printf '80\n'
          ;;
      esac
      ;;
  esac
}

app_port=${PORT:-$(extract_port "$base_url")}

ensure_command() {
  command_name=$1
  help_text=$2

  if command -v "$command_name" >/dev/null 2>&1; then
    return
  fi

  if [ -x "$command_name" ]; then
    return
  fi

  echo "$help_text" >&2
  exit 1
}

ensure_required_env() {
  variable_name=$1
  variable_value=$2
  help_text=$3

  if [ -n "$variable_value" ]; then
    return
  fi

  echo "$help_text" >&2
  exit 1
}

ensure_web_dist() {
  if [ "${VERIFY_RESTORE_DRILL_WEB_BUILD:-1}" = "0" ]; then
    echo "[prep] Skipping web build because VERIFY_RESTORE_DRILL_WEB_BUILD=0"
    return
  fi

  if [ -f "$repo_root/web/dist/index.html" ] && ! grep -q "Flux Board Runtime Placeholder" "$repo_root/web/dist/index.html"; then
    return
  fi

  echo "[prep] web/dist is missing or still using the placeholder runtime; building the React runtime first"
  sh "$script_dir/verify-web.sh"
}

show_restore_logs() {
  for path in "$restore_stdout_log" "$restore_stderr_log"; do
    if [ -f "$path" ]; then
      echo "===== $(basename "$path") ====="
      tail -n 200 "$path" || true
    fi
  done
}

show_server_logs() {
  for path in "$stdout_log" "$stderr_log"; do
    if [ -f "$path" ]; then
      echo "===== $(basename "$path") ====="
      tail -n 200 "$path" || true
    fi
  done
}

write_dump_checksum() {
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$restore_dump_path" >"$results_dir/dump.sha256.txt"
    return
  fi

  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$restore_dump_path" >"$results_dir/dump.sha256.txt"
    return
  fi

  printf 'No SHA-256 checksum tool found for %s\n' "$restore_dump_path" >"$results_dir/dump.sha256.txt"
}

cleanup() {
  if [ "$forced_app_env" = "1" ]; then
    unset APP_ENV || true
  fi
  if [ -n "$server_pid" ] && kill -0 "$server_pid" 2>/dev/null; then
    kill "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
  fi
}

trap cleanup EXIT INT TERM

ensure_required_env "RESTORE_DRILL_DUMP_PATH" "$restore_dump_path" "RESTORE_DRILL_DUMP_PATH must point at a custom-format pg_dump artifact."
ensure_required_env "RESTORE_DATABASE_URL" "$restore_database_url" "RESTORE_DATABASE_URL is required for the scratch restore target."
ensure_required_env "FLUX_PASSWORD" "${FLUX_PASSWORD:-${APP_PASSWORD:-}}" "FLUX_PASSWORD (or APP_PASSWORD) is required so the drill can sign in after restore."

if [ ! -f "$restore_dump_path" ]; then
  echo "RESTORE_DRILL_DUMP_PATH does not exist: $restore_dump_path" >&2
  exit 1
fi

ensure_command go "go is required for verify-restore-drill.sh"
ensure_command node "node is required for verify-restore-drill.sh"
ensure_command curl "curl is required for verify-restore-drill.sh"
ensure_command "$pg_restore_bin" "pg_restore is required for verify-restore-drill.sh. Set RESTORE_DRILL_PG_RESTORE_BIN if it is installed in a non-standard location."

mkdir -p "$results_dir"
cd "$repo_root"

if [ -z "${APP_ENV:-}" ]; then
  case "$base_url" in
    http://127.0.0.1|http://127.0.0.1:*|http://localhost|http://localhost:*)
      echo "[prep] APP_ENV is unset on loopback HTTP; forcing APP_ENV=development so browser session cookies remain restore-drill testable."
      export APP_ENV="development"
      forced_app_env=1
      ;;
  esac
fi

write_dump_checksum
ensure_web_dist

echo "[restore] Restore $restore_dump_path into scratch database"
if ! "$pg_restore_bin" \
  --clean \
  --if-exists \
  --no-owner \
  --exit-on-error \
  --dbname "$restore_database_url" \
  "$restore_dump_path" >"$restore_stdout_log" 2>"$restore_stderr_log"; then
  show_restore_logs
  echo "pg_restore failed for $restore_dump_path" >&2
  exit 1
fi

if [ "${VERIFY_RESTORE_DRILL_BUILD:-1}" != "0" ]; then
  echo "[build] go build -o $app_binary ."
  go build -o "$app_binary" .
else
  echo "[build] Skipping app build because VERIFY_RESTORE_DRILL_BUILD=0"
fi

echo "[start] Start app $app_binary"
DATABASE_URL="$restore_database_url" PORT="$app_port" "$app_binary" >"$stdout_log" 2>"$stderr_log" &
server_pid=$!

echo "[ready] Wait for $base_url/readyz"
attempt=1
while [ "$attempt" -le "$ready_attempts" ]; do
  if ! kill -0 "$server_pid" 2>/dev/null; then
    show_server_logs
    echo "app process exited before readiness check passed" >&2
    exit 1
  fi

  code=$(curl -s -o /dev/null -w "%{http_code}" "$base_url/readyz" || true)
  if [ "$code" = "200" ]; then
    echo "App is ready."
    break
  fi

  if [ "$attempt" -eq "$ready_attempts" ]; then
    show_server_logs
    echo "app did not become ready in time" >&2
    exit 1
  fi

  sleep "$ready_delay_seconds"
  attempt=$((attempt + 1))
done

echo "[status] Verify /healthz, /readyz, /api/status, /status, and /metrics"
if [ -n "${EXPECT_ENVIRONMENT:-${APP_ENV:-}}" ]; then
  if ! TEST_RESULTS_DIR="$status_results_dir" \
    BASE_URL="$base_url" \
    EXPECT_NEEDS_SETUP=false \
    EXPECT_ENVIRONMENT="${EXPECT_ENVIRONMENT:-$APP_ENV}" \
    sh "$script_dir/verify-status-contract.sh"; then
    show_server_logs
    exit 1
  fi
else
  if ! TEST_RESULTS_DIR="$status_results_dir" \
    BASE_URL="$base_url" \
    EXPECT_NEEDS_SETUP=false \
    sh "$script_dir/verify-status-contract.sh"; then
    show_server_logs
    exit 1
  fi
fi

echo "[browser] Verify /login, /board, /settings, and /api/export (browser=$playwright_browser)"
if ! TEST_RESULTS_DIR="$browser_results_dir" \
  BASE_URL="$base_url" \
  PLAYWRIGHT_BROWSER="$playwright_browser" \
  node "$script_dir/verify-restore-drill.mjs"; then
  show_server_logs
  exit 1
fi

echo "Restore drill completed successfully. Results: $results_dir"
