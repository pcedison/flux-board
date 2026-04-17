$ErrorActionPreference = "Stop"

$previousSmokeScript = $env:SMOKE_SCRIPT

try {
  $env:SMOKE_SCRIPT = "smoke:board-keyboard"
  & (Join-Path $PSScriptRoot "verify-smoke.ps1")
} finally {
  if ([string]::IsNullOrEmpty($previousSmokeScript)) {
    Remove-Item Env:SMOKE_SCRIPT -ErrorAction SilentlyContinue
  } else {
    $env:SMOKE_SCRIPT = $previousSmokeScript
  }
}
