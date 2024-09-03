$ErrorActionPreference = "Stop"

Import-Module -Name @(
    "$PSScriptRoot\utils.psm1"
) -WarningAction Ignore

Describe "SUC Config Updater" {
    BeforeEach {
        # create the config file
        $env:CATTLE_AGENT_CONFIG_DIR = "C:/etc/rancher/wins"
        $env:STRICT_VERIFY = $true
        Remove-Item -Force -Recurse $env:CATTLE_AGENT_CONFIG_DIR
        New-Item -Path $env:CATTLE_AGENT_CONFIG_DIR -ItemType Directory -ErrorAction Ignore | Out-Null
        Set-WinsConfig

        # register the service
        Log-Info "Adding rancher-wins service"
        $ret = .\bin\wins.exe srv app run --register
        if (-not $?) {
            Log-Error $ret
            $false | Should -Be $true
        }

        # verify
        Get-Service -Name rancher-wins -ErrorAction Ignore | Should -Not -BeNullOrEmpty
    }

    AfterEach {
        # Remove the config file
        rm env:CATTLE_WINS_DEBUG
        # remove the services
        $ret = .\bin\wins.exe srv app run --unregister
        if (-not $?) {
            Log-Error $ret
            $false | Should -Be $true
        }

        Stop-Service csiproxy -ErrorAction Ignore
        sc.exe delete csiproxy
        Stop-Process -Name csiproxy -ErrorAction Ignore
        Stop-Process -Name rancher-wins -ErrorAction Ignore
    }

    It "updates fields in config file" {
        $env:CATTLE_WINS_DEBUG="true"
        $env:STRICT_VERIFY="true"

        Execute-Binary -FilePath "bin\wins-suc.exe"

        Log-Info "Command exited successfully: $?"
        # Ensure command executed successfully by looking at the most recent exit code
        $? | Should -BeTrue
        $out = $(Get-Content $env:CATTLE_AGENT_CONFIG_DIR/config | out-string)
        Log-Info $out

        Log-Info "Confirming config was updated"
        # Ensure the debug flag is properly set in the config file
        $x = $out | select-string "debug: true"
        $x | Should -Not -BeNullOrEmpty
        $y = $out | select-string "agentStrictTLSMode: true"
        $y | Should -Not -BeNullOrEmpty
        Log-Info "Config updated successfully"
    }
}