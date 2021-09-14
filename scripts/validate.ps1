#Requires -Version 5.0

$ErrorActionPreference = 'Stop'
Import-Module -WarningAction Ignore -Name "$PSScriptRoot\utils.psm1"

$SRC_PATH = (Resolve-Path "$PSScriptRoot\..").Path
Push-Location $SRC_PATH

Log-Info "Running validation"
Get-Command -ErrorAction Ignore -Name @("golangci-lint.exe") | Out-Null
if (-not $?) {
    Log-Info "Skipping validation: no golangci-lint available"
    exit 1
}

Log-Info "Running: golangci-lint"
golangci-lint run

Log-Info "Running: go fmt"
go fmt ./...