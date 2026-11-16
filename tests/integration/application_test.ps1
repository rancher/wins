$ErrorActionPreference = "Stop"
$serviceName = "rancher-wins"

Import-Module -Name @(
    "$PSScriptRoot\utils.psm1"
) -WarningAction Ignore

# clean interferences
try {
    Get-Process -Name "wins" -ErrorAction Ignore | Stop-Process -Force -ErrorAction Ignore
}
catch {
    Log-Warn $_.Exception.Message
}

Describe "application" {
    It "register" {
        $ret = .\bin\wins.exe srv app run --register
        if (-not $?) {
            Log-Error $ret
            $false | Should -Be $true
        }

        # verify
        Get-Service -Name $serviceName -ErrorAction Ignore | Should -Not -BeNullOrEmpty
    }

    It "unregister" {
        $ret = .\bin\wins.exe srv app run --unregister
        if (-not $?) {
            Log-Error $ret
            $false | Should -Be $true
        }

        # verify
        Get-Service -Name $serviceName -ErrorAction Ignore | Should -BeNullOrEmpty
    }
}
