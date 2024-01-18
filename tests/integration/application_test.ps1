$ErrorActionPreference = "Stop"

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
    $serviceName = "rancher-wins"

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

    Context "upgrade" {
        BeforeEach {
            .\bin\wins.exe srv app run --register
            Start-Service -Name $serviceName | Out-Null
            Wait-Ready -Path //./pipe/rancher_wins
        }

        AfterEach {
            Stop-Service -Name $serviceName | Out-Null
            .\bin\wins.exe srv app run --unregister
        }

        It "watching" {
            # docker run --rm -v //./pipe/rancher_wins://./pipe/rancher_wins -v c:/etc/rancher/wins:c:/etc/rancher/wins wins-upgrade
            $ret = Execute-Binary -FilePath "docker.exe" -ArgumentList @("run", "--rm", "-v", "//./pipe/rancher_wins://./pipe/rancher_wins", "-v", "c:/etc/rancher/wins:c:/etc/rancher/wins", "wins-upgrade") -PassThru
            if (-not $ret.Ok) {
                Log-Error $ret.Output
                $false | Should -Be $true
            }

            #verify
            $expectedObj = $ret.Output | ConvertFrom-Json
            $expectedObj.Server.Version | Should -Be "container"
            $expectedObj.Server.Commit | Should -Be "container"
        }
    }

}
