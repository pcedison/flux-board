$ErrorActionPreference = "Stop"

function Invoke-WebCommand {
  param(
    [string]$Label,
    [string[]]$Arguments
  )

  Write-Host $Label
  & npm @Arguments
  if ($LASTEXITCODE -ne 0) {
    throw "npm $($Arguments -join ' ') failed with exit code $LASTEXITCODE"
  }
}

Invoke-WebCommand "[1/3] npm --prefix web ci --no-fund --no-audit" @("--prefix", "web", "ci", "--no-fund", "--no-audit")
Invoke-WebCommand "[2/3] npm --prefix web run typecheck" @("--prefix", "web", "run", "typecheck")
Invoke-WebCommand "[3/3] npm --prefix web run build" @("--prefix", "web", "run", "build")

Write-Host "Web verification completed successfully."
