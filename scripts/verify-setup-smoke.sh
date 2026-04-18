#!/usr/bin/env sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
previous_smoke_script=${SMOKE_SCRIPT-}
export SMOKE_SCRIPT="smoke:setup"

cleanup() {
  if [ -n "${previous_smoke_script}" ]; then
    export SMOKE_SCRIPT="$previous_smoke_script"
  else
    unset SMOKE_SCRIPT
  fi
}

trap cleanup EXIT

sh "$script_dir/verify-smoke.sh"
