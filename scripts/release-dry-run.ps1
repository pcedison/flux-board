$ErrorActionPreference = "Stop"

$root = (Resolve-Path ".").Path
$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"

if ([string]::IsNullOrWhiteSpace($env:RELEASE_OUTPUT_DIR)) {
  $env:RELEASE_OUTPUT_DIR = Join-Path (Join-Path "test-results" "release") "release-dry-run-$timestamp"
}

$outputDir = $env:RELEASE_OUTPUT_DIR
New-Item -ItemType Directory -Force -Path $outputDir | Out-Null

$binaryName = if ($env:OS -eq "Windows_NT") { "flux-board.exe" } else { "flux-board" }
$binaryPath = Join-Path $outputDir $binaryName
$checksumPath = Join-Path $outputDir "$binaryName.sha256"
$runSmoke = if ([string]::IsNullOrWhiteSpace($env:RELEASE_RUN_SMOKE)) { $true } else { $env:RELEASE_RUN_SMOKE -ne "0" }

Write-Host "[1/3] go build -o $binaryPath ."
& go build -o $binaryPath .
if ($LASTEXITCODE -ne 0) {
  throw "go build failed with exit code $LASTEXITCODE"
}

Write-Host "[2/3] Generate checksum $checksumPath"
$hash = Get-FileHash -Path $binaryPath -Algorithm SHA256
"$($hash.Hash.ToLowerInvariant())  $binaryName" | Set-Content -Path $checksumPath -NoNewline

if ($runSmoke) {
  Write-Host "[3/3] Run smoke with release artifact"
  $previousAppBinary = $env:APP_BINARY
  $previousReleaseSmoke = $env:VERIFY_SMOKE_BUILD
  $previousResultsDir = $env:TEST_RESULTS_DIR

  try {
    $env:APP_BINARY = $binaryPath
    $env:VERIFY_SMOKE_BUILD = "0"
    if ([string]::IsNullOrWhiteSpace($env:TEST_RESULTS_DIR)) {
      $env:TEST_RESULTS_DIR = Join-Path $outputDir "smoke"
    }
    & (Join-Path $PSScriptRoot "verify-smoke.ps1")
    if ($LASTEXITCODE -ne 0) {
      throw "release smoke failed with exit code $LASTEXITCODE"
    }
  } finally {
    $env:APP_BINARY = $previousAppBinary
    $env:VERIFY_SMOKE_BUILD = $previousReleaseSmoke
    $env:TEST_RESULTS_DIR = $previousResultsDir
  }
} else {
  Write-Host "[3/3] Skipping smoke because RELEASE_RUN_SMOKE=0"
}

Write-Host "Release dry run completed successfully. Output: $outputDir"
