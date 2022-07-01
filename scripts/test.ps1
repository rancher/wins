#Requires -Version 5.0

$ErrorActionPreference = 'Stop'

Import-Module -WarningAction Ignore -Name "$PSScriptRoot\utils.psm1"

Invoke-Script -File "$PSScriptRoot\version.ps1"

$SRC_PATH = (Resolve-Path "$PSScriptRoot\..").Path
Push-Location $SRC_PATH

$env:GOARCH = $env:ARCH
$env:GOOS = 'windows'
$env:CGO_ENABLED = 0
$LINKFLAGS = ('-X github.com/rancher/wins/pkg/defaults.AppVersion={0} -X github.com/rancher/wins/pkg/defaults.AppCommit={1}' -f $env:VERSION, $env:COMMIT)
go test --ldflags $LINKFLAGS -v -cover ./...
if (-not $?)
{
    Log-Fatal "validation test failed"
}

Pop-Location
