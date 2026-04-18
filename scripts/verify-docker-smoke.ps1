$ErrorActionPreference = "Stop"

$root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"

if ([string]::IsNullOrWhiteSpace($env:TEST_RESULTS_DIR)) {
  $env:TEST_RESULTS_DIR = Join-Path (Join-Path "test-results" "docker-smoke") "verify-docker-smoke-$timestamp"
}

$resultsDir = $env:TEST_RESULTS_DIR
New-Item -ItemType Directory -Force -Path $resultsDir | Out-Null

$baseURL = if ([string]::IsNullOrWhiteSpace($env:BASE_URL)) {
  "http://127.0.0.1:8080"
} else {
  $env:BASE_URL.TrimEnd("/")
}
$env:BASE_URL = $baseURL

$hostPort = if ($env:DOCKER_HOST_PORT) { $env:DOCKER_HOST_PORT } else { "8080" }
$containerPort = if ($env:DOCKER_CONTAINER_PORT) { $env:DOCKER_CONTAINER_PORT } else { "8080" }
$imageTag = if ([string]::IsNullOrWhiteSpace($env:DOCKER_IMAGE_TAG)) { "flux-board:docker-smoke" } else { $env:DOCKER_IMAGE_TAG }
$containerName = if ([string]::IsNullOrWhiteSpace($env:DOCKER_CONTAINER_NAME)) { "flux-board-docker-smoke-$timestamp" } else { $env:DOCKER_CONTAINER_NAME }
$dockerDatabaseURL = if ([string]::IsNullOrWhiteSpace($env:DOCKER_DATABASE_URL)) { $env:DATABASE_URL } else { $env:DOCKER_DATABASE_URL }
$containerAppEnv = if ([string]::IsNullOrWhiteSpace($env:APP_ENV)) { "development" } else { $env:APP_ENV }
$containerAppPassword = $env:APP_PASSWORD
$smokeScript = if ([string]::IsNullOrWhiteSpace($env:SMOKE_SCRIPT)) { "smoke:login" } else { $env:SMOKE_SCRIPT }
$readyAttempts = if ($env:SMOKE_READY_ATTEMPTS) { [int]$env:SMOKE_READY_ATTEMPTS } else { 60 }
$readyDelaySeconds = if ($env:SMOKE_READY_DELAY_SECONDS) { [int]$env:SMOKE_READY_DELAY_SECONDS } else { 2 }
$stdoutLog = Join-Path $resultsDir "container.stdout.log"

if (-not (Get-Command docker -ErrorAction SilentlyContinue)) {
  throw "docker is required for verify-docker-smoke.ps1"
}

if ([string]::IsNullOrWhiteSpace($dockerDatabaseURL)) {
  throw "DOCKER_DATABASE_URL or DATABASE_URL must be set for verify-docker-smoke.ps1"
}

function Show-ContainerLogs {
  $names = docker ps -a --format '{{.Names}}'
  if ($names -contains $containerName) {
    docker logs $containerName *> $stdoutLog
    Write-Host "===== $(Split-Path $stdoutLog -Leaf) ====="
    Get-Content $stdoutLog -Tail 200
  }
}

function Remove-Container {
  $names = docker ps -a --format '{{.Names}}'
  if ($names -contains $containerName) {
    docker rm -f $containerName | Out-Null
  }
}

try {
  if ($env:VERIFY_DOCKER_SMOKE_BUILD -ne "0") {
    Write-Host "[1/4] docker build -t $imageTag ."
    docker build -t $imageTag $root
  } else {
    Write-Host "[1/4] Skipping docker build because VERIFY_DOCKER_SMOKE_BUILD=0"
  }

  Write-Host "[2/4] Start container $containerName"
  Remove-Container
  $useHostGateway = if ([string]::IsNullOrWhiteSpace($env:DOCKER_USE_HOST_GATEWAY)) { "1" } else { $env:DOCKER_USE_HOST_GATEWAY }
  if ($useHostGateway -eq "1") {
    docker run -d --name $containerName --add-host=host.docker.internal:host-gateway `
      -e PORT=$containerPort `
      -e APP_ENV=$containerAppEnv `
      -e APP_PASSWORD=$containerAppPassword `
      -e DATABASE_URL=$dockerDatabaseURL `
      -p "127.0.0.1:${hostPort}:${containerPort}" `
      $imageTag | Out-Null
  } else {
    docker run -d --name $containerName `
      -e PORT=$containerPort `
      -e APP_ENV=$containerAppEnv `
      -e APP_PASSWORD=$containerAppPassword `
      -e DATABASE_URL=$dockerDatabaseURL `
      -p "127.0.0.1:${hostPort}:${containerPort}" `
      $imageTag | Out-Null
  }

  Write-Host "[3/4] Wait for $baseURL/readyz"
  for ($attempt = 1; $attempt -le $readyAttempts; $attempt++) {
    $running = docker ps --format '{{.Names}}'
    if (-not ($running -contains $containerName)) {
      Show-ContainerLogs
      throw "container exited before readiness check passed"
    }

    try {
      $request = [System.Net.HttpWebRequest]::Create("$baseURL/readyz")
      $request.Method = "GET"
      $request.Timeout = 5000
      $response = $request.GetResponse()
      try {
        if ([int]$response.StatusCode -eq 200) {
          Write-Host "Container is ready."
          break
        }
      } finally {
        $response.Close()
      }
    } catch {
      # keep polling until timeout
    }

    if ($attempt -eq $readyAttempts) {
      Show-ContainerLogs
      throw "container did not become ready in time"
    }

    Start-Sleep -Seconds $readyDelaySeconds
  }

  Write-Host "[4/4] npm run $smokeScript against Docker runtime"
  npm run $smokeScript
  if ($LASTEXITCODE -ne 0) {
    Show-ContainerLogs
    throw "npm run $smokeScript failed with exit code $LASTEXITCODE"
  }

  Show-ContainerLogs
  Write-Host "Docker smoke verification completed successfully. Results: $resultsDir"
} finally {
  Remove-Container
}
