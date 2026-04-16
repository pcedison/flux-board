#!/usr/bin/env sh
set -eu

packages=$(
  find . -type f -name '*.go' \
    ! -path './web/*' \
    ! -path './test-results/*' \
    ! -path './node_modules/*' \
    -exec dirname {} \; |
    sort -u |
    awk '{ if ($0 == ".") { print "." } else { print $0 } }' |
    tr '\n' ' '
)

if [ -z "$packages" ]; then
  echo "No Go packages discovered for race verification." >&2
  exit 1
fi

echo "[1/1] go test -race -count=1 $packages"
go test -race -count=1 $packages

echo "Go race verification completed successfully."
