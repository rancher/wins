#Requires -Version 5.0

$ErrorActionPreference = 'Stop'

Import-Module -WarningAction Ignore -Name "$PSScriptRoot\scripts\utils.psm1"

function WinsCIAction() {
    param (
        [parameter(Mandatory = $true, ValueFromPipeline = $true)] [string]$Action
    )
    Invoke-Expression -Command "$PSScriptRoot\scripts\version.ps1"

    $IMAGE = ('{0}/wins:{1}-windows-{2}' -f $env:REPO, $env:TAG, $env:SERVERCORE_VERSION)
    Write-Host -ForegroundColor Yellow "Starting docker build of $IMAGE`n"
    Write-Host "CI Action: $Action"
    Write-Host "VERSION = $env:TAG"

    docker build `
    --build-arg SERVERCORE_VERSION=$env:SERVERCORE_VERSION `
    --build-arg ACTION=$Action `
    --build-arg VERSION=$env:VERSION `
    --build-arg MAINTAINERS=$env:MAINTAINERS `
    --build-arg REPO=https://github.com/rancher/wins `
    --tag $IMAGE `
    -f Dockerfile .

    if ($LASTEXITCODE -ne 0) {
        $env:TAG=""
        $env:VERSION=""
        $env:SERVERCORE_VERSION=""
        exit $LASTEXITCODE
    }
    Write-Host -ForegroundColor Green "Successfully built $IMAGE`n"
    docker cp $IMAGE:./wins.exe ./wins.exe
    Write-Host -ForegroundColor Green "Successfully staged wins binary from $IMAGE`n"
}

trap {
    Write-Host -NoNewline -ForegroundColor Red "[ERROR]: "
    Write-Host -ForegroundColor Red "$_"

    Pop-Location
    exit 1
}

if ($args[0] -eq "integration") {
    Write-Host "Running Integration Tests"
    WinsCIAction -Action "integration"
    exit
}

if ($args[0] -eq "build" -or $args[0] -eq "package") {
    Write-Host "Building wins"
    WinsCIAction -Action "build"
    exit
}

if ($args[0] -eq "package") {
    Write-Host "Building and Packaging wins"
    WinsCIAction -Action "build"
    WinsCIAction -Action "package"
    exit
}

if ($args[0] -eq "all" -or $args.Count -eq 0 -or $args[0] -eq "ci") {
    Write-Host "Running CI and Integration Tests"
    WinsCIAction -Action "ci"
    exit
}

if ($args[0] -eq "no-docker") {
    if ($args[1] -eq "") {
        Write-Host "Running CI without Docker"
        Invoke-Expression -Command "scripts\ci.ps1 ci"
    } else {
        Write-Host ('Running {0}.ps1 without Docker' -f $($args[1]))
        Invoke-Expression -Command "scripts\ci.ps1 $($args[1])"
    }
    exit
}

if ($args[0] -eq "clean") {
    Remove-Item .dapper.exe
    Remove-Item Dockerfile.dapper* -Exclude "Dockerfile.dapper"
}

if ($args[0] -eq "build") {
    Write-Host "Building"
    .dapper.exe -f Dockerfile.dapper build
    exit
}

if (Test-Path scripts\$($args[0]).ps1) {
    WinsCIAction -Action "$($args[0])"
    exit
}

