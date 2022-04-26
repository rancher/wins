<#
.SYNOPSIS 
    Installs Rancher Wins to create Windows Worker Nodes.
.DESCRIPTION 
    Run the script to install all Rancher Wins related needs.
.NOTES
    Environment variables:
      System Agent Variables
      - CATTLE_AGENT_BIN_PREFIX (default: c:/usr/local)
      - CATTLE_AGENT_CONFIG_DIR (default: C:/etc/rancher/agent)
      - CATTLE_AGENT_VAR_DIR (default: C:/var/lib/rancher/agent)     
.EXAMPLE 
    
#>
#Make sure this params matches the CmdletBinding below
param (
    [Parameter()]
    [String]
    $Address,
    [Parameter()]
    [String]
    $CaChecksum,
    [Parameter()]
    [String]
    $InternalAddress,
    [Parameter()]
    [String]
    $Label,
    [Parameter()]
    [String]
    $NodeName,
    [Parameter()]
    [String]
    $Server,
    [Parameter()]
    [String]
    $Taint,
    [Parameter()]
    [String]
    $Token,
    [Parameter()]
    [Switch]
    $Worker
)
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

function Invoke-WinsUninstaller {
    [CmdletBinding()]
    param (
        [Parameter()]
        [String]
        $Address,
        [Parameter()]
        [String]
        $CaChecksum,
        [Parameter()]
        [String]
        $InternalAddress,
        [Parameter()]
        [String]
        $Label,
        [Parameter()]
        [String]
        $NodeName,
        [Parameter()]
        [String]
        $Server,
        [Parameter()]
        [String]
        $Taint,
        [Parameter()]
        [String]
        $Token,
        [Parameter()]
        [Switch]
        $Worker
    )

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

        Stop-Agent -ServiceName $serviceName
        Stop-Agent -ServiceName $csiProxyServiceName
        Remove-WinsForCharts
        Remove-WinsConfig
        sc.exe delete $serviceName
        sc.exe delete $csiProxyServiceName
    }

    Invoke-WinsAgentUninstall
    if (Test-Path $env:CATTLE_AGENT_BIN_PREFIX/bin/rke2-uninstall.ps1) {
        . $env:CATTLE_AGENT_BIN_PREFIX/bin/rke2-uninstall.ps1
    }
}

Invoke-WinsUninstaller