<#
.SYNOPSIS
    Uninstalls Rancher Wins from Windows Worker Nodes.
.DESCRIPTION
    Run the script to uninstall all Rancher Wins related components.
    This script will abort if RKE2 is currently running, as wins is primarily
    used in the Rancher provisioning process and uninstalling while RKE2 is
    active may leave the node in an inconsistent state.
.NOTES
    Environment variables:
      System Agent Variables
      - CATTLE_AGENT_BIN_PREFIX (default: c:/usr/local)
      - CATTLE_AGENT_CONFIG_DIR (default: C:/etc/rancher/agent)
      - CATTLE_AGENT_VAR_DIR (default: C:/var/lib/rancher/agent)
.EXAMPLE
    ./uninstall.ps1
#>

$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

function Invoke-WinsUninstaller {
    [CmdletBinding()]
    param ()

    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls -bor [Net.SecurityProtocolType]::Tls11 -bor [Net.SecurityProtocolType]::Tls12

    function Write-LogInfo {
        Write-Host -NoNewline -ForegroundColor Blue "INFO: "
        Write-Host -ForegroundColor Gray ("{0,-44}" -f ($args -join " "))
    }
    function Write-LogWarn {
        Write-Host -NoNewline -ForegroundColor DarkYellow "WARN: "
        Write-Host -ForegroundColor Gray ("{0,-44}" -f ($args -join " "))
    }
    function Write-LogError {
        Write-Host -NoNewline -ForegroundColor DarkRed "ERROR: "
        Write-Host -ForegroundColor Gray ("{0,-44}" -f ($args -join " "))
    }
    function Write-LogFatal {
        Write-Host -NoNewline -ForegroundColor DarkRed "FATA: "
        Write-Host -ForegroundColor Gray ("{0,-44}" -f ($args -join " "))
        exit 255
    }

    function Set-Path {
        $env:PATH += ";C:\var\lib\rancher\rke2\bin;C:\usr\local\bin"
        $environment = [System.Environment]::GetEnvironmentVariable("Path", "Machine")
        $environment = $environment.Insert($environment.Length, ";C:\var\lib\rancher\rke2\bin;C:\usr\local\bin")
        [System.Environment]::SetEnvironmentVariable("Path", $environment, "Machine")
    }

    function Set-Environment {
        if (-Not $env:CATTLE_AGENT_LOGLEVEL) {
            $env:CATTLE_AGENT_LOGLEVEL = "debug"
        }
        else {
            $env:CATTLE_AGENT_LOGLEVEL = $env:CATTLE_AGENT_LOGLEVEL.ToLower()
        }

        if (-Not $env:CATTLE_AGENT_CONFIG_DIR) {
            $env:CATTLE_AGENT_CONFIG_DIR = "C:/etc/rancher/wins"
            Write-LogInfo "Using default agent configuration directory $( $env:CATTLE_AGENT_CONFIG_DIR )"
        }
        if (-Not (Test-Path $env:CATTLE_AGENT_CONFIG_DIR)) {
            New-Item -Path $env:CATTLE_AGENT_CONFIG_DIR -ItemType Directory -Force | Out-Null
        }

        if (-Not $env:CATTLE_AGENT_VAR_DIR) {
            $env:CATTLE_AGENT_VAR_DIR = "C:/var/lib/rancher/agent"
            Write-LogInfo "Using default agent var directory $( $env:CATTLE_AGENT_VAR_DIR )"
        }
        if (-Not (Test-Path $env:CATTLE_AGENT_VAR_DIR)) {
            New-Item -Path $env:CATTLE_AGENT_VAR_DIR -ItemType Directory -Force | Out-Null
        }

        if (-Not $env:CATTLE_AGENT_BIN_PREFIX) {
            $env:CATTLE_AGENT_BIN_PREFIX = "c:/usr/local"
        }
        if (-Not (Test-Path $env:CATTLE_AGENT_BIN_PREFIX)) {
            New-Item -Path $env:CATTLE_AGENT_BIN_PREFIX -ItemType Directory -Force | Out-Null
        }
    }

    # Safeguard: abort uninstall if RKE2 is currently running.
    # Wins is a core component of the Rancher/RKE2 provisioning process on Windows.
    # Uninstalling while RKE2 is active risks leaving the node in a broken state.
    function Assert-Rke2NotRunning {
        Write-LogInfo "Checking if RKE2 is running"
        $rke2Service = Get-Service -Name "rke2-agent" -ErrorAction SilentlyContinue
        if ($rke2Service -and $rke2Service.Status -eq 'Running') {
            Write-LogFatal "RKE2 service is currently running. Stop RKE2 before uninstalling Wins to avoid leaving the node in an inconsistent state."
        }
        $rke2Process = Get-Process -Name "rke2-agent" -ErrorAction SilentlyContinue
        if ($rke2Process) {
            Write-LogFatal "RKE2 process is currently running. Stop RKE2 before uninstalling Wins to avoid leaving the node in an inconsistent state."
        }
        Write-LogInfo "RKE2 is not running, proceeding with uninstall"
    }

    function Remove-WinsConfig() {
        Remove-Item -Path $env:CATTLE_AGENT_CONFIG_DIR -Recurse -Force
        Remove-Item -Path $env:CATTLE_AGENT_VAR_DIR -Recurse -Force
        if (Test-Path "C:/etc/windows-exporter") {
            Remove-Item -Path "C:/etc/wmi-exporter" -Recurse -Force
        }
        if (Test-Path "C:/etc/wmi-exporter") {
            Remove-Item -Path "C:/etc/windows-exporter" -Recurse -Force
        }
    }

    function Stop-Agent() {
        [CmdletBinding()]
        param (
            [Parameter()]
            [string]
            $ServiceName
        )
        Write-LogInfo "Checking if $ServiceName service exists"
        if ((Get-Service -Name $ServiceName -ErrorAction SilentlyContinue)) {
            Write-LogInfo "$ServiceName service found, stopping now"
            Stop-Service -Name $ServiceName
            while ((Get-Service -Name $ServiceName).Status -ne 'Stopped') {
                Write-LogInfo "Waiting for $ServiceName service to stop"
                Start-Sleep -s 20
            }
        }
        else {
            Write-LogInfo "$ServiceName isn't installed, continuing"
        }
    }

    # Explicitly stops the csi-proxy process by name before attempting service removal.
    # The service deletion can fail on first run if the process is still running after
    # Stop-Service returns, since the process may take a moment to fully exit.
    function Stop-CsiProxyProcess() {
        Write-LogInfo "Checking if csi-proxy process is running"
        $csiProxyProcess = Get-Process -Name "csi-proxy" -ErrorAction SilentlyContinue
        if ($csiProxyProcess) {
            Write-LogInfo "csi-proxy process found (PID $($csiProxyProcess.Id)), stopping now"
            Stop-Process -Name "csi-proxy" -Force
            # Wait for the process to fully exit before proceeding with service deletion
            $csiProxyProcess.WaitForExit(30000) | Out-Null
            Write-LogInfo "csi-proxy process stopped"
        }
        else {
            Write-LogInfo "csi-proxy process is not running, continuing"
        }
    }

    function Remove-WinsForCharts() {
        $winsForChartsPath = "c:/windows"
        if (Test-Path "$winsForChartsPath/wins.exe") {
            Remove-Item "$winsForChartsPath/wins.exe" -Force
        }
    }

    function Invoke-WinsAgentUninstall() {
        $serviceName = "rancher-wins"
        $csiProxyServiceName = "csiproxy"
        Set-Environment
        Set-Path

        Assert-Rke2NotRunning

        Stop-Agent -ServiceName $serviceName
        Stop-Agent -ServiceName $csiProxyServiceName
        Stop-CsiProxyProcess
        Remove-WinsForCharts
        Remove-WinsConfig
        sc.exe delete $serviceName
        sc.exe delete $csiProxyServiceName
    }

    Invoke-WinsAgentUninstall
}

Invoke-WinsUninstaller