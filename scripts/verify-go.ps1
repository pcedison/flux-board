$ErrorActionPreference = "Stop"

function Get-GoPackages {
  $root = (Resolve-Path ".").Path
  $files = Get-ChildItem -Recurse -File -Filter *.go |
    Where-Object {
      $_.FullName -notlike "$root\\web\\*" -and
        $_.FullName -notlike "$root\\test-results\\*" -and
        $_.FullName -notlike "$root\\node_modules\\*"
    }

  $dirs = $files |
    ForEach-Object {
      if ($_.DirectoryName -eq $root) {
        "."
      } else {
        "./" + $_.DirectoryName.Substring($root.Length + 1).Replace("\", "/")
      }
    } |
    Sort-Object -Unique

  if (-not $dirs) {
    throw "no Go packages discovered for verification"
  }

  return [string[]]$dirs
}

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

$packages = Get-GoPackages
$packageLabel = $packages -join " "

Invoke-GoCommand "[1/3] go test -count=1 $packageLabel" @("test", "-count=1") + $packages

Invoke-GoCommand "[2/3] go vet $packageLabel" @("vet") + $packages

Invoke-GoCommand "[3/3] go build $packageLabel" @("build") + $packages

Write-Host "Backend verification completed successfully."
