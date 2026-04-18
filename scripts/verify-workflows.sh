#!/usr/bin/env sh
set -eu

actionlint_bin="${ACTIONLINT_BIN:-"$(go env GOPATH)/bin/actionlint"}"
actionlint_version="${ACTIONLINT_VERSION:-v1.7.12}"

if [ ! -x "$actionlint_bin" ] || [ "${VERIFY_WORKFLOWS_INSTALL:-1}" != "0" ]; then
  echo "[1/2] go install github.com/rhysd/actionlint/cmd/actionlint@$actionlint_version"
  GOBIN="$(dirname "$actionlint_bin")" go install "github.com/rhysd/actionlint/cmd/actionlint@$actionlint_version"
else
  echo "[1/2] Reusing existing actionlint binary at $actionlint_bin"
fi

echo "[2/2] $actionlint_bin"
"$actionlint_bin"

echo "Workflow verification completed successfully."
