$ErrorActionPreference = "Stop"

$shell = Get-Command sh -ErrorAction SilentlyContinue
if (-not $shell) {
  $shell = Get-Command bash -ErrorAction SilentlyContinue
}

if (-not $shell) {
  throw "verify-hosted-auth-browser.ps1 requires sh or bash on PATH."
}

& $shell.Source (Join-Path $PSScriptRoot "verify-hosted-auth-browser.sh")
