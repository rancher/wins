#Requires -Version 5.0
$ErrorActionPreference = "Stop"

Invoke-Expression -Command "$PSScriptRoot\version.ps1"

$DIR_PATH = Split-Path -Parent $MyInvocation.MyCommand.Definition
$SRC_PATH = (Resolve-Path "$DIR_PATH\..").Path
cd $SRC_PATH\package\windows


$TAG = $env:TAG
if (-not $TAG) {
    $TAG = ('{0}{1}' -f $env:VERSION, $env:SUFFIX)
}
$REPO = $env:REPO
if (-not $REPO) {
    $REPO = "rancher"
}

if ($TAG | Select-String -Pattern 'dirty') {
    $TAG = "dev"
}

if ($env:DRONE_TAG) {
    $TAG = $env:DRONE_TAG
}

# Get release id as image tag suffix
$HOST_RELEASE_ID = (Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\' -ErrorAction Ignore).ReleaseId
$IMAGE = ('{0}/wins:{1}-windows-{2}' -f $REPO, $TAG, $HOST_RELEASE_ID)
if (-not $HOST_RELEASE_ID) {
    Log-Fatal "release ID not found"
}

$ARCH = $env:ARCH

docker build `
    --build-arg SERVERCORE_VERSION=$HOST_RELEASE_ID `
    --build-arg ARCH=$ARCH `
    --build-arg VERSION=$TAG `
    -t $IMAGE `
    -f Dockerfile .

Write-Host "Built $IMAGE`n"
