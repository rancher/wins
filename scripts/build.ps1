#Requires -Version 5.0

$ErrorActionPreference = 'Stop'

Import-Module -WarningAction Ignore -Name "$PSScriptRoot\utils.psm1"

function Build
{
    param (
        [parameter(Mandatory = $true)] [string]$Version,
        [parameter(Mandatory = $true)] [string]$Commit,
        [parameter(Mandatory = $true)] [string]$Output
    )

    $linkFlags = ('-s -w -X github.com/rancher/wins/pkg/defaults.AppVersion={0} -X github.com/rancher/wins/pkg/defaults.AppCommit={1} -extldflags "-static"' -f $Version, $Commit)
    go build -i -ldflags $linkFlags -o $Output cmd\main.go
    if (-not $?) {
        Log-Fatal "go build failed!"
    }
}

Invoke-Script -File "$PSScriptRoot\version.ps1"

$SRC_PATH = (Resolve-Path "$PSScriptRoot\..").Path
Push-Location $SRC_PATH

Remove-Item -Path "$SRC_PATH\bin\*" -Force -ErrorAction Ignore
$null = New-Item -Type Directory -Path bin -ErrorAction Ignore
$env:GOARCH = $env:ARCH
$env:GOOS = 'windows'
$env:CGO_ENABLED = 0
Build -Version $env:VERSION -Commit $env:COMMIT -Output "bin\wins.exe"
Build -Version "container" -Commit "container" -Output "bin\wins-container.exe"

Pop-Location
