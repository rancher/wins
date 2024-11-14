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
        $env:CATTLE_WINS_SKIP_BINARY_UPGRADE="true"

        Execute-Binary -FilePath "bin\wins-suc.exe"
        # Ensure command executed successfully by looking at the most recent exit code
        $LASTEXITCODE | Should -Be -ExpectedValue 0
        Log-Info "Command exited successfully: $?"
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

    It "enables rancher-wins delayed start" {
        Log-Info "TEST: Enabling rancher-wins delayed start"
        $env:CATTLE_ENABLE_WINS_DELAYED_START = "true"
        $env:CATTLE_WINS_SKIP_BINARY_UPGRADE = "true"

        Execute-Binary -FilePath "bin\wins-suc.exe"
        $LASTEXITCODE | Should -Be -ExpectedValue 0
        Log-Info "Command exited successfully: $($LASTEXITCODE -eq 0)"

        Log-Info (sc.exe qc "rancher-wins" | Out-String)
        $winsStartType = (sc.exe qc "rancher-wins" | Select-String "START_TYPE" | ForEach-Object { ($_ -replace '\s+', ' ').trim().Split(" ") | Select-Object -Last 1 })
        $winsStartType | Should -Be -ExpectedValue "(DELAYED)"
        Log-Info "Delayed start configured correctly"
    }

    It "disables rancher-wins delayed start" {
        Log-Info "TEST: Disabling rancher-wins delayed start"
        $env:CATTLE_ENABLE_WINS_DELAYED_START = "false"
        $env:CATTLE_WINS_SKIP_BINARY_UPGRADE = "true"

        Execute-Binary -FilePath "bin\wins-suc.exe"
        $LASTEXITCODE | Should -Be -ExpectedValue 0
        Log-Info "Command exited successfully: $?"

        Log-Info (sc.exe qc "rancher-wins" | Out-String)
        $winsStartType = (sc.exe qc "rancher-wins" | Select-String "START_TYPE" | ForEach-Object { ($_ -replace '\s+', ' ').trim().Split(" ") | Select-Object -Last 1 })
        $winsStartType | Should -Be -ExpectedValue "AUTO_START"
        Log-Info "Delayed start configured correctly"
    }

    It "enables rke2 service dependency" {
        Log-Info "TEST: Enabling rke2 service dependency"
        $env:CATTLE_ENABLE_WINS_SERVICE_DEPENDENCY = "true"
        $env:CATTLE_WINS_SKIP_BINARY_UPGRADE = "true"

        Execute-Binary -FilePath "bin\wins-suc.exe"
        $LASTEXITCODE | Should -Be -ExpectedValue 0
        Log-Info "Command exited successfully: $($LASTEXITCODE -eq 0)"

        Log-Info (sc.exe qc rke2 | Out-String)
        $dependencies = (Get-Service -Name rke2).ServicesDependedOn
        Log-Info "Confirming rancher-wins service dependency has been added..."
        $found = Ensure-DependencyExistsForService -ServiceName rke2 -DependencyName rancher-wins
        $found | Should -BeTrue
        Log-Info "Service dependency correctly configured"
    }

    It "disables rke2 service dependency" {
        Log-Info "TEST: Disabling rke2 service dependency"
        # Dependency will be disabled whenever CATTLE_ENABLE_WINS_SERVICE_DEPENDENCY is
        # set to a value other than "true" (including not being set at all)
        $env:CATTLE_ENABLE_WINS_SERVICE_DEPENDENCY = ""
        $env:CATTLE_WINS_SKIP_BINARY_UPGRADE = "true"

        Execute-Binary -FilePath "bin\wins-suc.exe"
        $LASTEXITCODE | Should -Be -ExpectedValue 0
        Log-Info "Command exited successfully: $($LASTEXITCODE -eq 0)"

        Log-Info (sc.exe qc rke2 | Out-String)
        $dependencies = (Get-Service -Name rke2).ServicesDependedOn
        Log-Info "Confirming rancher-wins service dependency has been removed..."
        $found = Ensure-DependencyExistsForService -ServiceName rke2 -DependencyName rancher-wins
        $found | Should -BeFalse
        $dependencies.Count | Should -Be -ExpectedValue 0
        Log-Info "Service dependency correctly removed"
    }

    It "Updates rancher-wins config file while rke2 dependency exists" {
        Log-Info "TEST: Enabling rke2 service dependency"
        $env:CATTLE_ENABLE_WINS_SERVICE_DEPENDENCY = "true"
        $env:CATTLE_WINS_SKIP_BINARY_UPGRADE = "true"

        Execute-Binary -FilePath "bin\wins-suc.exe"
        $LASTEXITCODE | Should -Be -ExpectedValue 0
        Log-Info "Command exited successfully: $($LASTEXITCODE -eq 0)"

        Log-Info (sc.exe qc rke2 | Out-String)
        Log-Info "Confirming rancher-wins service dependency has been added..."
        $found = Ensure-DependencyExistsForService -ServiceName rke2 -DependencyName rancher-wins
        $found | Should -BeTrue
        Log-Info "Service dependency correctly configured"

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
