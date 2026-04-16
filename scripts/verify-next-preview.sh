#!/usr/bin/env sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname "$0")" && pwd)

sh "$script_dir/verify-web.sh"
SMOKE_SCRIPT=smoke:next sh "$script_dir/verify-smoke.sh"
