#Requires -Version 5.0

$ErrorActionPreference = 'Stop'

Import-Module -WarningAction Ignore -Name "$PSScriptRoot\utils.psm1"

try {

    if (($args[0] -eq "all") -or ($args[0] -eq "ci") -or ($args[0] -eq "")) {
        Write-Host "Invoking validate.ps1"
        Invoke-Script -File "$PSScriptRoot\validate.ps1"

        Write-Host "Invoking test.ps1"
        Invoke-Script -File "$PSScriptRoot\test.ps1"

        Write-Host "Invoking build.ps1"
        Invoke-Script -File "$PSScriptRoot\build.ps1"

        Write-Host "Invoking integration.ps1"
        Invoke-Script -File "$PSScriptRoot\integration.ps1"

        Write-Host "Invoking package.ps1 to validate build artifacts"
        Invoke-Script -File "$PSScriptRoot\package.ps1"
    }

    if ($args[0] -eq "package") {
        Write-Host "Invoking build.ps1"
        Invoke-Script -File "$PSScriptRoot\build.ps1"
        Write-Host "Invoking package.ps1"
        Invoke-Script -File "$PSScriptRoot\package.ps1"
    }

    if ($args[0] -eq "integration") {
        Write-Host "Invoking build.ps1"
        Invoke-Script -File "$PSScriptRoot\build.ps1"

        Write-Host "Invoking integration.ps1"
        Invoke-Script -File "$PSScriptRoot\integration.ps1"
    }

    if ($args[0] -eq "validate") {
        Write-Host "Invoking validate.ps1"
        Invoke-Script -File "$PSScriptRoot\validate.ps1"
    }

    if ($args[0] -eq "test") {
        Write-Host "Invoking test.ps1"
        Invoke-Script -File "$PSScriptRoot\test.ps1"
    }

    if ($args[0] -eq "build") {
        Write-Host "Invoking build.ps1"
        Invoke-Script -File "$PSScriptRoot\build.ps1"
    }


} catch {
    Write-Host -NoNewline -ForegroundColor Red "[ERROR]: "
    Write-Host -ForegroundColor Red "Failed running $_"
    exit 1
}
