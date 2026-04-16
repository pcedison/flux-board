$ErrorActionPreference = "Stop"

$msysBin = "C:\msys64\ucrt64\bin"

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
    throw "no Go packages discovered for race verification"
  }

  return [string[]]$dirs
}

function Ensure-WindowsRaceToolchain {
  if ($env:OS -ne "Windows_NT") {
    return
  }

  if (-not (Test-Path $msysBin)) {
    throw "MSYS2 UCRT64 toolchain not found at $msysBin. Install MSYS2 and mingw-w64-ucrt-x86_64-gcc first."
  }

  if (-not ($env:Path -split ";" | Where-Object { $_ -eq $msysBin })) {
    $env:Path = "$msysBin;$env:Path"
  }

  $env:CGO_ENABLED = "1"
  $env:CC = Join-Path $msysBin "gcc.exe"
  $env:CXX = Join-Path $msysBin "g++.exe"

  $syncLib = & $env:CC --print-file-name libsynchronization.a
  if ($LASTEXITCODE -ne 0) {
    throw "failed to probe libsynchronization.a with $env:CC"
  }
  if ([string]::Equals(($syncLib | Out-String).Trim(), "libsynchronization.a", [System.StringComparison]::OrdinalIgnoreCase)) {
    throw "The configured GCC toolchain does not expose libsynchronization.a. Install a newer mingw-w64 runtime."
  }
}

function Invoke-GoRace {
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

Ensure-WindowsRaceToolchain
$packages = Get-GoPackages
$packageLabel = $packages -join " "
Invoke-GoRace "[1/1] go test -race -count=1 $packageLabel" @("test", "-race", "-count=1") + $packages

Write-Host "Go race verification completed successfully."
