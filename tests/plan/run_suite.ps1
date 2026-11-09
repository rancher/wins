# run_suite.ps1 is the single entry point for the Windows plan e2e suite: it bootstraps
# (rebuilding wins.exe and planctl.exe from the current checkout every time, so a run always
# reflects the current code, and safely reusing an already-installed rancher-wins) and then runs
# the Pester suite. If every test passes, it stops and unregisters rancher-wins, leaving the
# machine clean; if any test fails, it leaves the installation in place so the failure can be
# diagnosed against a live service.
#
# Usage:
#   .\run_suite.ps1 -Kubeconfig C:\path\to\external-cluster.kubeconfig
#
# To re-run only specific tests, pass -TestName the same way plan_suite_test.ps1 accepts it:
#   .\run_suite.ps1 -Kubeconfig C:\path\to\external-cluster.kubeconfig -TestName "*C1*", "*P1*"

param(
    [parameter(Mandatory = $true)] [string]$Kubeconfig,
    [parameter(Mandatory = $false)] [string[]]$TestName
)

$ErrorActionPreference = "Stop"

Import-Module -Name "$PSScriptRoot\planutils.psm1" -WarningAction Ignore -Force

$ServiceName = "rancher-wins"
$BinDir = "C:/usr/local/bin"

function Uninstall-WinsService {
    Log-Info "All tests passed; stopping and uninstalling $ServiceName"

    Stop-Service -Name $ServiceName -ErrorAction SilentlyContinue

    Push-Location $BinDir
    try {
        .\wins.exe srv app run --unregister
        if ($LASTEXITCODE -ne 0) {
            Log-Warn "wins.exe srv app run --unregister failed with exit code $LASTEXITCODE"
        }
    }
    finally {
        Pop-Location
    }
}

& "$PSScriptRoot\bootstrap.ps1" -Kubeconfig $Kubeconfig
if ($LASTEXITCODE -ne 0) {
    # bootstrap.ps1's own Log-Fatal already named the failure; nothing useful can run after it.
    exit $LASTEXITCODE
}

$testArgs = @{ Kubeconfig = $Kubeconfig }
if ($TestName) {
    $testArgs.TestName = $TestName
}
& "$PSScriptRoot\plan_suite_test.ps1" @testArgs
$failedCount = $LASTEXITCODE

if ($failedCount -eq 0 -and -not $TestName) {
    Uninstall-WinsService
}
elseif ($failedCount -eq 0) {
    Log-Warn "All selected tests passed, but -TestName ran a subset of the suite; leaving $ServiceName installed rather than uninstalling on a partial run"
}
else {
    Log-Warn "$failedCount test(s) failed; leaving $ServiceName installed for diagnosis"
}

exit $failedCount
