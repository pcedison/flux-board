$ErrorActionPreference = "Stop"

function Invoke-GoCommand {
  param(
    [string]$Label,
    [string[]]$Arguments
  )

  Write-Host $Label
  & go @Arguments
  if ($LASTEXITCODE -ne 0) {
    throw "go $($Arguments -join ' ') failed with exit code $LASTEXITCODE"
  }
}

Invoke-GoCommand "[1/3] go test -count=1 ./..." @("test", "-count=1", "./...")

Invoke-GoCommand "[2/3] go vet ./..." @("vet", "./...")

Invoke-GoCommand "[3/3] go build ./..." @("build", "./...")

Write-Host "Backend verification completed successfully."
