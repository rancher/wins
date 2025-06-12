$ErrorActionPreference = "Stop"

# docker build
$buildTags = @{ "17763" = "1809"; "20348" = "ltsc2022"; "26100" = "2025"}
$buildNumber = (Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\' -ErrorAction Ignore).CurrentBuildNumber
$SERVERCORE_VERSION = $buildTags[$buildNumber]
if (-not $SERVERCORE_VERSION) {
    $SERVERCORE_VERSION = "1809"
}

# Testing on 2025 is a bit tricky. We're actually only using it to build the 2019 artifacts since 2019 was removed from GHA.
# So, all of our integration tests expect us to be on 2019. This prevents some test containers from
# building correctly. Since we don't actually support 2025 yet, there isn't much point running the integration
# tests anyway. It's also important to note that these integration tests cover functionality that is no longer used
# by Rancher provisioning, so we aren't losing any relevant test coverage by short circuiting here.
if ($SERVERCORE_VERSION -eq "2025") {
    Write-Host "Detected that we are running on Windows 2025. Integration tests are not yet supported on this version. Exiting."
    exit 0
}


Write-Host "Server core version: $SERVERCORE_VERSION"

$NGINX_URL = 'https://nginx.org/download/nginx-1.21.3.zip';
Write-Host ('Downloading Nginx from {0}...'  -f $NGINX_URL);

[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12;
Invoke-WebRequest -UseBasicParsing -OutFile $PSScriptRoot\bin\nginx.zip -Uri $NGINX_URL;

Get-ChildItem -Path $PSScriptRoot\docker -Name Dockerfile.* | ForEach-Object {
    $dockerfile = $_
    $tag = $dockerfile -replace "Dockerfile.", ""
    Write-Host "Building $tag from $_"

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
