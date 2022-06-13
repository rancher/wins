#Requires -Version 5.0
$ErrorActionPreference = "Stop"

Invoke-Expression -Command "$PSScriptRoot\version.ps1"

$DIR_PATH = Split-Path -Parent $MyInvocation.MyCommand.Definition
$SRC_PATH = (Resolve-Path "$DIR_PATH\..").Path

# Reference binary in ./bin/wins.exe
Copy-Item -Force -Path $SRC_PATH\bin\wins.exe -Destination $SRC_PATH\package\windows | Out-Null
Copy-Item -Force -Path $SRC_PATH\install.ps1 -Destination $SRC_PATH\package\windows | Out-Null
Copy-Item -Force -Path $SRC_PATH\suc\run.ps1 -Destination $SRC_PATH\package\windows | Out-Null

Set-Location -Path $SRC_PATH\package\windows


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

$buildTags = @{ "17763" = "1809"; "20348" = "ltsc2022";}
$buildNumber = (Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\' -ErrorAction Ignore).CurrentBuildNumber
$WINDOWS_VERSION = $buildTags[$buildNumber]
if (-not $WINDOWS_VERSION) {
    $WINDOWS_VERSION = "1809"
}

$IMAGE = ('{0}/wins:{1}-windows-{2}' -f $REPO, $TAG, $WINDOWS_VERSION)

$ARCH = $env:ARCH

docker build `
    --build-arg SERVERCORE_VERSION=$WINDOWS_VERSION `
    --build-arg ARCH=$ARCH `
    --build-arg VERSION=$TAG `
    -t $IMAGE `
    -f Dockerfile .

if ($LASTEXITCODE -ne 0) {
    exit $LASTEXITCODE
}

Write-Host "Built $IMAGE`n"
Set-Location -Path $SRC_PATH
