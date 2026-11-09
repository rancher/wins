# plan_suite_test.ps1 runs the Windows plan e2e Pester suite.
#
# Usage: .\plan_suite_test.ps1 -Kubeconfig C:\path\to\admin.kubeconfig
#
# To re-run only specific tests (e.g. ones that just failed), pass -TestName with one or more
# wildcard patterns matched against each test's full name ("<Describe name>.<It name>"), for
# example:
#   .\plan_suite_test.ps1 -Kubeconfig C:\path\to\admin.kubeconfig -TestName "*C1*", "*C4*"
param(
    [parameter(Mandatory = $true)] [string]$Kubeconfig,
    [parameter(Mandatory = $false)] [string[]]$TestName
)

$ErrorActionPreference = "Stop"

# Windows ships Pester 3.4.0 built in, which has neither New-PesterContainer nor
# -Configuration; both are Pester 5+ only. Force-load a Pester 5+ module explicitly rather than
# relying on whatever autoloads, and fail with an actionable message if none is installed.
try {
    Import-Module -Name Pester -MinimumVersion 5.0.0 -ErrorAction Stop
}
catch {
    Write-Error "Pester 5.0.0 or newer is required but not installed.
    Install it with: Install-Module -Name Pester -MinimumVersion 5.0.0 -Force -SkipPublisherCheck -Scope CurrentUser"
    exit 1
}

# Exported so specs picked up by Invoke-Pester's container list can read it without importing
# planutils.psm1 themselves; each spec file imports planutils.psm1 directly and calls
# Set-PlanKubeconfig in its own BeforeAll.
$env:WINS_PLAN_E2E_KUBECONFIG = $Kubeconfig

$specFiles = @(
    "cancellation_test.ps1",
    "pause_test.ps1",
    "failure_handling_test.ps1",
    "instruction_execution_test.ps1",
    "periodic_probe_test.ps1",
    "file_operations_test.ps1"
) | ForEach-Object { Join-Path $PSScriptRoot $_ }

$containers = $specFiles | ForEach-Object { New-PesterContainer -Path $_ }

$configuration = [PesterConfiguration]::Default
$configuration.Run.Container = $containers
$configuration.Run.PassThru = $true
$configuration.Output.Verbosity = "Detailed"
if ($TestName) {
    $configuration.Filter.FullName = $TestName
}

$result = Invoke-Pester -Configuration $configuration

exit $result.FailedCount
