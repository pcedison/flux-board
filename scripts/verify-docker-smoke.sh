#!/usr/bin/env sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/.." && pwd)
timestamp=$(date +"%Y%m%d-%H%M%S")
results_dir=${TEST_RESULTS_DIR:-"test-results/docker-smoke/verify-docker-smoke-$timestamp"}
base_url=${BASE_URL:-"http://127.0.0.1:8080"}
base_url=${base_url%/}
host_port=${DOCKER_HOST_PORT:-8080}
container_port=${DOCKER_CONTAINER_PORT:-8080}
image_tag=${DOCKER_IMAGE_TAG:-"flux-board:docker-smoke"}
container_name=${DOCKER_CONTAINER_NAME:-"flux-board-docker-smoke-$timestamp"}
docker_database_url=${DOCKER_DATABASE_URL:-${DATABASE_URL:-}}
container_app_env=${APP_ENV:-development}
container_app_password=${APP_PASSWORD-}
playwright_browser=${PLAYWRIGHT_BROWSER:-"${SMOKE_BROWSER:-chromium}"}
ready_attempts=${SMOKE_READY_ATTEMPTS:-60}
ready_delay_seconds=${SMOKE_READY_DELAY_SECONDS:-2}
smoke_script=${SMOKE_SCRIPT:-"smoke:login"}
use_host_gateway=${DOCKER_USE_HOST_GATEWAY:-1}
docker_add_host_arg="--add-host=host.docker.internal:host-gateway"

mkdir -p "$results_dir"
stdout_log="$results_dir/container.stdout.log"

export TEST_RESULTS_DIR="$results_dir"
export BASE_URL="$base_url"
export PLAYWRIGHT_BROWSER="$playwright_browser"

if ! command -v docker >/dev/null 2>&1; then
  echo "docker is required for verify-docker-smoke.sh" >&2
  exit 1
fi

if [ -z "$docker_database_url" ]; then
  echo "DOCKER_DATABASE_URL or DATABASE_URL must be set for verify-docker-smoke.sh" >&2
  exit 1
fi

show_container_logs() {
  if docker ps -a --format '{{.Names}}' | grep -qx "$container_name"; then
    docker logs "$container_name" >"$stdout_log" 2>&1 || true
    echo "===== $(basename "$stdout_log") ====="
    tail -n 200 "$stdout_log" || true
  fi
}

cleanup() {
  if docker ps -a --format '{{.Names}}' | grep -qx "$container_name"; then
    docker rm -f "$container_name" >/dev/null 2>&1 || true
  fi
}

trap cleanup EXIT INT TERM

if [ "${VERIFY_DOCKER_SMOKE_BUILD:-1}" != "0" ]; then
  echo "[1/4] docker build -t $image_tag ."
  docker build -t "$image_tag" "$repo_root"
else
  echo "[1/4] Skipping docker build because VERIFY_DOCKER_SMOKE_BUILD=0"
fi

echo "[2/4] Start container $container_name"
cleanup
if [ "$use_host_gateway" = "1" ]; then
  docker run -d \
    --name "$container_name" \
    "$docker_add_host_arg" \
    -e PORT="$container_port" \
    -e APP_ENV="$container_app_env" \
    -e APP_PASSWORD="$container_app_password" \
    -e DATABASE_URL="$docker_database_url" \
    -p "127.0.0.1:$host_port:$container_port" \
    "$image_tag" >/dev/null
else
  docker run -d \
    --name "$container_name" \
    -e PORT="$container_port" \
    -e APP_ENV="$container_app_env" \
    -e APP_PASSWORD="$container_app_password" \
    -e DATABASE_URL="$docker_database_url" \
    -p "127.0.0.1:$host_port:$container_port" \
    "$image_tag" >/dev/null
fi

echo "[3/4] Wait for $base_url/readyz"
attempt=1
while [ "$attempt" -le "$ready_attempts" ]; do
  if ! docker ps --format '{{.Names}}' | grep -qx "$container_name"; then
    show_container_logs
    echo "container exited before readiness check passed" >&2
    exit 1
  fi

  code=$(curl -s -o /dev/null -w "%{http_code}" "$base_url/readyz" || true)
  if [ "$code" = "200" ]; then
    echo "Container is ready."
    break
  fi

  if [ "$attempt" -eq "$ready_attempts" ]; then
    show_container_logs
    echo "container did not become ready in time" >&2
    exit 1
  fi

  sleep "$ready_delay_seconds"
  attempt=$((attempt + 1))
done

echo "[4/4] npm run $smoke_script against Docker runtime (browser=$PLAYWRIGHT_BROWSER)"
if ! npm run "$smoke_script"; then
  show_container_logs
  exit 1
fi

show_container_logs
echo "Docker smoke verification completed successfully. Results: $results_dir"
