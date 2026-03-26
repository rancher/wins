$ErrorActionPreference = "Stop"

Import-Module -Name @(
    "$PSScriptRoot\utils.psm1"
) -WarningAction Ignore

function New-DummyService {
    param(
        [Parameter(Mandatory)][string]$Name,
        [string]$DisplayName = $Name
    )

    Remove-DummyService -Name $Name

    $testExe   = "tests\integration\bin\test_service.exe"
    $svcExe    = "$PSScriptRoot\$Name-svc.exe"
    Copy-Item $testExe $svcExe -Force

    $result = sc.exe create $Name binPath= "$svcExe $Name" start= demand DisplayName= $DisplayName
    if ($LASTEXITCODE -ne 0) {
        throw "sc.exe create failed for '$Name': $result"
    }

    Start-Service -Name $Name

    $timeout = 30
    $elapsed = 0
    while ((Get-Service -Name $Name).Status -ne 'Running' -and $elapsed -lt $timeout) {
        Start-Sleep -Seconds 1
        $elapsed++
    }
    if ((Get-Service -Name $Name).Status -ne 'Running') {
        throw "Timed out waiting for dummy service '$Name' to reach Running state"
    }
    Log-Info "Dummy service '$Name' is Running"
}

# ---------------------------------------------------------------------------
# Helper: Cleanly stop, delete, and remove the binary of a dummy service.
# ---------------------------------------------------------------------------
function Remove-DummyService {
    param([Parameter(Mandatory)][string]$Name)

    $svc = Get-Service -Name $Name -ErrorAction SilentlyContinue
    if ($svc) {
        if ($svc.Status -ne 'Stopped') {
            Stop-Service -Name $Name -Force -ErrorAction SilentlyContinue
            $timeout = 30
            $elapsed = 0
            while ((Get-Service -Name $Name -ErrorAction SilentlyContinue).Status -ne 'Stopped' -and $elapsed -lt $timeout) {
                Start-Sleep -Seconds 1
                $elapsed++
            }
        }
        sc.exe delete $Name | Out-Null
        Log-Info "Removed dummy service '$Name'"
    }

    # Remove the copied service binary regardless of whether the service existed
    $svcExe = "$PSScriptRoot\$Name-svc.exe"
    Remove-Item $svcExe -Force -ErrorAction SilentlyContinue
}


# ---------------------------------------------------------------------------
# Helper: Copy test_service.exe to <n>.exe in the
# same directory, then start it so Get-Process -Name <n> finds a real
# running process. test_service.exe is a simple long-running executable that stays
# alive for 15 minutes until killed — no arguments needed.
# ---------------------------------------------------------------------------
function New-DummyProcess {
    param([Parameter(Mandatory)][string]$Name)

    Remove-DummyProcess -Name $Name

    $testExe  = "tests\integration\bin\test_service.exe"
    $dummyExe = "$PSScriptRoot\$Name.exe"
    Copy-Item $testExe $dummyExe -Force

    Start-Process -FilePath $dummyExe -WindowStyle Hidden

    $timeout = 15
    $elapsed = 0
    while (-not (Get-Process -Name $Name -ErrorAction SilentlyContinue) -and $elapsed -lt $timeout) {
        Start-Sleep -Seconds 1
        $elapsed++
    }
    if (-not (Get-Process -Name $Name -ErrorAction SilentlyContinue)) {
        throw "Dummy process '$Name' did not appear within $timeout seconds"
    }
    Log-Info "Dummy process '$Name' is running"
}

# ---------------------------------------------------------------------------
# Helper: Kill all instances of a dummy process and remove the copied exe.
# Waits for the process to fully exit before deleting the file, since
# Windows holds a file lock on the exe while the process is alive.
# ---------------------------------------------------------------------------
function Remove-DummyProcess {
    param([Parameter(Mandatory)][string]$Name)

    Get-Process -Name $Name -ErrorAction SilentlyContinue | Stop-Process -Force -ErrorAction SilentlyContinue

    $timeout = 15
    $elapsed = 0
    while ((Get-Process -Name $Name -ErrorAction SilentlyContinue) -and $elapsed -lt $timeout) {
        Start-Sleep -Seconds 1
        $elapsed++
    }

    $dummyExe = "$PSScriptRoot\$Name.exe"
    Remove-Item $dummyExe -Force -ErrorAction SilentlyContinue
}

# ---------------------------------------------------------------------------
# Helper: Dot-source the uninstaller in the current session.
# Returns $true if the script completed without error, $false if it threw
# (e.g. Write-LogFatal calling exit 255 surfaces as a terminating error
# when dot-sourced). Dot-sourcing keeps all $env: mutations visible to the
# calling test without spawning a child process.
# ---------------------------------------------------------------------------
function Invoke-Uninstaller {
    try {
        . .\uninstaller-test.ps1
        return $true
    }
    catch {
        return $false
    }
}

# ---------------------------------------------------------------------------
# Pre-suite: remove any leftovers from previous runs
# ---------------------------------------------------------------------------
try {
    Remove-DummyService -Name "rancher-wins"
    Remove-DummyService -Name "csiproxy"
    Remove-DummyService -Name "rke2"
    Remove-DummyProcess -Name "rke2"
    Remove-DummyProcess -Name "wins"
    Remove-DummyProcess -Name "csi-proxy"
    Get-NetFirewallRule -PolicyStore ActiveStore -Name "rancher-wins-*" -ErrorAction Ignore | ForEach-Object { Remove-NetFirewallRule -Name $_.Name -PolicyStore ActiveStore -ErrorAction Ignore } | Out-Null
}
catch {
    Log-Warn $_.Exception.Message
}

Describe "uninstall" {

    BeforeEach {
        Log-Info "Setting up environment for uninstall test"

        $env:CATTLE_AGENT_CONFIG_DIR = "C:/etc/rancher/wins"
        $env:CATTLE_AGENT_VAR_DIR    = "C:/var/lib/rancher/agent"
        $env:CATTLE_AGENT_BIN_PREFIX = "C:/usr/local"

        New-Item -Path $env:CATTLE_AGENT_CONFIG_DIR -ItemType Directory -Force | Out-Null
        New-Item -Path $env:CATTLE_AGENT_VAR_DIR    -ItemType Directory -Force | Out-Null
        New-Item -Path $env:CATTLE_AGENT_BIN_PREFIX -ItemType Directory -Force | Out-Null
        New-Item -Path "C:/windows/wins.exe"         -ItemType File      -Force | Out-Null

        Copy-Item uninstall.ps1 uninstaller-test.ps1 -Force
    }

    AfterEach {
        Log-Info "Cleaning up after uninstall test"

        # Always clean up ALL dummy services and processes so a failed test
        # cannot leave behind a stale rke2/wins/csi-proxy that poisons the next test.
        Remove-DummyService -Name "rancher-wins"
        Remove-DummyService -Name "csiproxy"
        Remove-DummyService -Name "rke2"
        Remove-DummyProcess -Name "rke2"
        Remove-DummyProcess -Name "wins"
        Remove-DummyProcess -Name "csi-proxy"

        # Use literal paths here — env vars may be null if a test cleared them
        # before the script ran and the script aborted before restoring them.
        Remove-Item -Path "C:/etc/rancher/wins"        -Recurse -Force -ErrorAction Ignore
        Remove-Item -Path "C:/var/lib/rancher/agent"   -Recurse -Force -ErrorAction Ignore
        Remove-Item -Path "C:/windows/wins.exe"        -Force          -ErrorAction Ignore
        Remove-Item -Path "C:/etc/windows-exporter"    -Recurse -Force -ErrorAction Ignore
        Remove-Item -Path "C:/etc/wmi-exporter"        -Recurse -Force -ErrorAction Ignore
        Remove-Item -Path "uninstaller-test.ps1"       -Force          -ErrorAction Ignore

        Remove-Item Env:\CATTLE_AGENT_LOGLEVEL -ErrorAction Ignore
    }

    # -------------------------------------------------------------------------
    # Default environment variable handling
    # Set-Environment runs before the RKE2 safeguard, so even when the script
    # later aborts (exit 255), the env vars are already written to the session.
    # -------------------------------------------------------------------------

    It "uses default CATTLE_AGENT_CONFIG_DIR when env var is not set" {
        Log-Info "TEST: [uses default CATTLE_AGENT_CONFIG_DIR when env var is not set]"

        Remove-Item Env:\CATTLE_AGENT_CONFIG_DIR -ErrorAction Ignore

        Invoke-Uninstaller

        $env:CATTLE_AGENT_CONFIG_DIR | Should -Be "C:/etc/rancher/wins"
    }

    It "uses default CATTLE_AGENT_VAR_DIR when env var is not set" {
        Log-Info "TEST: [uses default CATTLE_AGENT_VAR_DIR when env var is not set]"

        Remove-Item Env:\CATTLE_AGENT_VAR_DIR -ErrorAction Ignore

        Invoke-Uninstaller

        $env:CATTLE_AGENT_VAR_DIR | Should -Be "C:/var/lib/rancher/agent"
    }

    It "uses default CATTLE_AGENT_BIN_PREFIX when env var is not set" {
        Log-Info "TEST: [uses default CATTLE_AGENT_BIN_PREFIX when env var is not set]"

        Remove-Item Env:\CATTLE_AGENT_BIN_PREFIX -ErrorAction Ignore

        Invoke-Uninstaller

        $env:CATTLE_AGENT_BIN_PREFIX | Should -Be "c:/usr/local"
    }

    It "sets CATTLE_AGENT_LOGLEVEL to debug when env var is not set" {
        Log-Info "TEST: [sets CATTLE_AGENT_LOGLEVEL to debug when env var is not set]"

        Remove-Item Env:\CATTLE_AGENT_LOGLEVEL -ErrorAction Ignore

        Invoke-Uninstaller

        $env:CATTLE_AGENT_LOGLEVEL | Should -Be "debug"
    }

    It "lowercases CATTLE_AGENT_LOGLEVEL when env var is already set" {
        Log-Info "TEST: [lowercases CATTLE_AGENT_LOGLEVEL when env var is already set]"

        $env:CATTLE_AGENT_LOGLEVEL = "DEBUG"

        Invoke-Uninstaller

        $env:CATTLE_AGENT_LOGLEVEL | Should -Be "debug"
    }

    # -------------------------------------------------------------------------
    # RKE2 safeguard
    #
    # Write-LogFatal calls exit 255, which surfaces as a terminating error when
    # dot-sourced. Invoke-Uninstaller catches it and returns $false. We detect
    # the abort by confirming the config directory was NOT deleted (i.e. the
    # uninstall was halted early) and that rancher-wins and csiproxy are still
    # running, confirming no teardown occurred when the safeguard fired.
    # -------------------------------------------------------------------------

    It "aborts uninstall when a real rke2 service is running" {
        Log-Info "TEST: [aborts uninstall when a real rke2 service is running]"

        New-DummyService -Name "rke2"         -DisplayName "RKE2 (test dummy)"
        New-DummyService -Name "rancher-wins" -DisplayName "Rancher Wins (test dummy)"
        New-DummyService -Name "csiproxy"     -DisplayName "CSI Proxy (test dummy)"

        (Get-Service -Name "rke2").Status | Should -Be "Running"

        $succeeded = Invoke-Uninstaller

        # Script must have aborted — Invoke-Uninstaller should return $false
        $succeeded | Should -Be $false

        # Config dir must still be present — Remove-WinsConfig was never reached
        Test-Path $env:CATTLE_AGENT_CONFIG_DIR | Should -Be $true

        # Dependent services must still be intact — no teardown should have occurred
        (Get-Service -Name "rancher-wins" -ErrorAction SilentlyContinue).Status | Should -Be "Running"
        (Get-Service -Name "csiproxy"     -ErrorAction SilentlyContinue).Status | Should -Be "Running"
    }

    It "aborts uninstall when a real rke2 process is running" {
        Log-Info "TEST: [aborts uninstall when a real rke2 process is running]"

        New-DummyProcess -Name "rke2"
        New-DummyService -Name "rancher-wins" -DisplayName "Rancher Wins (test dummy)"
        New-DummyService -Name "csiproxy"     -DisplayName "CSI Proxy (test dummy)"

        (Get-Process -Name "rke2" -ErrorAction SilentlyContinue) | Should -Not -BeNullOrEmpty

        $succeeded = Invoke-Uninstaller

        # Script must have aborted — Invoke-Uninstaller should return $false
        $succeeded | Should -Be $false

        # Config dir must still be present — Remove-WinsConfig was never reached
        Test-Path $env:CATTLE_AGENT_CONFIG_DIR | Should -Be $true

        # Dependent services must still be intact — no teardown should have occurred
        (Get-Service -Name "rancher-wins" -ErrorAction SilentlyContinue).Status | Should -Be "Running"
        (Get-Service -Name "csiproxy"     -ErrorAction SilentlyContinue).Status | Should -Be "Running"
    }

    It "proceeds with uninstall when rke2 is not running" {
        Log-Info "TEST: [proceeds with uninstall when rke2 is not running]"

        (Get-Service -Name "rke2" -ErrorAction SilentlyContinue) | Should -BeNullOrEmpty
        (Get-Process -Name "rke2" -ErrorAction SilentlyContinue) | Should -BeNullOrEmpty

        $succeeded = Invoke-Uninstaller

        $succeeded | Should -Be $true
        Test-Path $env:CATTLE_AGENT_CONFIG_DIR | Should -Be $false
    }

    # -------------------------------------------------------------------------
    # Config and var directory removal
    # -------------------------------------------------------------------------

    It "removes CATTLE_AGENT_CONFIG_DIR after uninstall" {
        Log-Info "TEST: [removes CATTLE_AGENT_CONFIG_DIR after uninstall]"

        Test-Path $env:CATTLE_AGENT_CONFIG_DIR | Should -Be $true

        $succeeded = Invoke-Uninstaller

        $succeeded | Should -Be $true
        Test-Path $env:CATTLE_AGENT_CONFIG_DIR | Should -Be $false
    }

    It "removes CATTLE_AGENT_VAR_DIR after uninstall" {
        Log-Info "TEST: [removes CATTLE_AGENT_VAR_DIR after uninstall]"

        Test-Path $env:CATTLE_AGENT_VAR_DIR | Should -Be $true

        $succeeded = Invoke-Uninstaller

        $succeeded | Should -Be $true
        Test-Path $env:CATTLE_AGENT_VAR_DIR | Should -Be $false
    }

    It "removes nested files inside CATTLE_AGENT_CONFIG_DIR" {
        Log-Info "TEST: [removes nested files inside CATTLE_AGENT_CONFIG_DIR]"

        $nestedFile = "$env:CATTLE_AGENT_CONFIG_DIR/config"
        New-Item -Path $nestedFile -ItemType File -Force | Out-Null
        Test-Path $nestedFile | Should -Be $true

        $succeeded = Invoke-Uninstaller

        $succeeded | Should -Be $true
        Test-Path $nestedFile | Should -Be $false
    }

    # -------------------------------------------------------------------------
    # wins.exe removal (wins-for-charts)
    # -------------------------------------------------------------------------

    It "removes wins.exe from C:/windows after uninstall" {
        Log-Info "TEST: [removes wins.exe from C:/windows after uninstall]"

        Test-Path "C:/windows/wins.exe" | Should -Be $true

        $succeeded = Invoke-Uninstaller

        $succeeded | Should -Be $true
        Test-Path "C:/windows/wins.exe" | Should -Be $false
    }

    It "does not fail when wins.exe is already absent from C:/windows" {
        Log-Info "TEST: [does not fail when wins.exe is already absent from C:/windows]"

        Remove-Item "C:/windows/wins.exe" -Force -ErrorAction Ignore

        $succeeded = Invoke-Uninstaller

        $succeeded | Should -Be $true
    }

    # -------------------------------------------------------------------------
    # Exporter directory removal
    # -------------------------------------------------------------------------

    It "deletes windows-exporter when only windows-exporter exists" {
        Log-Info "TEST: [deletes windows-exporter when only windows-exporter exists]"

        New-Item -Path "C:/etc/windows-exporter" -ItemType Directory -Force | Out-Null

        $succeeded = Invoke-Uninstaller

        $succeeded | Should -Be $true
        Test-Path "C:/etc/windows-exporter" | Should -Be $false
    }

    It "deletes wmi-exporter when only wmi-exporter exists" {
        Log-Info "TEST: [deletes wmi-exporter when only wmi-exporter exists]"

        New-Item -Path "C:/etc/wmi-exporter" -ItemType Directory -Force | Out-Null

        $succeeded = Invoke-Uninstaller

        $succeeded | Should -Be $true
        Test-Path "C:/etc/wmi-exporter" | Should -Be $false
    }

    It "does not fail when neither exporter directory exists" {
        Log-Info "TEST: [does not fail when neither exporter directory exists]"

        Remove-Item "C:/etc/windows-exporter" -Recurse -Force -ErrorAction Ignore
        Remove-Item "C:/etc/wmi-exporter"     -Recurse -Force -ErrorAction Ignore

        $succeeded = Invoke-Uninstaller

        $succeeded | Should -Be $true
    }

    # -------------------------------------------------------------------------
    # Service teardown — real dummy services
    #
    # New-DummyService registers a service with sc.exe using the full path to
    # powershell.exe as the binary, which satisfies the SCM start handshake.
    # The uninstaller calls the real Stop-Service and sc.exe delete against SCM.
    # -------------------------------------------------------------------------

    It "stops and removes rancher-wins service if it is running" {
        Log-Info "TEST: [stops and removes rancher-wins service if it is running]"

        New-DummyService -Name "rancher-wins" -DisplayName "Rancher Wins (test dummy)"

        (Get-Service -Name "rancher-wins").Status | Should -Be "Running"

        $succeeded = Invoke-Uninstaller

        $succeeded | Should -Be $true
        (Get-Service -Name "rancher-wins" -ErrorAction SilentlyContinue) | Should -BeNullOrEmpty
    }

    It "does not fail when rancher-wins service is not installed" {
        Log-Info "TEST: [does not fail when rancher-wins service is not installed]"

        (Get-Service -Name "rancher-wins" -ErrorAction SilentlyContinue) | Should -BeNullOrEmpty

        $succeeded = Invoke-Uninstaller

        $succeeded | Should -Be $true
    }

    It "stops and removes csiproxy service if it is running" {
        Log-Info "TEST: [stops and removes csiproxy service if it is running]"

        New-DummyService -Name "csiproxy" -DisplayName "CSI Proxy (test dummy)"

        (Get-Service -Name "csiproxy").Status | Should -Be "Running"

        $succeeded = Invoke-Uninstaller

        $succeeded | Should -Be $true
        (Get-Service -Name "csiproxy" -ErrorAction SilentlyContinue) | Should -BeNullOrEmpty
    }

    It "does not fail when csiproxy service is not installed" {
        Log-Info "TEST: [does not fail when csiproxy service is not installed]"

        (Get-Service -Name "csiproxy" -ErrorAction SilentlyContinue) | Should -BeNullOrEmpty

        $succeeded = Invoke-Uninstaller

        $succeeded | Should -Be $true
    }

    # -------------------------------------------------------------------------
    # Process teardown — real dummy processes
    #
    # New-DummyProcess copies test_service.exe to $PSScriptRoot\<n>.exe and
    # starts it so Get-Process -Name <n> finds a real running process.
    # AfterEach always calls Remove-DummyProcess so a failed test cannot leave
    # a stale process that breaks subsequent tests.
    # -------------------------------------------------------------------------

    It "stops wins process if it is running" {
        Log-Info "TEST: [stops wins process if it is running]"

        New-DummyProcess -Name "wins"

        (Get-Process -Name "wins" -ErrorAction SilentlyContinue) | Should -Not -BeNullOrEmpty

        $succeeded = Invoke-Uninstaller

        $succeeded | Should -Be $true
        (Get-Process -Name "wins" -ErrorAction SilentlyContinue) | Should -BeNullOrEmpty
    }

    It "does not fail when wins process is not running" {
        Log-Info "TEST: [does not fail when wins process is not running]"

        (Get-Process -Name "wins" -ErrorAction SilentlyContinue) | Should -BeNullOrEmpty

        $succeeded = Invoke-Uninstaller

        $succeeded | Should -Be $true
    }

    It "stops csi-proxy process if it is running" {
        Log-Info "TEST: [stops csi-proxy process if it is running]"

        New-DummyProcess -Name "csi-proxy"

        (Get-Process -Name "csi-proxy" -ErrorAction SilentlyContinue) | Should -Not -BeNullOrEmpty

        $succeeded = Invoke-Uninstaller

        $succeeded | Should -Be $true
        (Get-Process -Name "csi-proxy" -ErrorAction SilentlyContinue) | Should -BeNullOrEmpty
    }

    It "does not fail when csi-proxy process is not running" {
        Log-Info "TEST: [does not fail when csi-proxy process is not running]"

        (Get-Process -Name "csi-proxy" -ErrorAction SilentlyContinue) | Should -BeNullOrEmpty

        $succeeded = Invoke-Uninstaller

        $succeeded | Should -Be $true
    }
}
