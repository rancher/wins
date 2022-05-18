$ErrorActionPreference = "Stop"

# docker build
$buildTags = @{ "17763" = "1809"; "20348" = "ltsc2022";}
$buildNumber = (Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\' -ErrorAction Ignore).CurrentBuildNumber
$SERVERCORE_VERSION = $buildTags[$buildNumber]
if (-not $SERVERCORE_VERSION) {
    $SERVERCORE_VERSION = "1809"
}

Get-ChildItem -Path $PSScriptRoot\docker -Name Dockerfile.* | ForEach-Object {
    $dockerfile = $_
    $tag = $dockerfile -replace "Dockerfile.", ""
    docker build `
        --build-arg SERVERCORE_VERSION=$SERVERCORE_VERSION `
        -t $tag `
        -f $PSScriptRoot\docker\$dockerfile .
    if ($LASTEXITCODE -ne 0) {
        Log-Fatal "Failed to build testing docker image"
        exit $LASTEXITCODE
    }
}

# test
New-Item -Type Directory -Force -ErrorAction Ignore -Path @(
    "c:\etc\rancher\wins"
)  | Out-Null
Get-ChildItem -Path $PSScriptRoot -Name *.ps1 -Exclude $MyInvocation.MyCommand.Name | ForEach-Object {
    Invoke-Expression -Command "$PSScriptRoot\$_"
    if ($LASTEXITCODE -ne 0) {
        Log-Fatal "Failed to pass $PSScriptRoot\$_"
        exit $LASTEXITCODE
    }
}
Remove-Item -Recurse -Force -ErrorAction Ignore -Path @(
    "c:\etc\rancher\wins"
)