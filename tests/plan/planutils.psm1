# planutils.psm1 provides plan-authoring helpers and thin wrappers around planctl.exe for the
# Windows plan e2e suite.
# The suite always runs from within a checked-out repo on the Windows machine (bootstrap.ps1
# builds wins.exe and planctl.exe locally there), so utils.psm1 is always reachable at its normal
# repo-relative path.
Import-Module -Name "$PSScriptRoot\..\integration\utils.psm1" -WarningAction Ignore -Force

$script:PlanCtl = Join-Path $PSScriptRoot "planctl.exe"
$script:Kubeconfig = $null

function Set-PlanKubeconfig {
    param(
        [parameter(Mandatory = $true)] [string]$Path
    )
    $script:Kubeconfig = $Path
}

function Invoke-PlanCtl {
    # PlanCtlArgs is bound explicitly by name rather than via ValueFromRemainingArguments, and
    # marked AllowEmptyString: PowerShell's Mandatory validation on a string[] parameter rejects
    # an empty string value in ANY element by default (not just the array as a whole), which
    # annotate's "clear this annotation" call (-Canceled "") depends on being able to send as
    # one of this array's elements.
    param(
        [parameter(Mandatory = $true)]
        [AllowEmptyString()]
        [string[]]$PlanCtlArgs
    )
    if (-not $script:Kubeconfig) {
        throw "Set-PlanKubeconfig must be called before Invoke-PlanCtl"
    }
    $allArgs = @("--kubeconfig", $script:Kubeconfig) + $PlanCtlArgs
    $output = & $script:PlanCtl @allArgs 2>&1
    if ($LASTEXITCODE -ne 0) {
        throw "planctl $($PlanCtlArgs -join ' ') failed with exit code ${LASTEXITCODE}: $output"
    }
    return $output
}

function Invoke-PlanBootstrap {
    $json = Invoke-PlanCtl -PlanCtlArgs @("bootstrap")
    return $json | ConvertFrom-Json
}

function Invoke-PlanApply {
    param(
        [parameter(Mandatory = $true)] [string]$PlanFile,
        [parameter(Mandatory = $false)] [string]$State = "pending",
        # Setting Canceled or Paused here writes the corresponding annotation in the same
        # planctl invocation and the same Secret Update as the plan content, so the agent can
        # never observe an intermediate version that is pending but not yet interrupted. A
        # separate Invoke-PlanAnnotate call afterward cannot guarantee that: it is a second
        # process invocation, and the agent can finish a trivial instruction before that second
        # process even starts up.
        [parameter(Mandatory = $false)] [string]$Canceled,
        [parameter(Mandatory = $false)] [string]$Paused
    )
    $planCtlArgs = @("apply-plan", "--file", $PlanFile, "--state", $State)
    if ($PSBoundParameters.ContainsKey("Canceled")) {
        $planCtlArgs += "--canceled=$Canceled"
    }
    if ($PSBoundParameters.ContainsKey("Paused")) {
        $planCtlArgs += "--paused=$Paused"
    }
    return (Invoke-PlanCtl -PlanCtlArgs $planCtlArgs | Out-String).Trim()
}

function Invoke-PlanAnnotate {
    param(
        [parameter(Mandatory = $false)] [string]$Canceled,
        [parameter(Mandatory = $false)] [string]$Paused
    )
    # Uses the combined "--flag=value" form rather than two separate array elements
    # ("--flag", $value): when invoking a native executable (not a PowerShell function) via &,
    # PowerShell silently drops an array element that is an empty string before the argument
    # ever reaches the process, so "--canceled", "" arrives at planctl.exe as a bare "--canceled"
    # with no value at all. "--canceled=" is a single non-empty token, so it can never be
    # dropped, and Go's flag package (which urfave/cli wraps) parses "--flag=value" natively,
    # including an empty value after "=".
    $planCtlArgs = @()
    if ($PSBoundParameters.ContainsKey("Canceled")) {
        $planCtlArgs += "--canceled=$Canceled"
    }
    if ($PSBoundParameters.ContainsKey("Paused")) {
        $planCtlArgs += "--paused=$Paused"
    }
    Invoke-PlanCtl -PlanCtlArgs (@("annotate") + $planCtlArgs) | Out-Null
}

function Get-PlanOutcome {
    $json = Invoke-PlanCtl -PlanCtlArgs @("outcome")
    return $json | ConvertFrom-Json
}

function Reset-Plan {
    Invoke-PlanCtl -PlanCtlArgs @("reset") | Out-Null
}

# New-PlanSpec creates an empty plan hashtable, mirroring planapi.Plan
# (github.com/rancher/rancher/pkg/plan): files, instructions (one-time), probes,
# periodicInstructions.
function New-PlanSpec {
    return @{
        files        = @()
        instructions = @()
    }
}

# Add-PlanFile appends a file entry to the plan, base64-encoding Content, mirroring
# framework.PlanBuilder.WithFile in system-agent's e2e suite.
function Add-PlanFile {
    param(
        [parameter(Mandatory = $true)] [hashtable]$Plan,
        [parameter(Mandatory = $true)] [string]$Path,
        [parameter(Mandatory = $false)] [string]$Content = "",
        [parameter(Mandatory = $false)] [string]$Permissions,
        [parameter(Mandatory = $false)] [switch]$Directory,
        [parameter(Mandatory = $false)] [switch]$Delete
    )
    $file = @{
        path = $Path
    }
    if ($Directory) {
        $file.directory = $true
    }
    else {
        $bytes = [System.Text.Encoding]::UTF8.GetBytes($Content)
        $file.content = [System.Convert]::ToBase64String($bytes)
    }
    if ($Permissions) {
        $file.permissions = $Permissions
    }
    if ($Delete) {
        $file.action = "delete"
    }
    $Plan.files += , $file
}

# Add-PlanInstruction appends a one-time instruction to the plan. Command must always be set
# explicitly: applyinator's defaultCommand ("/run.sh") has no Windows form.
function Add-PlanInstruction {
    param(
        [parameter(Mandatory = $true)] [hashtable]$Plan,
        [parameter(Mandatory = $true)] [string]$Name,
        [parameter(Mandatory = $true)] [string]$Command,
        [parameter(Mandatory = $false)] [string[]]$Args = @(),
        [parameter(Mandatory = $false)] [string[]]$Env = @(),
        [parameter(Mandatory = $false)] [switch]$SaveOutput
    )
    $instruction = @{
        name    = $Name
        command = $Command
        args    = $Args
        env     = $Env
    }
    if ($SaveOutput) {
        $instruction.saveOutput = $true
    }
    $Plan.instructions += , $instruction
}

# Add-PlanPeriodicInstruction appends a periodic instruction to the plan, mirroring
# planapi.PeriodicInstruction: CommonInstruction plus periodSeconds and saveStderrOutput.
# Command must always be set explicitly, for the same reason as Add-PlanInstruction.
function Add-PlanPeriodicInstruction {
    param(
        [parameter(Mandatory = $true)] [hashtable]$Plan,
        [parameter(Mandatory = $true)] [string]$Name,
        [parameter(Mandatory = $true)] [string]$Command,
        [parameter(Mandatory = $false)] [string[]]$Args = @(),
        [parameter(Mandatory = $false)] [string[]]$Env = @(),
        [parameter(Mandatory = $false)] [int]$PeriodSeconds = 600,
        [parameter(Mandatory = $false)] [switch]$SaveStderrOutput
    )
    if (-not $Plan.ContainsKey("periodicInstructions")) {
        $Plan.periodicInstructions = @()
    }
    $instruction = @{
        name          = $Name
        command       = $Command
        args          = $Args
        env           = $Env
        periodSeconds = $PeriodSeconds
    }
    if ($SaveStderrOutput) {
        $instruction.saveStderrOutput = $true
    }
    $Plan.periodicInstructions += , $instruction
}

# Add-PlanProbe adds an HTTP-based probe to the plan, mirroring planapi.Probe. Probes is a JSON
# object keyed by probe name (planapi.Plan.Probes is map[string]Probe), not an array, so this
# assigns into a hashtable rather than appending to a list.
function Add-PlanProbe {
    param(
        [parameter(Mandatory = $true)] [hashtable]$Plan,
        [parameter(Mandatory = $true)] [string]$Name,
        [parameter(Mandatory = $true)] [string]$Url,
        [parameter(Mandatory = $false)] [int]$InitialDelaySeconds = 0,
        [parameter(Mandatory = $false)] [int]$TimeoutSeconds = 2,
        [parameter(Mandatory = $false)] [int]$SuccessThreshold = 1,
        [parameter(Mandatory = $false)] [int]$FailureThreshold = 1
    )
    if (-not $Plan.ContainsKey("probes")) {
        $Plan.probes = @{}
    }
    $Plan.probes[$Name] = @{
        name                = $Name
        initialDelaySeconds = $InitialDelaySeconds
        timeoutSeconds      = $TimeoutSeconds
        successThreshold    = $SuccessThreshold
        failureThreshold    = $FailureThreshold
        httpGet             = @{
            url      = $Url
            insecure = $true
        }
    }
}

# Write-PlanFile serializes a plan hashtable to a JSON file that planctl apply-plan can read.
function Write-PlanFile {
    param(
        [parameter(Mandatory = $true)] [hashtable]$Plan,
        [parameter(Mandatory = $true)] [string]$Path
    )
    # -Encoding utf8 on Windows PowerShell 5.1 (unlike PowerShell 7+) writes a UTF-8 byte order
    # mark, which encoding/json on the agent side does not skip when parsing the plan. Write
    # without a BOM directly via .NET so this works the same on both PowerShell versions.
    $json = $Plan | ConvertTo-Json -Depth 10
    $utf8NoBom = New-Object System.Text.UTF8Encoding($false)
    [System.IO.File]::WriteAllText($Path, $json, $utf8NoBom)
}

$script:PlanScratchDirectory = "C:\wins-plan-e2e"

# Get-PlanScratchDirectory returns the directory to which every plan spec confines its side
# effects.
function Get-PlanScratchDirectory {
    return $script:PlanScratchDirectory
}

# The probe target handler always returns a non-2xx status. A probe pointed at an unreachable
# port instead would produce a connection error, which prober.DoProbe returns early on without
# ever calling applyProbeResult -- FailureCount would never increment and Healthy would never be
# meaningfully set, indistinguishable from "never evaluated". A real HTTP response, even a
# failing one, is required to prove a probe keeps being evaluated over time.
$script:ProbeTargetHandler = {
    param(
        [int]$Port,
        [int]$StatusCode
    )

    $http = New-Object System.Net.HttpListener
    $http.Prefixes.Add("http://localhost:$Port/")
    $http.Start()

    while ($http.IsListening) {
        $ctx = $http.GetContext()
        if ($ctx.Request.RawUrl -eq "/kill") {
            # A dedicated kill endpoint avoids a deadlock encountered when Stop-Job is invoked at
            # the same time that this function is blocked inside GetContext().
            $ctx.Response.OutputStream.Close()
            exit 0
        }
        $ctx.Response.StatusCode = $StatusCode
        $ctx.Response.OutputStream.Close()
    }
}

# Start-ProbeTargetServer starts a background HTTP listener that always returns StatusCode, for
# use as a plan probe's httpGet target. The caller must stop it with Stop-ProbeTargetServer,
# typically in a try/finally around the spec body.
function Start-ProbeTargetServer {
    param(
        [parameter(Mandatory = $true)] [int]$Port,
        [parameter(Mandatory = $false)] [int]$StatusCode = 500
    )
    return Start-Job -ScriptBlock $script:ProbeTargetHandler -ArgumentList $Port, $StatusCode
}

# Stop-ProbeTargetServer stops a server started by Start-ProbeTargetServer.
function Stop-ProbeTargetServer {
    param(
        [parameter(Mandatory = $true)] [int]$Port,
        [parameter(Mandatory = $true)] $Job
    )
    try {
        Invoke-WebRequest -UseBasicParsing -Uri "http://localhost:$Port/kill" -TimeoutSec 5 | Out-Null
    }
    catch {
        # The listener may already have exited; ignore.
    }
    Stop-Job -Job $Job -ErrorAction SilentlyContinue | Out-Null
    Remove-Job -Job $Job -ErrorAction SilentlyContinue | Out-Null
}

# Reset-PlanScratchDirectory recreates the scratch directory used by plan instructions, called
# from each spec's BeforeEach.
function Reset-PlanScratchDirectory {
    Remove-Item -Path $script:PlanScratchDirectory -Recurse -Force -ErrorAction Ignore
    New-Item -Path $script:PlanScratchDirectory -ItemType Directory -Force | Out-Null
}

# Clear-PlanScratchDirectory removes the scratch directory without recreating it, called from
# each spec's AfterEach. The next spec's BeforeEach recreates it via Reset-PlanScratchDirectory.
function Clear-PlanScratchDirectory {
    Remove-Item -Path $script:PlanScratchDirectory -Recurse -Force -ErrorAction Ignore
}

# Clear-PlanScratchProcesses stops any process whose command line references the scratch
# directory, called from each spec's AfterEach before clearing the directory.
function Clear-PlanScratchProcesses {
    $scratch = $script:PlanScratchDirectory
    Get-CimInstance Win32_Process -ErrorAction SilentlyContinue |
        Where-Object { $_.CommandLine -and $_.CommandLine.Contains($scratch) } |
        ForEach-Object {
            try {
                Stop-Process -Id $_.ProcessId -Force -ErrorAction Stop
            }
            catch {
                Log-Warn "Failed to stop process $($_.ProcessId): $($_.Exception.Message)"
            }
        }
}

# Assert-WinsServiceRunning fails loudly if rancher-wins is not Running. A failed plan Secret
# update makes the agent call logrus.Fatalf and exit; without this check every subsequent spec
# would time out identically rather than naming the real cause.
function Assert-WinsServiceRunning {
    $service = Get-Service -Name "rancher-wins" -ErrorAction SilentlyContinue
    if (-not $service -or $service.Status -ne "Running") {
        throw "rancher-wins is not Running (status: $($service.Status)); the agent may have crashed on a prior Secret update"
    }
}

# Restart-WinsService stops and restarts rancher-wins, for specs that verify state (e.g. a pause
# checkpoint) survives an agent process restart, not just a live agent's own reconcile loop.
function Restart-WinsService {
    $serviceName = "rancher-wins"
    $timeout = 60

    Stop-Service -Name $serviceName
    $elapsed = 0
    while ((Get-Service $serviceName).Status -ne 'Stopped') {
        if ($elapsed -ge $timeout) {
            throw "$serviceName did not stop within $timeout seconds"
        }
        Start-Sleep -s 2
        $elapsed += 2
    }

    Start-Service -Name $serviceName
    $elapsed = 0
    while ((Get-Service $serviceName).Status -ne 'Running') {
        if ($elapsed -ge $timeout) {
            throw "$serviceName did not start within $timeout seconds"
        }
        Start-Sleep -s 2
        $elapsed += 2
    }
}

# Show-PlanFailureDiagnostics dumps the decoded plan outcome and the rancher-wins Event Log
# entries since $Since. Called unconditionally from each spec's AfterEach: Pester does not
# expose a reliable, version-independent way to check the current test's pass/fail state from
# inside AfterEach, so this always dumps rather than depending on an undocumented internal.
function Show-PlanFailureDiagnostics {
    param(
        [parameter(Mandatory = $true)] [datetime]$Since
    )
    Log-Info "Plan outcome after this test"
    try {
        Get-PlanOutcome | ConvertTo-Json -Depth 10 | Write-Host
    }
    catch {
        Log-Error "Failed to fetch plan outcome: $($_.Exception.Message)"
    }

    Log-Info "rancher-wins Event Log entries since $Since"
    try {
        Get-WinEvent -FilterHashtable @{ ProviderName = "rancher-wins"; StartTime = $Since } -ErrorAction Stop |
            Format-Table -AutoSize -Wrap |
            Out-String |
            Write-Host
    }
    catch {
        Log-Error "Failed to fetch rancher-wins Event Log entries: $($_.Exception.Message)"
    }
}

Export-ModuleMember -Function Set-PlanKubeconfig
Export-ModuleMember -Function Invoke-PlanCtl
Export-ModuleMember -Function Invoke-PlanBootstrap
Export-ModuleMember -Function Invoke-PlanApply
Export-ModuleMember -Function Invoke-PlanAnnotate
Export-ModuleMember -Function Get-PlanOutcome
Export-ModuleMember -Function Reset-Plan
Export-ModuleMember -Function New-PlanSpec
Export-ModuleMember -Function Add-PlanFile
Export-ModuleMember -Function Add-PlanInstruction
Export-ModuleMember -Function Add-PlanPeriodicInstruction
Export-ModuleMember -Function Add-PlanProbe
Export-ModuleMember -Function Write-PlanFile
Export-ModuleMember -Function Assert-WinsServiceRunning
Export-ModuleMember -Function Restart-WinsService
Export-ModuleMember -Function Reset-PlanScratchDirectory
Export-ModuleMember -Function Clear-PlanScratchDirectory
Export-ModuleMember -Function Get-PlanScratchDirectory
Export-ModuleMember -Function Show-PlanFailureDiagnostics
Export-ModuleMember -Function Clear-PlanScratchProcesses

# Re-exported so callers that only import planutils.psm1 (bootstrap.ps1, the spec files) get
# these too. Import-Module creates a separate module scope: the nested Import-Module of
# utils.psm1 above makes Log-Info, Judge, Wait-Ready, etc. visible inside this module's own
# functions, but does not automatically propagate them to planutils.psm1's own importers.
Export-ModuleMember -Function Log-Info
Export-ModuleMember -Function Log-Warn
Export-ModuleMember -Function Log-Error
Export-ModuleMember -Function Log-Fatal
Export-ModuleMember -Function Judge
Export-ModuleMember -Function Wait-Ready
Export-ModuleMember -Function Start-ProbeTargetServer
Export-ModuleMember -Function Stop-ProbeTargetServer
