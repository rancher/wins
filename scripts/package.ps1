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
$RELEASE_ID = $env:RELEASE_ID
if (-not $RELEASE_ID) {
    $RELEASE_ID = $HOST_RELEASE_ID
}
$IMAGE = ('{0}/wins:{1}-windows-{2}' -f $REPO, $TAG, $RELEASE_ID)

$ARCH = $env:ARCH
if ($RELEASE_ID -eq $HOST_RELEASE_ID) {
    docker build `
        --build-arg SERVERCORE_VERSION=$RELEASE_ID `
        --build-arg ARCH=$ARCH `
        --build-arg VERSION=$TAG `
        -t $IMAGE `
        -f Dockerfile .
} else {
    docker build `
        --isolation hyperv `
        --build-arg SERVERCORE_VERSION=$RELEASE_ID `
        --build-arg ARCH=$ARCH `
        --build-arg VERSION=$TAG `
        -t $IMAGE `
        -f Dockerfile .
}

Write-Host "Built $IMAGE`n"
