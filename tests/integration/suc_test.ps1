$ErrorActionPreference = "Stop"

Import-Module -Name @(
    "$PSScriptRoot\utils.psm1"
) -WarningAction Ignore

Describe "SUC rancher-wins Config File Updater" {

    BeforeAll {
        Add-RancherWinsService
        # Start the service
        Log-Info "Starting rancher-wins service"
        try
        {
            Start-Service rancher-wins
        } catch
        {
            Log-Info "rancher-wins failed to start"
            Log-Info (cat C:/etc/rancher/wins/config)
            Log-Info (Get-winevent -providername rancher-wins | select-object message | format-table -wrap)
            throw
        }
    }

    AfterAll {
        Remove-RancherWinsService
    }

    It "updates fields in config file" {
        $env:CATTLE_WINS_DEBUG="true"
        $env:STRICT_VERIFY="true"

        Execute-Binary -FilePath "bin\wins-suc.exe"

        Log-Info "Command exited successfully: $?"
        # Ensure command executed successfully by looking at the most recent exit code
        $LASTEXITCODE | Should -Be -ExpectedValue 0
        $out = $(Get-Content $env:CATTLE_AGENT_CONFIG_DIR/config | out-string)
        Log-Info $out

        Log-Info "Confirming config was updated"
        # Ensure the debug flag is properly set in the config file
        $out | select-string "debug: true" | Should -Not -BeNullOrEmpty
        $out | select-string "agentStrictTLSMode: true" | Should -Not -BeNullOrEmpty
        Log-Info "Config updated successfully"
    }
}

Describe "SUC rancher-wins Service Configurator" {
    BeforeAll {
        Add-RancherWinsService
        Add-DummyRKE2Service
    }

    AfterAll {
        Remove-RancherWinsService
        Remove-DummyRKE2Service
    }

    It "Updates rancher-wins config file while rke2 dependency exists" {
        Log-Info "TEST: Updating rancher-wins config file with an existing rke2 service dependency"
        Add-RKE2WinsDependency

        Log-Info "Updating rancher-wins config file"
        $env:CATTLE_WINS_DEBUG="false"
        Execute-Binary -FilePath "bin\wins-suc.exe"

        Log-Info "Command exited successfully: $?"
        $LASTEXITCODE | Should -Be -ExpectedValue 0

        $out = $(Get-Content $env:CATTLE_AGENT_CONFIG_DIR/config | out-string)
        Log-Info $out
        Log-Info "Confirming config was updated"
        # Ensure the debug flag is properly set in the config file
        $out | select-string "debug: true" | Should -BeNullOrEmpty
        Log-Info "Config updated successfully"

        Log-Info "Ensuring that rke2 service dependency still exists"
        Log-Info (sc.exe qc rke2 | Out-String)
        $found = Ensure-DependencyExistsForService -ServiceName rke2 -DependencyName rancher-wins
        $found | Should -BeTrue
        Log-Info "Service dependency correctly configured"
    }
}
