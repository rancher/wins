# bootstrap.ps1 prepares a regular Windows machine to run the plan e2e suite. It builds
# wins.exe and planctl.exe locally from this checked-out repo, writes a wins config that points
# at an external Kubernetes cluster for the plan Secret, and registers/starts rancher-wins
# directly, without going through install.ps1: there is no Rancher and no RKE2 involved.
#
# Safe to re-run: if rancher-wins is already installed, it is stopped, its binary replaced with
# a freshly built one, and restarted, so a rebuild-and-redeploy loop is just re-running this
# script.
#
# Usage: .\bootstrap.ps1 -Kubeconfig C:\path\to\external-cluster.kubeconfig
param(
    [parameter(Mandatory = $true)] [string]$Kubeconfig
)

$ErrorActionPreference = "Stop"

Import-Module -Name "$PSScriptRoot\planutils.psm1" -WarningAction Ignore -Force

$ServiceName = "rancher-wins"
$ConfigDir = "C:/etc/rancher/wins"
$VarDir = "C:/var/lib/rancher/agent"
$BinDir = "C:/usr/local/bin"
$RepoRoot = Resolve-Path (Join-Path $PSScriptRoot "..\..")

function Test-Elevated {
    $identity = [System.Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object System.Security.Principal.WindowsPrincipal($identity)
    return $principal.IsInRole([System.Security.Principal.WindowsBuiltInRole]::Administrator)
}

function Invoke-Preflight {
    Log-Info "Running preflight checks"

    if (-not (Test-Elevated)) {
        Log-Fatal "This script must be run from an elevated (Administrator) shell"
    }

    try {
        go version | Out-Null
    }
    catch {
        Log-Fatal "Go toolchain not found on PATH; it is required to build wins.exe and planctl.exe locally"
    }

    Log-Info "Preflight checks passed"
}

# Stop-ExistingService stops a prior rancher-wins installation, if one exists, so a freshly
# built wins.exe can be copied over its binary: Windows keeps a running executable's image file
# locked, so the copy in Install-WinsService would otherwise fail. Returns whether the service
# was already registered, so Install-WinsService knows whether registration can be skipped.
function Stop-ExistingService {
    $existing = Get-Service -Name $ServiceName -ErrorAction SilentlyContinue
    if (-not $existing) {
        return $false
    }

    if ($existing.Status -ne 'Stopped') {
        Log-Info "$ServiceName is already installed and running; stopping it to deploy the freshly built binary"
        Stop-Service -Name $ServiceName

        $timeout = 60
        $elapsed = 0
        while ((Get-Service $ServiceName).Status -ne 'Stopped') {
            if ($elapsed -ge $timeout) {
                Log-Fatal "$ServiceName did not stop within $timeout seconds"
            }
            Start-Sleep -s 2
            $elapsed += 2
        }
    }
    else {
        Log-Info "$ServiceName is already installed and stopped"
    }

    return $true
}

# Build-LocalBinaries builds wins.exe and planctl.exe on this machine from the checked-out repo,
# rather than shipping prebuilt artifacts: this is the machine the tests run on, so a native
# `go build` needs no cross-compilation.
function Build-LocalBinaries {
    Log-Info "Building wins.exe from $RepoRoot"
    Push-Location $RepoRoot
    try {
        $env:CGO_ENABLED = "0"
        go build -o (Join-Path $PSScriptRoot "wins.exe") ./cmd/
        if ($LASTEXITCODE -ne 0) {
            Log-Fatal "go build failed for wins.exe with exit code $LASTEXITCODE"
        }

        Log-Info "Building planctl.exe from $RepoRoot"
        go build -o (Join-Path $PSScriptRoot "planctl.exe") ./tests/plan/planctl
        if ($LASTEXITCODE -ne 0) {
            Log-Fatal "go build failed for planctl.exe with exit code $LASTEXITCODE"
        }
    }
    finally {
        Pop-Location
    }
    Log-Info "Local build complete"
}

# Install-WinsService writes the wins config directly, deploys the freshly built binary, and
# registers/starts rancher-wins, without going through install.ps1: there is no Rancher endpoint
# to simulate here, only an external Kubernetes cluster reachable via $Kubeconfig.
function Install-WinsService {
    param(
        [parameter(Mandatory = $true)] [string]$ConnectionInfoJson,
        [parameter(Mandatory = $true)] [bool]$AlreadyRegistered
    )

    New-Item -Path $ConfigDir -ItemType Directory -Force | Out-Null
    New-Item -Path $VarDir -ItemType Directory -Force | Out-Null
    New-Item -Path $BinDir -ItemType Directory -Force | Out-Null

    $connectionInfoPath = Join-Path $VarDir "rancher2_connection_info.json"
    Set-Content -Path $connectionInfoPath -Value $ConnectionInfoJson -NoNewline

    $config = @"
debug: true
systemagent:
  workDirectory: $VarDir/work
  appliedPlanDirectory: $VarDir/applied
  remoteEnabled: true
  localEnabled: false
  preserveWorkDirectory: false
  connectionInfoFile: $connectionInfoPath
"@
    Set-Content -Path "$ConfigDir/config" -Value $config

    Copy-Item -Path (Join-Path $PSScriptRoot "wins.exe") -Destination (Join-Path $BinDir "wins.exe") -Force

    if (-not $AlreadyRegistered) {
        Log-Info "Registering $ServiceName"
        Push-Location $BinDir
        try {
            .\wins.exe srv app run --register
            if ($LASTEXITCODE -ne 0) {
                Log-Fatal "wins.exe srv app run --register failed with exit code $LASTEXITCODE"
            }
        }
        finally {
            Pop-Location
        }
    }
    else {
        Log-Info "$ServiceName is already registered; reusing the existing registration and starting the freshly built binary"
    }

    Log-Info "Starting $ServiceName"
    Start-Service -Name $ServiceName

    $timeout = 60
    $elapsed = 0
    while ((Get-Service $ServiceName).Status -ne 'Running') {
        if ($elapsed -ge $timeout) {
            Log-Fatal "$ServiceName did not reach 'Running' state within $timeout seconds"
        }
        Start-Sleep -s 5
        $elapsed += 5
    }
}

function Invoke-SelfCheck {
    Log-Info "Running self-check: applying a trivial plan and waiting for it to succeed"
    Reset-Plan

    $plan = New-PlanSpec
    Add-PlanInstruction -Plan $plan -Name "self-check" -Command "cmd.exe" -Args @("/c", "exit 0")
    $planFile = Join-Path $env:TEMP "wins-plan-e2e-selfcheck.json"
    Write-PlanFile -Plan $plan -Path $planFile

    Invoke-PlanApply -PlanFile $planFile -State "pending" | Out-Null

    {
        (Get-PlanOutcome).planState -eq "succeeded"
    } | Judge -Timeout 60 -Throw

    Remove-Item -Path $planFile -Force -ErrorAction Ignore
    Log-Info "Self-check passed: $ServiceName successfully applied a plan"
}

Invoke-Preflight

Set-PlanKubeconfig -Path $Kubeconfig
$alreadyRegistered = Stop-ExistingService
Build-LocalBinaries

Log-Info "Bootstrapping the plan Secret, ServiceAccount, and RBAC via planctl against the external cluster"
# planctl bootstrap is idempotent and creates the namespace/Secret/RBAC on first use, so it
# doubles as the "can we reach the cluster" check: a connection failure here is reported with
# planctl's own error rather than a separate, redundant reachability probe.
try {
    $bootstrapResult = Invoke-PlanBootstrap
}
catch {
    Log-Fatal "planctl could not bootstrap against the cluster with the provided kubeconfig: $($_.Exception.Message)"
}

Install-WinsService -ConnectionInfoJson $bootstrapResult.connectionInfoJson -AlreadyRegistered $alreadyRegistered
Invoke-SelfCheck

Log-Info "Bootstrap complete. Run .\plan_suite_test.ps1 -Kubeconfig $Kubeconfig to execute the suite."
exit 0
