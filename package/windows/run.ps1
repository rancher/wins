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
            Remove-Item -Recurse -Force -Path $Path -ErrorAction Ignore | Out-Null
        }
        return
    }

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
}

function Invoke-WinsHostProcessUpgrade {
    $tmpdirbase = "/etc/rancher/agent"
    $tmpdir = Join-Path "C:\host\" -ChildPath $tmpdirbase

    New-Directory -Path $tmpdir

    Start-TransferFile -Source "C:\wins.exe" -Destination  $tmpdir
    Start-TransferFile -Source "C:\install.ps1" -Destination $tmpdir

    if (-Not $env:CATTLE_ROLE_WORKER) {
        $env:CATTLE_ROLE_WORKER = "true"
    }

    $env:CATTLE_AGENT_BINARY_LOCAL = "true"
    $env:CATTLE_AGENT_BINARY_LOCAL_LOCATION = Join-Path -Path $tmpdir -ChildPath "wins.exe"

    Set-Location -Path $tmpdir

    ./install.ps1

    Pop-Location

    Remove-Item -Path $tmpdir\wins.exe
    Remove-Item -Path $tmpdir\install.ps1
    exit 0
}

function Invoke-WinsWinsUpgrade {
    $tmpdirbase = "/etc/rancher/wins"
    $tmpdir = Join-Path "C:\host\" -ChildPath $tmpdirbase
    $tmpdirLocal = Join-Path "C:\" -ChildPath $tmpdirbase
    $winsUpgradePath = Join-Path -Path $tmpdir -ChildPath "wins-upgrade.exe"
    $winsUpgradePathLocal = Join-Path -Path $tmpdirLocal -ChildPath "wins-upgrade.exe"

    New-Directory -Path $tmpdirLocal
    Copy-Item -Force -Path "C:\wins.exe" -Destination $winsUpgradePathLocal | Out-Null

    Write-Host "Transferring file to host..."
    Start-TransferFile -Source "C:\wins.exe" -Destination $winsUpgradePath

    Write-Host "Checking if $($winsUpgradePath) exists"
    if(Test-Path $winsUpgradePath) {
        Write-Host "$($winsUpgradePath) exists..."
    }
    else {
        Write-Host "$($winsUpgradePath) was not copied to host..."
        exit 1
    }

    $winsOut = wins.exe cli prc run --path=$winsUpgradePathLocal --args="up"

    Remove-Item -Path $winsUpgradePath

    if ($winsOut -match ".* rpc error: code = Unavailable desc = transport is closing") {
        Write-Host "Successfully upgraded"
        exit 0
    }
    elseif ($LastExitCode -ne 0) {
        Write-Host "Returned exit $LastExitCode"
        Write-Host $winsOut
        exit $LastExitCode
    }
    else {
        Write-Host "Returned exit 0, but did not receive expected output from .\wins up"
        Write-Host $winsOut
        exit 1
    }  
}

if($HostProcess) {
    Invoke-WinsHostProcessUpgrade
}
else {
    Invoke-WinsWinsUpgrade
}
