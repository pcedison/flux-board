$ErrorActionPreference = "Stop"

function Invoke-WebCommand {
  param(
    [string]$Label,
    [string[]]$Arguments,
    [int]$Retries = 1
  )

  for ($attempt = 1; $attempt -le $Retries; $attempt++) {
    Write-Host $Label
    & npm @Arguments
    if ($LASTEXITCODE -eq 0) {
      return
    }
    if ($attempt -lt $Retries) {
      Write-Host "npm $($Arguments -join ' ') failed with exit code $LASTEXITCODE, retrying ($attempt/$Retries)..."
      Start-Sleep -Seconds 2
      continue
    }
    throw "npm $($Arguments -join ' ') failed with exit code $LASTEXITCODE"
  }
}

Invoke-WebCommand "[1/4] npm --prefix web ci --no-fund --no-audit" @("--prefix", "web", "ci", "--no-fund", "--no-audit") 3
Invoke-WebCommand "[2/4] npm --prefix web run typecheck" @("--prefix", "web", "run", "typecheck")
Invoke-WebCommand "[3/4] npm --prefix web run test:run" @("--prefix", "web", "run", "test:run")
Invoke-WebCommand "[4/4] npm --prefix web run build" @("--prefix", "web", "run", "build")

Write-Host "Web verification completed successfully."
