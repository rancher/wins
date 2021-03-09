#Requires -Version 5.0

$ErrorActionPreference = 'Stop'

Import-Module -WarningAction Ignore -Name "$PSScriptRoot\utils.psm1"

Invoke-Script -File "$PSScriptRoot\test.ps1"
Invoke-Script -File "$PSScriptRoot\build.ps1"

Invoke-Expression -Command "$PSScriptRoot\package.ps1"
