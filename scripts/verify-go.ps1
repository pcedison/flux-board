$ErrorActionPreference = "Stop"

Write-Host "[1/3] go test ./..."
go test ./...

Write-Host "[2/3] go vet ./..."
go vet ./...

Write-Host "[3/3] go build ./..."
go build ./...

Write-Host "Backend verification completed successfully."
