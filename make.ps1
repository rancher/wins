#if (!(Test-Path .dapper.exe)) {
#    $dapperURL = "https://releases.rancher.com/dapper/latest/dapper-Windows-x86_64.exe"
#    Write-Host "no .dapper.exe, downloading $dapperURL"
#    curl.exe -sfL -o .dapper.exe $dapperURL
#}

if ($args.Count -eq 0) {
    $args = @("ci")
}


function WinsCIAction() {
    param (
        [parameter(Mandatory = $true, ValueFromPipeline = $true)] [string]$Action
    )
    Import-Module -WarningAction Ignore -Name "$PSScriptRoot\scripts\utils.psm1"
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
    -t $IMAGE `
    -f Dockerfile .

    if ($LASTEXITCODE -ne 0) {
        $env:TAG=""
        $env:VERSION=""
        $env:SERVERCORE_VERSION=""
        exit $LASTEXITCODE
    }
    Write-Host -ForegroundColor Green "Successfully built $IMAGE`n"
}

if ($args[0] -eq "integration") {
    Write-Host "Running Integration Tests"
    WinsCIAction -Action "integration"
    exit
}

if ($args[0] -eq "build") {
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

if ($args[0] -eq "all") {
    Write-Host "Running CI and Integration Tests"
    WinsCIAction -Action "ci"
    WinsCIAction -Action "integration"
    exit
}

if ($args[0] -eq "clean") {
    Remove-Item .dapper.exe
    Remove-Item Dockerfile.dapper* -Exclude "Dockerfile.dapper"
}

if (Test-Path scripts\$($args[0]).ps1) {
    WinsCIAction -Action "$($args[0])"
    exit
}

