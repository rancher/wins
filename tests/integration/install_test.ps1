$ErrorActionPreference = "Stop"

Import-Module -Name @(
    "$PSScriptRoot\utils.psm1"
) -WarningAction Ignore

# clean interferences
try {
    Get-Process -Name "rancher-wins-*" -ErrorAction Ignore | Stop-Process -Force -ErrorAction Ignore
    Get-NetFirewallRule -PolicyStore ActiveStore -Name "rancher-wins-*" -ErrorAction Ignore | ForEach-Object { Remove-NetFirewallRule -Name $_.Name -PolicyStore ActiveStore -ErrorAction Ignore } | Out-Null
    Get-Process -Name "wins" -ErrorAction Ignore | Stop-Process -Force -ErrorAction Ignore
}
catch {
    Log-Warn $_.Exception.Message
}

Describe "install" {
    BeforeEach {
        # note: we cannot test system agent install today since we need a mocked API server
        Log-Info "Running install script"
        # note: Simply running the install script does not do anything. During normal provisioning,
        # Rancher will mutate the install script to both add environment variables, and to call
        # the primary function 'Invoke-WinsInstaller'. As this is an integration test, we need to manually
        # update the install script ourselves.
        Add-Content -Path ./install.ps1 -Value '$env:CATTLE_REMOTE_ENABLED = "false"'
        Add-Content -Path ./install.ps1 -Value '$env:CATTLE_LOCAL_ENABLED = "true"'
        Add-Content -Path ./install.ps1 -Value Invoke-WinsInstaller

        .\install.ps1
    }

    AfterEach {
        Log-Info "Running uninstall script"
        try {
            # note: since this script may not be run by an administrator, it's possible that it might fail
            # on trying to delete certain files with ACLs attached to them.
            # If you are running this locally, make sure you run with admin privileges.
            .\uninstall.ps1
        } catch {
            Log-Warn "You need to manually run uninstall.ps1, encountered error: $($_.Exception.Message)"
        }
    }

    It "creates files and directories with scoped down permissions" {
        # While these get set in install.ps1, pester removes them as
        # install.ps1 is called in the BeforeEach block
        $env:CATTLE_AGENT_VAR_DIR = "c:/var/lib/rancher/agent"
        $env:CATTLE_AGENT_CONFIG_DIR = "c:/etc/rancher/wins"

        $restrictedPaths = @(
            $env:CATTLE_AGENT_VAR_DIR,
            $env:CATTLE_AGENT_CONFIG_DIR,
            "$env:CATTLE_AGENT_CONFIG_DIR/config"

        # TODO: to test the creation of rancher2_connection_info.json, we need to mock the Rancher server.
        # Once this capability is added to tests, uncomment this and remove $env:CATTLE_REMOTE_ENABLED = "false" above.
        # "$env:CATTLE_AGENT_VAR_DIR/rancher2_connection_info.json"
        )
        foreach ($path in $restrictedPaths) {
            Log-Info "Checking $path"

            Test-Path -Path $path | Should -Be $true

            Test-Permissions -Path $path -ExpectedOwner "BUILTIN\Administrators" -ExpectedGroup "NT AUTHORITY\SYSTEM" -ExpectedPermissions @(
                [PSCustomObject]@{
                    AccessMask = "FullControl"
                    Type = 0
                    Identity = "NT AUTHORITY\SYSTEM"
                },
                [PSCustomObject]@{
                    AccessMask = "FullControl"
                    Type = 0
                    Identity = "BUILTIN\Administrators"
                }
            )

            Log-Info "Confirmed expected ACLs on $path"
        }
    }
}