#!/usr/bin/env sh
set -eu

timestamp=$(date +"%Y%m%d-%H%M%S")
results_dir=${TEST_RESULTS_DIR:-"test-results/smoke/verify-smoke-$timestamp"}
base_url=${BASE_URL:-"http://127.0.0.1:8080"}
base_url=${base_url%/}
app_binary=${APP_BINARY:-"./flux-board"}
ready_attempts=${SMOKE_READY_ATTEMPTS:-60}
ready_delay_seconds=${SMOKE_READY_DELAY_SECONDS:-2}

export TEST_RESULTS_DIR="$results_dir"
export BASE_URL="$base_url"

mkdir -p "$results_dir"

stdout_log="$results_dir/server.stdout.log"
stderr_log="$results_dir/server.stderr.log"
server_pid=""

show_server_logs() {
  for path in "$stdout_log" "$stderr_log"; do
    if [ -f "$path" ]; then
      echo "===== $(basename "$path") ====="
      tail -n 200 "$path" || true
    fi
  done
}

cleanup() {
  if [ -n "$server_pid" ] && kill -0 "$server_pid" 2>/dev/null; then
    kill "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
  fi
}

trap cleanup EXIT INT TERM

if [ "${VERIFY_SMOKE_BUILD:-1}" != "0" ]; then
  echo "[1/4] go build -o $app_binary ."
  go build -o "$app_binary" .
else
  echo "[1/4] Skipping app build because VERIFY_SMOKE_BUILD=0"
fi

echo "[2/4] Start app $app_binary"
"$app_binary" >"$stdout_log" 2>"$stderr_log" &
server_pid=$!

echo "[3/4] Wait for $base_url/readyz"
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

echo "[4/4] npm run smoke:login"
if ! npm run smoke:login; then
  show_server_logs
  exit 1
fi

echo "Smoke verification completed successfully. Results: $results_dir"
