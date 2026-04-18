$ErrorActionPreference = "Stop"

$root = (Resolve-Path ".").Path
$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"

if ([string]::IsNullOrWhiteSpace($env:TEST_RESULTS_DIR)) {
  $env:TEST_RESULTS_DIR = Join-Path (Join-Path "test-results" "smoke") "verify-smoke-$timestamp"
}

$resultsDir = $env:TEST_RESULTS_DIR
New-Item -ItemType Directory -Force -Path $resultsDir | Out-Null

$baseURL = if ([string]::IsNullOrWhiteSpace($env:BASE_URL)) {
  "http://127.0.0.1:8080"
} else {
  $env:BASE_URL.TrimEnd("/")
}
$env:BASE_URL = $baseURL
$forcedAppEnv = $false

if ([string]::IsNullOrWhiteSpace($env:APP_ENV) -and $baseURL -match '^http://(127\.0\.0\.1|localhost)(:\d+)?$') {
  Write-Host "[prep] APP_ENV is unset on loopback HTTP; forcing APP_ENV=development so browser session cookies remain smoke-testable."
  $env:APP_ENV = "development"
  $forcedAppEnv = $true
}

if ([string]::IsNullOrWhiteSpace($env:PLAYWRIGHT_BROWSER)) {
  $env:PLAYWRIGHT_BROWSER = "chromium"
}

$smokeScript = if ([string]::IsNullOrWhiteSpace($env:SMOKE_SCRIPT)) {
  "smoke:login"
} else {
  $env:SMOKE_SCRIPT
}

$binaryName = if ($env:OS -eq "Windows_NT") { "flux-board.exe" } else { "flux-board" }
$binaryPath = if ([string]::IsNullOrWhiteSpace($env:APP_BINARY)) {
  Join-Path $root $binaryName
} elseif (Test-Path $env:APP_BINARY) {
  (Resolve-Path $env:APP_BINARY).Path
} else {
  Join-Path $root $env:APP_BINARY
}

$stdoutLog = Join-Path $resultsDir "server.stdout.log"
$stderrLog = Join-Path $resultsDir "server.stderr.log"
$readyAttempts = if ($env:SMOKE_READY_ATTEMPTS) { [int]$env:SMOKE_READY_ATTEMPTS } else { 60 }
$readyDelaySeconds = if ($env:SMOKE_READY_DELAY_SECONDS) { [int]$env:SMOKE_READY_DELAY_SECONDS } else { 2 }
$webDistIndex = Join-Path $root "web/dist/index.html"

function Show-ServerLogs {
  foreach ($path in @($stdoutLog, $stderrLog)) {
    if (Test-Path $path) {
      Write-Host "===== $(Split-Path $path -Leaf) ====="
      Get-Content $path -Tail 200
    }
  }
}

function Invoke-GoBuildBinary {
  param([string]$OutputPath)

  if ($env:VERIFY_SMOKE_BUILD -eq "0") {
    Write-Host "[1/4] Skipping app build because VERIFY_SMOKE_BUILD=0"
    return
  }

  Write-Host "[1/4] go build -o $OutputPath ."
  & go build -o $OutputPath .
  if ($LASTEXITCODE -ne 0) {
    throw "go build failed with exit code $LASTEXITCODE"
  }
}

function Ensure-WebDist {
  if ($env:VERIFY_SMOKE_WEB_BUILD -eq "0") {
    Write-Host "[prep] Skipping web build because VERIFY_SMOKE_WEB_BUILD=0"
    return
  }

  if (Test-Path $webDistIndex) {
    $webDistContents = Get-Content $webDistIndex -Raw
    if ($webDistContents -notmatch "Flux Board Runtime Placeholder") {
      return
    }
  }

  Write-Host "[prep] web/dist is missing or still using the placeholder runtime; building the React runtime first"
  & (Join-Path $PSScriptRoot "verify-web.ps1")
  if ($LASTEXITCODE -ne 0) {
    throw "verify-web.ps1 failed with exit code $LASTEXITCODE"
  }
}

function Wait-ForReady {
  param(
    [string]$Url,
    [System.Diagnostics.Process]$Process
  )

  Write-Host "[3/4] Wait for $Url/readyz"
  for ($attempt = 1; $attempt -le $readyAttempts; $attempt++) {
    if ($Process.HasExited) {
      Show-ServerLogs
      throw "app process exited before readiness check passed"
    }

    try {
      $request = [System.Net.HttpWebRequest]::Create("$Url/readyz")
      $request.Method = "GET"
      $request.Timeout = 5000
      $response = $request.GetResponse()
      try {
        if ([int]$response.StatusCode -eq 200) {
          Write-Host "App is ready."
          return
        }
      } finally {
        $response.Close()
      }
    } catch {
      # keep polling until timeout
    }

    Start-Sleep -Seconds $readyDelaySeconds
  }

  Show-ServerLogs
  throw "app did not become ready in time"
}

function Invoke-Smoke {
  Write-Host "[4/4] npm run $smokeScript (browser=$env:PLAYWRIGHT_BROWSER)"
  & npm run $smokeScript
  if ($LASTEXITCODE -ne 0) {
    Show-ServerLogs
    throw "npm run $smokeScript failed with exit code $LASTEXITCODE"
  }
}

$process = $null

try {
  Ensure-WebDist
  Invoke-GoBuildBinary -OutputPath $binaryPath

  Write-Host "[2/4] Start app $binaryPath"
  $process = Start-Process -FilePath $binaryPath -WorkingDirectory $root -RedirectStandardOutput $stdoutLog -RedirectStandardError $stderrLog -PassThru

  Wait-ForReady -Url $baseURL -Process $process
  Invoke-Smoke

  Write-Host "Smoke verification completed successfully. Results: $resultsDir"
} catch {
  Write-Host "Smoke verification failed. Results: $resultsDir"
  throw
} finally {
  if ($forcedAppEnv) {
    Remove-Item Env:APP_ENV -ErrorAction SilentlyContinue
  }
  if ($process -and -not $process.HasExited) {
    Stop-Process -Id $process.Id -Force
  }
}
