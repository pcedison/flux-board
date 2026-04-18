#!/usr/bin/env sh
set -eu

actionlint_bin="${ACTIONLINT_BIN:-"$(go env GOPATH)/bin/actionlint"}"
actionlint_version="${ACTIONLINT_VERSION:-v1.7.12}"
actionlint_toolchain="${ACTIONLINT_GOTOOLCHAIN:-auto}"

if [ ! -x "$actionlint_bin" ] || [ "${VERIFY_WORKFLOWS_INSTALL:-1}" != "0" ]; then
  echo "[1/2] GOTOOLCHAIN=$actionlint_toolchain go install github.com/rhysd/actionlint/cmd/actionlint@$actionlint_version"
  GOTOOLCHAIN="$actionlint_toolchain" GOBIN="$(dirname "$actionlint_bin")" go install "github.com/rhysd/actionlint/cmd/actionlint@$actionlint_version"
else
  echo "[1/2] Reusing existing actionlint binary at $actionlint_bin"
fi

echo "[2/2] $actionlint_bin"
"$actionlint_bin"

echo "Workflow verification completed successfully."
