#Requires -Version 5.0

$ErrorActionPreference = 'Stop'

Import-Module -WarningAction Ignore -Name "$PSScriptRoot\utils.psm1"

try {
    Write-Host "Invoking validate.ps1"
    Invoke-Script -File "$PSScriptRoot\validate.ps1"

    Write-Host "Invoking test.ps1"
    Invoke-Script -File "$PSScriptRoot\test.ps1"

    Write-Host "Invoking build.ps1"
    Invoke-Script -File "$PSScriptRoot\build.ps1"

    Write-Host "Invoking package.ps1"
    Invoke-Script -File "$PSScriptRoot\package.ps1"
} catch {
    Write-Host "Failed running $_"
    exit 1
}
