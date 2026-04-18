$ErrorActionPreference = "Stop"

$scriptDir = $PSScriptRoot
$root = (Resolve-Path (Join-Path $scriptDir "..")).Path
$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"

if ([string]::IsNullOrWhiteSpace($env:TEST_RESULTS_DIR)) {
  $env:TEST_RESULTS_DIR = Join-Path (Join-Path "test-results" "restore-drill") "verify-restore-drill-$timestamp"
}

$resultsDir = $env:TEST_RESULTS_DIR
$statusResultsDir = Join-Path $resultsDir "status-contract"
$browserResultsDir = Join-Path $resultsDir "browser"
$restoreDumpPath = $env:RESTORE_DRILL_DUMP_PATH
$restoreDatabaseURL = $env:RESTORE_DATABASE_URL
$pgRestoreCommand = if ([string]::IsNullOrWhiteSpace($env:RESTORE_DRILL_PG_RESTORE_BIN)) {
  "pg_restore"
} else {
  $env:RESTORE_DRILL_PG_RESTORE_BIN
}
$baseURL = if ([string]::IsNullOrWhiteSpace($env:BASE_URL)) {
  "http://127.0.0.1:8080"
} else {
  $env:BASE_URL.TrimEnd("/")
}
$baseUri = [Uri]$baseURL
$playwrightBrowser = if ([string]::IsNullOrWhiteSpace($env:PLAYWRIGHT_BROWSER)) {
  if ([string]::IsNullOrWhiteSpace($env:SMOKE_BROWSER)) { "chromium" } else { $env:SMOKE_BROWSER }
} else {
  $env:PLAYWRIGHT_BROWSER
}
$readyAttempts = if ($env:RESTORE_READY_ATTEMPTS) { [int]$env:RESTORE_READY_ATTEMPTS } else { 60 }
$readyDelaySeconds = if ($env:RESTORE_READY_DELAY_SECONDS) { [int]$env:RESTORE_READY_DELAY_SECONDS } else { 2 }
$appPort = if ($env:PORT) {
  [int]$env:PORT
} elseif ($baseUri.IsDefaultPort) {
  if ($baseUri.Scheme -eq "https") { 443 } else { 80 }
} else {
  $baseUri.Port
}
$forcedAppEnv = $false
$stdoutLog = Join-Path $resultsDir "server.stdout.log"
$stderrLog = Join-Path $resultsDir "server.stderr.log"
$restoreStdoutLog = Join-Path $resultsDir "pg-restore.stdout.log"
$restoreStderrLog = Join-Path $resultsDir "pg-restore.stderr.log"
$binaryName = if ($env:OS -eq "Windows_NT") { "flux-board.exe" } else { "flux-board" }
$binaryPath = if ([string]::IsNullOrWhiteSpace($env:APP_BINARY)) {
  Join-Path $root $binaryName
} elseif (Test-Path $env:APP_BINARY) {
  (Resolve-Path $env:APP_BINARY).Path
} else {
  Join-Path $root $env:APP_BINARY
}

function Resolve-CommandPath {
  param([string]$CommandName)

  if (Test-Path $CommandName) {
    return (Resolve-Path $CommandName).Path
  }

  $command = Get-Command $CommandName -ErrorAction SilentlyContinue
  if ($command) {
    return $command.Source
  }

  return $null
}

function Show-Logs {
  param([string[]]$Paths)

  foreach ($path in $Paths) {
    if (Test-Path $path) {
      Write-Host "===== $(Split-Path $path -Leaf) ====="
      Get-Content $path -Tail 200
    }
  }
}

function Ensure-WebDist {
  if ($env:VERIFY_RESTORE_DRILL_WEB_BUILD -eq "0") {
    Write-Host "[prep] Skipping web build because VERIFY_RESTORE_DRILL_WEB_BUILD=0"
    return
  }

  $webDistIndex = Join-Path $root "web/dist/index.html"
  if (Test-Path $webDistIndex) {
    $webDistContents = Get-Content $webDistIndex -Raw
    if ($webDistContents -notmatch "Flux Board Runtime Placeholder") {
      return
    }
  }

  Write-Host "[prep] web/dist is missing or still using the placeholder runtime; building the React runtime first"
  & (Join-Path $scriptDir "verify-web.ps1")
  if ($LASTEXITCODE -ne 0) {
    throw "verify-web.ps1 failed with exit code $LASTEXITCODE"
  }
}

function Wait-ForReady {
  param(
    [string]$Url,
    [System.Diagnostics.Process]$Process
  )

  Write-Host "[ready] Wait for $Url/readyz"
  for ($attempt = 1; $attempt -le $readyAttempts; $attempt++) {
    if ($Process.HasExited) {
      Show-Logs -Paths @($stdoutLog, $stderrLog)
      throw "app process exited before readiness check passed"
    }

    try {
      $response = Invoke-WebRequest -Uri "$Url/readyz" -Method Get -TimeoutSec 5
      if ([int]$response.StatusCode -eq 200) {
        Write-Host "App is ready."
        return
      }
    } catch {
      # keep polling until timeout
    }

    Start-Sleep -Seconds $readyDelaySeconds
  }

  Show-Logs -Paths @($stdoutLog, $stderrLog)
  throw "app did not become ready in time"
}

if ([string]::IsNullOrWhiteSpace($restoreDumpPath)) {
  throw "RESTORE_DRILL_DUMP_PATH must point at a custom-format pg_dump artifact."
}
if (-not (Test-Path $restoreDumpPath)) {
  throw "RESTORE_DRILL_DUMP_PATH does not exist: $restoreDumpPath"
}
if ([string]::IsNullOrWhiteSpace($restoreDatabaseURL)) {
  throw "RESTORE_DATABASE_URL is required for the scratch restore target."
}
if ([string]::IsNullOrWhiteSpace($env:FLUX_PASSWORD) -and [string]::IsNullOrWhiteSpace($env:APP_PASSWORD)) {
  throw "FLUX_PASSWORD (or APP_PASSWORD) is required so the drill can sign in after restore."
}

foreach ($required in @("go", "node")) {
  if (-not (Resolve-CommandPath $required)) {
    throw "$required is required for verify-restore-drill.ps1"
  }
}

$resolvedPgRestore = Resolve-CommandPath $pgRestoreCommand
if (-not $resolvedPgRestore) {
  throw "pg_restore is required for verify-restore-drill.ps1. Set RESTORE_DRILL_PG_RESTORE_BIN if it is installed in a non-standard location."
}
$pgRestoreCommand = $resolvedPgRestore

New-Item -ItemType Directory -Force -Path $resultsDir | Out-Null
Push-Location $root

if ([string]::IsNullOrWhiteSpace($env:APP_ENV) -and $baseURL -match '^http://(127\.0\.0\.1|localhost)(:\d+)?$') {
  Write-Host "[prep] APP_ENV is unset on loopback HTTP; forcing APP_ENV=development so browser session cookies remain restore-drill testable."
  $env:APP_ENV = "development"
  $forcedAppEnv = $true
}

$appProcess = $null
$originalDatabaseURL = $env:DATABASE_URL
$originalPort = $env:PORT

try {
  Get-FileHash -Path $restoreDumpPath -Algorithm SHA256 | Format-List | Out-String | Set-Content (Join-Path $resultsDir "dump.sha256.txt")

  Ensure-WebDist

  Write-Host "[restore] Restore $restoreDumpPath into scratch database"
  & $pgRestoreCommand `
    --clean `
    --if-exists `
    --no-owner `
    --exit-on-error `
    --dbname $restoreDatabaseURL `
    $restoreDumpPath 1> $restoreStdoutLog 2> $restoreStderrLog
  if ($LASTEXITCODE -ne 0) {
    Show-Logs -Paths @($restoreStdoutLog, $restoreStderrLog)
    throw "pg_restore failed for $restoreDumpPath"
  }

  if ($env:VERIFY_RESTORE_DRILL_BUILD -eq "0") {
    Write-Host "[build] Skipping app build because VERIFY_RESTORE_DRILL_BUILD=0"
  } else {
    Write-Host "[build] go build -o $binaryPath ."
    & go build -o $binaryPath .
    if ($LASTEXITCODE -ne 0) {
      throw "go build failed with exit code $LASTEXITCODE"
    }
  }

  $env:DATABASE_URL = $restoreDatabaseURL
  $env:PORT = [string]$appPort

  Write-Host "[start] Start app $binaryPath"
  $appProcess = Start-Process -FilePath $binaryPath -WorkingDirectory $root -RedirectStandardOutput $stdoutLog -RedirectStandardError $stderrLog -PassThru

  Wait-ForReady -Url $baseURL -Process $appProcess

  $expectedEnvironment = if ([string]::IsNullOrWhiteSpace($env:EXPECT_ENVIRONMENT)) { $env:APP_ENV } else { $env:EXPECT_ENVIRONMENT }

  Write-Host "[status] Verify /healthz, /readyz, /api/status, /status, and /metrics"
  $env:TEST_RESULTS_DIR = $statusResultsDir
  $env:BASE_URL = $baseURL
  $env:EXPECT_NEEDS_SETUP = "false"
  if (-not [string]::IsNullOrWhiteSpace($expectedEnvironment)) {
    $env:EXPECT_ENVIRONMENT = $expectedEnvironment
  }
  & (Join-Path $scriptDir "verify-status-contract.ps1")
  if ($LASTEXITCODE -ne 0) {
    Show-Logs -Paths @($stdoutLog, $stderrLog)
    throw "verify-status-contract.ps1 failed with exit code $LASTEXITCODE"
  }

  Write-Host "[browser] Verify /login, /board, /settings, and /api/export (browser=$playwrightBrowser)"
  $env:TEST_RESULTS_DIR = $browserResultsDir
  $env:BASE_URL = $baseURL
  $env:PLAYWRIGHT_BROWSER = $playwrightBrowser
  & node (Join-Path $scriptDir "verify-restore-drill.mjs")
  if ($LASTEXITCODE -ne 0) {
    Show-Logs -Paths @($stdoutLog, $stderrLog)
    throw "verify-restore-drill.mjs failed with exit code $LASTEXITCODE"
  }

  Write-Host "Restore drill completed successfully. Results: $resultsDir"
} finally {
  if ($null -ne $appProcess -and -not $appProcess.HasExited) {
    Stop-Process -Id $appProcess.Id -Force
  }

  if ($forcedAppEnv) {
    Remove-Item Env:APP_ENV -ErrorAction SilentlyContinue
  }

  if ($null -eq $originalDatabaseURL) {
    Remove-Item Env:DATABASE_URL -ErrorAction SilentlyContinue
  } else {
    $env:DATABASE_URL = $originalDatabaseURL
  }

  if ($null -eq $originalPort) {
    Remove-Item Env:PORT -ErrorAction SilentlyContinue
  } else {
    $env:PORT = $originalPort
  }

  Pop-Location
}
