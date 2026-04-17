param(
  [string]$ReleaseTag = $env:RELEASE_TAG,
  [string]$ReleaseNotesOutput = $env:RELEASE_NOTES_OUTPUT,
  [string]$VersionFile = $env:RELEASE_VERSION_FILE,
  [string]$ChangelogFile = $env:RELEASE_CHANGELOG_FILE
)

$ErrorActionPreference = "Stop"

$scriptRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$repoRoot = (Resolve-Path (Join-Path $scriptRoot "..")).Path

if ([string]::IsNullOrWhiteSpace($VersionFile)) {
  $VersionFile = Join-Path $repoRoot "VERSION"
}

if ([string]::IsNullOrWhiteSpace($ChangelogFile)) {
  $ChangelogFile = Join-Path $repoRoot "CHANGELOG.md"
}

if (-not (Test-Path -LiteralPath $VersionFile)) {
  throw "missing VERSION file: $VersionFile"
}

if (-not (Test-Path -LiteralPath $ChangelogFile)) {
  throw "missing CHANGELOG.md file: $ChangelogFile"
}

$version = Get-Content -LiteralPath $VersionFile |
  ForEach-Object { $_.Trim() } |
  Where-Object { $_ } |
  Select-Object -First 1

if ([string]::IsNullOrWhiteSpace($version)) {
  throw "VERSION file is empty: $VersionFile"
}

$semverPattern = '^[0-9]+\.[0-9]+\.[0-9]+(?:-[0-9A-Za-z.-]+)?(?:\+[0-9A-Za-z.-]+)?$'
if ($version -notmatch $semverPattern) {
  throw "invalid semantic version in ${VersionFile}: $version"
}

$expectedTag = "v$version"
if (-not [string]::IsNullOrWhiteSpace($ReleaseTag) -and $ReleaseTag -ne $expectedTag) {
  throw "release tag $ReleaseTag does not match VERSION $expectedTag"
}

$changelog = Get-Content -LiteralPath $ChangelogFile -Raw
$escapedVersion = [Regex]::Escape($version)
$pattern = "(?ms)^## \[$escapedVersion\] - [^\r\n]+\r?\n(?<body>.*?)(?=^## \[|\z)"
$match = [Regex]::Match($changelog, $pattern)

if (-not $match.Success) {
  throw "CHANGELOG.md is missing a section for version $version"
}

$notes = $match.Groups["body"].Value.Trim()
if ([string]::IsNullOrWhiteSpace($notes)) {
  throw "CHANGELOG.md is missing a populated section for version $version"
}

if (-not [string]::IsNullOrWhiteSpace($ReleaseNotesOutput)) {
  $notesDir = Split-Path -Parent $ReleaseNotesOutput
  if (-not [string]::IsNullOrWhiteSpace($notesDir)) {
    New-Item -ItemType Directory -Force -Path $notesDir | Out-Null
  }
  Set-Content -LiteralPath $ReleaseNotesOutput -Value $notes
}

Write-Output $version
