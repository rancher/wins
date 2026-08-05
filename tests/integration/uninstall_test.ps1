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

function Invoke-Uninstaller {
    powershell.exe -NonInteractive -File ".\uninstaller-test.ps1" | Write-Host
    return ($LASTEXITCODE -eq 0)
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

        New-Item -Path $env:CATTLE_AGENT_CONFIG_DIR -ItemType Directory -Force | Out-Null
        New-Item -Path $env:CATTLE_AGENT_VAR_DIR    -ItemType Directory -Force | Out-Null
        New-Item -Path "C:/windows/wins.exe"         -ItemType File      -Force | Out-Null

        Copy-Item uninstall.ps1 uninstaller-test.ps1 -Force
    }

    AfterEach {
        Log-Info "Cleaning up after uninstall test"

        # clean up ALL dummy services and processes so a failed test
        # cannot leave behind a stale rke2/wins/csi-proxy that poisons the next test.
        Remove-DummyService -Name "rancher-wins"
        Remove-DummyService -Name "csiproxy"
        Remove-DummyService -Name "rke2"
        Remove-DummyProcess -Name "rke2"
        Remove-DummyProcess -Name "wins"
        Remove-DummyProcess -Name "csi-proxy"

        # removing paths related to script
        Remove-Item -Path "C:/etc/rancher/wins"        -Recurse -Force -ErrorAction Ignore
        Remove-Item -Path "C:/var/lib/rancher/agent"   -Recurse -Force -ErrorAction Ignore
        Remove-Item -Path "C:/windows/wins.exe"        -Force          -ErrorAction Ignore
        Remove-Item -Path "C:/etc/windows-exporter"    -Recurse -Force -ErrorAction Ignore
        Remove-Item -Path "C:/etc/wmi-exporter"        -Recurse -Force -ErrorAction Ignore
        Remove-Item -Path "C:/tmp/test-wins-config"    -Recurse -Force -ErrorAction Ignore
        Remove-Item -Path "C:/tmp/test-wins-var"       -Recurse -Force -ErrorAction Ignore
        Remove-Item -Path "uninstaller-test.ps1"       -Force          -ErrorAction Ignore

        Remove-Item Env:\CATTLE_AGENT_LOGLEVEL -ErrorAction Ignore
    }

    # -------------------------------------------------------------------------
    # RKE2 safeguard
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
    # Agent config and var directory removal
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

    It "removes custom CATTLE_AGENT_CONFIG_DIR and CATTLE_AGENT_VAR_DIR after uninstall" {
        Log-Info "TEST: [removes custom CATTLE_AGENT_CONFIG_DIR and CATTLE_AGENT_VAR_DIR after uninstall]"

        $originalConfigDir = "C:/etc/rancher/wins"
        $originalVarDir    = "C:/var/lib/rancher/agent"

        $customConfigDir = "C:/tmp/test-wins-config"
        $customVarDir    = "C:/tmp/test-wins-var"

        $env:CATTLE_AGENT_CONFIG_DIR = $customConfigDir
        $env:CATTLE_AGENT_VAR_DIR    = $customVarDir

        New-Item -Path $customConfigDir -ItemType Directory -Force | Out-Null
        New-Item -Path $customVarDir -ItemType Directory -Force | Out-Null

        Test-Path $customConfigDir | Should -Be $true
        Test-Path $customVarDir | Should -Be $true

        $succeeded = Invoke-Uninstaller

        $succeeded | Should -Be $true

        # Custom dirs should be removed
        Test-Path $customConfigDir | Should -Be $false
        Test-Path $customVarDir | Should -Be $false

        # Default dirs from BeforeEach should NOT be removed by this run
        Test-Path $originalConfigDir | Should -Be $true
        Test-Path $originalVarDir | Should -Be $true
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
    # Service Cleanup
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
    # Process Cleanup
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
