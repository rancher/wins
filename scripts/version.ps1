#Requires -Version 5.0
$ErrorActionPreference = 'Stop'

Import-Module -WarningAction Ignore -Name "$PSScriptRoot\utils.psm1"

$DIRTY = ""
if ("$(git status --porcelain --untracked-files=no)") {
    $DIRTY = "-dirty"
}

$COMMIT = $(git rev-parse --short HEAD)
$GIT_TAG = $env:DRONE_TAG
if (-not $GIT_TAG) {
    $GIT_TAG = $(git tag -l --contains HEAD | Select-Object -First 1)
}
$env:COMMIT = $COMMIT

$VERSION = "${env:COMMIT}${DIRTY}"
if ((-not $DIRTY) -and ($GIT_TAG)) {
    $VERSION = "${GIT_TAG}"
}
$env:VERSION = $VERSION

$TAG = ('{0}{1}' -f $env:VERSION, $env:SUFFIX)
if ($TAG | Select-String -Pattern 'dirty') {
    $TAG = "dev"
}

if (-not $env:REPO) {
    $env:REPO = "rancher"
}

if (($env:DRONE_TAG) -and (-Not ($TAG).Contains("dev") -or -Not ($TAG).Contains("dirty"))){
    $TAG = $env:DRONE_TAG
    $env:TAG = $TAG
} else {
    $env:TAG = $TAG
}

if (-not $env:ARCH) {
    $env:ARCH = "amd64"
}

$buildTags = @{ "17763" = "1809"; "20348" = "ltsc2022";}
$buildNumber = (Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\' -ErrorAction Ignore).CurrentBuildNumber
$env:SERVERCORE_VERSION = $buildTags[$buildNumber]
if (-not $env:SERVERCORE_VERSION) {
    $env:SERVERCORE_VERSION = "1809"
}

Write-Host "ARCH: $env:ARCH"
Write-Host "VERSION: $env:VERSION"
Write-Host "TAG: $env:TAG"
Write-Host "SERVERCORE_VERSION: $env:SERVERCORE_VERSION"
