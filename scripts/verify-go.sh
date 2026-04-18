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
  echo "No Go packages discovered for verification." >&2
  exit 1
fi

echo "[1/3] go test -count=1 $packages"
go test -count=1 $packages

echo "[2/3] go vet $packages"
go vet $packages

echo "[3/3] go build $packages"
go build $packages

echo "Backend verification completed successfully."
