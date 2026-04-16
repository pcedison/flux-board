#!/usr/bin/env sh
set -eu

echo "[1/3] go test -count=1 ./..."
go test -count=1 ./...

echo "[2/3] go vet ./..."
go vet ./...

echo "[3/3] go build ./..."
go build ./...

echo "Backend verification completed successfully."
