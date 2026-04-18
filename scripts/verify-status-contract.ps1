$ErrorActionPreference = "Stop"

$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"

if ([string]::IsNullOrWhiteSpace($env:TEST_RESULTS_DIR)) {
  $env:TEST_RESULTS_DIR = Join-Path (Join-Path "test-results" "status-contract") "verify-status-contract-$timestamp"
}

if ([string]::IsNullOrWhiteSpace($env:BASE_URL)) {
  $env:BASE_URL = "http://127.0.0.1:8080"
} else {
  $env:BASE_URL = $env:BASE_URL.TrimEnd("/")
}

& node (Join-Path $PSScriptRoot "verify-status-contract.mjs")
if ($LASTEXITCODE -ne 0) {
  throw "verify-status-contract.mjs failed with exit code $LASTEXITCODE"
}
