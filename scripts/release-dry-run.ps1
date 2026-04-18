$ErrorActionPreference = "Stop"

$root = (Resolve-Path ".").Path
$version = & (Join-Path $PSScriptRoot "validate-release.ps1")

$hostGoOS = (& go env GOHOSTOS).Trim()
if ($LASTEXITCODE -ne 0) {
  throw "go env GOHOSTOS failed with exit code $LASTEXITCODE"
}

$hostGoArch = (& go env GOHOSTARCH).Trim()
if ($LASTEXITCODE -ne 0) {
  throw "go env GOHOSTARCH failed with exit code $LASTEXITCODE"
}

$targetGoOS = if ([string]::IsNullOrWhiteSpace($env:RELEASE_GOOS)) { $hostGoOS } else { $env:RELEASE_GOOS }
$targetGoArch = if ([string]::IsNullOrWhiteSpace($env:RELEASE_GOARCH)) { $hostGoArch } else { $env:RELEASE_GOARCH }
$artifactStem = "{0}-v{1}-{2}-{3}" -f ($(if ([string]::IsNullOrWhiteSpace($env:RELEASE_BINARY_BASENAME)) { "flux-board" } else { $env:RELEASE_BINARY_BASENAME })), $version, $targetGoOS, $targetGoArch
$timestamp = Get-Date -Format "yyyyMMdd-HHmmss"

if ([string]::IsNullOrWhiteSpace($env:RELEASE_OUTPUT_DIR)) {
  $env:RELEASE_OUTPUT_DIR = Join-Path (Join-Path "test-results" "release") "release-dry-run-v$version-$targetGoOS-$targetGoArch-$timestamp"
}

$outputDir = $env:RELEASE_OUTPUT_DIR
New-Item -ItemType Directory -Force -Path $outputDir | Out-Null

$binaryName = if ($targetGoOS -eq "windows") { "$artifactStem.exe" } else { $artifactStem }
$binaryPath = Join-Path $outputDir $binaryName
$checksumPath = Join-Path $outputDir "$binaryName.sha256"
$checksumsPath = Join-Path $outputDir "SHA256SUMS"
$runSmoke = if ([string]::IsNullOrWhiteSpace($env:RELEASE_RUN_SMOKE)) { $true } else { $env:RELEASE_RUN_SMOKE -ne "0" }
$webDistIndex = Join-Path $root "web/dist/index.html"

Write-Host "[1/4] Validate VERSION and CHANGELOG for v$version"

if (($env:RELEASE_WEB_BUILD -ne "0") -and -not (Test-Path $webDistIndex)) {
  Write-Host "[prep] web/dist is missing; building the React runtime first"
  & (Join-Path $PSScriptRoot "verify-web.ps1")
  if ($LASTEXITCODE -ne 0) {
    throw "verify-web.ps1 failed with exit code $LASTEXITCODE"
  }
}

Write-Host "[2/4] go build -o $binaryPath ."
$previousGoOS = $env:GOOS
$previousGoArch = $env:GOARCH
$previousCGOEnabled = $env:CGO_ENABLED

try {
  $env:GOOS = $targetGoOS
  $env:GOARCH = $targetGoArch
  if ([string]::IsNullOrWhiteSpace($env:CGO_ENABLED)) {
    $env:CGO_ENABLED = "0"
  }

  & go build -trimpath -o $binaryPath .
} finally {
  $env:GOOS = $previousGoOS
  $env:GOARCH = $previousGoArch
  $env:CGO_ENABLED = $previousCGOEnabled
}

if ($LASTEXITCODE -ne 0) {
  throw "go build failed with exit code $LASTEXITCODE"
}

Write-Host "[3/4] Generate checksums"
$hash = Get-FileHash -Path $binaryPath -Algorithm SHA256
$checksumLine = "$($hash.Hash.ToLowerInvariant())  $binaryName"
$checksumLine | Set-Content -Path $checksumPath -NoNewline
$checksumLine | Set-Content -Path $checksumsPath -NoNewline

if ($runSmoke) {
  if ($targetGoOS -ne $hostGoOS -or $targetGoArch -ne $hostGoArch) {
    Write-Host "[4/4] Skipping smoke because target $targetGoOS/$targetGoArch does not match host $hostGoOS/$hostGoArch"
  } else {
    Write-Host "[4/4] Run smoke with release artifact"
    $previousAppBinary = $env:APP_BINARY
    $previousReleaseSmoke = $env:VERIFY_SMOKE_BUILD
    $previousResultsDir = $env:TEST_RESULTS_DIR

    try {
      $env:APP_BINARY = $binaryPath
      $env:VERIFY_SMOKE_BUILD = "0"
      if ([string]::IsNullOrWhiteSpace($env:TEST_RESULTS_DIR)) {
        $env:TEST_RESULTS_DIR = Join-Path $outputDir "smoke"
      }
      & (Join-Path $PSScriptRoot "verify-smoke.ps1")
    } finally {
      $env:APP_BINARY = $previousAppBinary
      $env:VERIFY_SMOKE_BUILD = $previousReleaseSmoke
      $env:TEST_RESULTS_DIR = $previousResultsDir
    }
  }
} else {
  Write-Host "[4/4] Skipping smoke because RELEASE_RUN_SMOKE=0"
}

Write-Host "Release dry run completed successfully. Output: $outputDir"
