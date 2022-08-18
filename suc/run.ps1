<#
.SYNOPSIS 
    Upgrades Rancher Wins.
.DESCRIPTION 
    Run the script to upgrade all Rancher Wins related needs.
.NOTES

.EXAMPLE 
    
#>
param (
    [Parameter()]
    [Switch]
    $HostProcess
)

$ErrorActionPreference = 'Stop'

function New-Directory {
    param (
        [parameter(Mandatory = $false, ValueFromPipeline = $true)] [string]$Path
    )

    if (Test-Path -Path $Path) {
        if (-not (Test-Path -Path $Path -PathType Container)) {
            # clean the same path file
            Invoke-LogWrite -LogName Application -Source rancher-wins -Message "Existing path found on host, recursively cleaning $($Path)"
            Remove-Item -Recurse -Force -Path $Path -ErrorAction Ignore | Out-Null
        }
        return
    }
    Invoke-LogWrite -LogName Application -Source rancher-wins -Message "Creating $($Path)"
    New-Item -Force -ItemType Directory -Path $Path | Out-Null
}

function Invoke-Cleanup {
    param (
        [parameter(Mandatory = $false, ValueFromPipeline = $true)] [string]$Path
    )

    if (Test-Path -Path $Path) {
        Remove-Item -Recurse -Force -Path $Path -ErrorAction Ignore | Out-Null
    }
}

function Start-TransferFile {
    param (
        [parameter(Mandatory = $true)] [string]$Source,
        [parameter(Mandatory = $true)] [string]$Destination
    )

    if (Test-Path -PathType leaf -Path $Destination) {
        $destinationHasher = Get-FileHash -Path $Destination
        $sourceHasher = Get-FileHash -Path $Source
        if ($destinationHasher.Hash -eq $sourceHasher.Hash) {
            return
        }
    }

    Copy-Item -Force -Path $Source -Destination $Destination | Out-Null
    Invoke-LogWrite -LogName Application -Source rancher-wins -Message "Transferred file to $Destination from $Source"
}

function Invoke-WinsHostProcessUpgrade {
    $tmpdirbase = "/etc/rancher/agent"
    $tmpdir = Join-Path "C:\host\" -ChildPath $tmpdirbase

    New-Directory -Path $tmpdir

    Start-TransferFile -Source "C:\wins.exe" -Destination  $tmpdir
    Invoke-LogWrite -LogName Application -Source rancher-wins -Message "Transferred C:\wins.exe to $($tmpdir)"
    Start-TransferFile -Source "C:\install.ps1" -Destination $tmpdir
    Invoke-LogWrite -LogName Application -Source rancher-wins -Message "Transferred C:\install.ps1 to $($tmpdir)"

    if (-Not $env:CATTLE_ROLE_WORKER) {
        $env:CATTLE_ROLE_WORKER = "true"
    }

    $env:CATTLE_AGENT_BINARY_LOCAL = "true"
    $env:CATTLE_AGENT_BINARY_LOCAL_LOCATION = Join-Path -Path $tmpdir -ChildPath "wins.exe"

    Set-Location -Path $tmpdir

    Invoke-LogWrite -LogName Application -Source rancher-wins -Message "Running install.ps1 in $($tmpdir)"
    ./install.ps1

    Pop-Location

    Invoke-LogWrite -LogName Application -Source rancher-wins -Message "Successfully ran install.ps1, cleaning $($tmpdir)\wins.exe"
    Remove-Item -Force -Path $tmpdir\wins.exe -ErrorAction Ignore | Out-Null
    Invoke-LogWrite -LogName Application -Source rancher-wins -Message "Successfully ran install.ps1, cleaning $($tmpdir)\install.ps1"
    Remove-Item -Force -Path $tmpdir\install.ps1 -ErrorAction Ignore | Out-Null
    exit 0
}

function Invoke-LogWrite {
    param (
        [parameter(Mandatory = $true)] [string]$Message,
        [parameter(Mandatory = $true)] [string]$LogName,
        [parameter(Mandatory = $true)] [string]$Source
    )
    Write-Host "$($Message)"
    Write-EventLog -LogName "$LogName" -Source "$Source" -Message "$Message" -EID 1
}

function Invoke-WinsWinsUpgrade {
    $tmpdirbase = "/etc/rancher/wins"
    $tmpdir = Join-Path "C:\host\" -ChildPath $tmpdirbase
    $tmpdirLocal = Join-Path "C:\" -ChildPath $tmpdirbase
    $winsUpgradePath = Join-Path -Path $tmpdir -ChildPath "wins-upgrade.exe"
    $winsUpgradePathLocal = Join-Path -Path $tmpdirLocal -ChildPath "wins-upgrade.exe"

    New-Directory -Path $tmpdirLocal
    Invoke-LogWrite -LogName Application -Source rancher-wins -Message "Created new directory $($tmpdirLocal)"
    Copy-Item -Force -Path "C:\wins.exe" -Destination $winsUpgradePathLocal | Out-Null
    Invoke-LogWrite -LogName Application -Source rancher-wins -Message "Copied C:\wins.exe to $($winsUpgradePathLocal)"

    Invoke-LogWrite -LogName Application -Source rancher-wins -Message "Transferring file to host..."
    New-Directory -Path $winsUpgradePath
    Invoke-LogWrite -LogName Application -Source rancher-wins -Message "Created new directory $($winsUpgradePath)"
    Start-TransferFile -Source "C:\wins.exe" -Destination $winsUpgradePath

    Invoke-LogWrite -LogName Application -Source rancher-wins -Message "Checking if $($winsUpgradePath) exists"
    if(Test-Path $winsUpgradePath) {
        Invoke-LogWrite -LogName Application -Source rancher-wins -Message "$($winsUpgradePath) exists..."
    }
    else {
        Invoke-LogWrite -LogName Application -Source rancher-wins -Message "$($winsUpgradePath) was not copied to host..."
        exit 1
    }
    Invoke-LogWrite -LogName Application -Source rancher-wins -Message "preparing to run wins.exe upgrade using $($winsUpgradePathLocal)"

    $winsOut = wins.exe cli prc run --path=$winsUpgradePathLocal --args="up"

    Remove-Item -Recurse -Force -Path $winsUpgradePath -ErrorAction Ignore | Out-Null


    if ($winsOut -match ".* rpc error: code = Unavailable desc = transport is closing" -or ".* rpc error: code = Unavailable desc = error reading from server: EOF") {
        Invoke-LogWrite -LogName Application -Source rancher-wins -Message "Successfully upgraded"
        exit 0
    }
    elseif ($LastExitCode -ne 0) {
        Invoke-LogWrite -LogName Application -Source rancher-wins -Message "Returned exit $LastExitCode"
        Invoke-LogWrite -LogName Application -Source rancher-wins -Message $winsOut
        exit $LastExitCode
    }
    else {
        Invoke-LogWrite -LogName Application -Source rancher-wins -Message "Returned exit 0, but did not receive expected output from .\wins up"
        Invoke-LogWrite -LogName Application -Source rancher-wins -Message $winsOut
        exit 1
    }  
}

if($HostProcess) {
    Invoke-WinsHostProcessUpgrade
}
else {
    Invoke-WinsWinsUpgrade
}
