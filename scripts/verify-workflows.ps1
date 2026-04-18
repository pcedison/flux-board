$ErrorActionPreference = "Stop"

$goBin = (& go env GOPATH).Trim()
if ($LASTEXITCODE -ne 0) {
  throw "go env GOPATH failed with exit code $LASTEXITCODE"
}

$actionlintVersion = if ([string]::IsNullOrWhiteSpace($env:ACTIONLINT_VERSION)) { "v1.7.12" } else { $env:ACTIONLINT_VERSION }
$actionlintToolchain = if ([string]::IsNullOrWhiteSpace($env:ACTIONLINT_GOTOOLCHAIN)) { "auto" } else { $env:ACTIONLINT_GOTOOLCHAIN }
$actionlintBin = if ([string]::IsNullOrWhiteSpace($env:ACTIONLINT_BIN)) {
  Join-Path (Join-Path $goBin "bin") "actionlint.exe"
} else {
  $env:ACTIONLINT_BIN
}

$installActionlint = -not (Test-Path $actionlintBin) -or $env:VERIFY_WORKFLOWS_INSTALL -ne "0"
if ($installActionlint) {
  Write-Host "[1/2] GOTOOLCHAIN=$actionlintToolchain go install github.com/rhysd/actionlint/cmd/actionlint@$actionlintVersion"
  $actionlintDir = Split-Path $actionlintBin -Parent
  New-Item -ItemType Directory -Force -Path $actionlintDir | Out-Null
  $previousGobin = $env:GOBIN
  $previousGoToolchain = $env:GOTOOLCHAIN
  try {
    $env:GOBIN = $actionlintDir
    $env:GOTOOLCHAIN = $actionlintToolchain
    & go install "github.com/rhysd/actionlint/cmd/actionlint@$actionlintVersion"
  } finally {
    $env:GOBIN = $previousGobin
    $env:GOTOOLCHAIN = $previousGoToolchain
  }
  if ($LASTEXITCODE -ne 0) {
    throw "go install actionlint failed with exit code $LASTEXITCODE"
  }
} else {
  Write-Host "[1/2] Reusing existing actionlint binary at $actionlintBin"
}

Write-Host "[2/2] $actionlintBin"
& $actionlintBin
if ($LASTEXITCODE -ne 0) {
  throw "actionlint failed with exit code $LASTEXITCODE"
}

Write-Host "Workflow verification completed successfully."
