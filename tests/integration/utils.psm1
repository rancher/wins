function Log-Info {
    $ts = (Get-Date).ToString("hh:mm:ss.fff")
    Write-Host -NoNewline -ForegroundColor Blue "[INFO $ts] "
    Write-Host -ForegroundColor Gray ("{0,-44}" -f ($args -join " "))
}

function Log-Warn {
    $ts = (Get-Date).ToString("hh:mm:ss.fff")
    Write-Host -NoNewline -ForegroundColor DarkYellow "[WARN $ts] "
    Write-Host -ForegroundColor Gray ("{0,-44}" -f ($args -join " "))
}

function Log-Error {
    $ts = (Get-Date).ToString("hh:mm:ss.fff")
    Write-Host -NoNewline -ForegroundColor DarkRed "[ERRO $ts] "
    Write-Host -ForegroundColor Gray ("{0,-44}" -f ($args -join " "))
}

function Log-Fatal {
    $ts = (Get-Date).ToString("hh:mm:ss.fff")
    Write-Host -NoNewline -ForegroundColor DarkRed "[FATA $ts] "
    Write-Host -ForegroundColor Gray ("{0,-44}" -f ($args -join " "))

    exit 255
}

function Execute-Binary {
    param (
        [parameter(Mandatory = $true)] [string]$FilePath,
        [parameter(Mandatory = $false)] [string[]]$ArgumentList,
        [parameter(Mandatory = $false)] [switch]$PassThru,
        [parameter(Mandatory = $false)] [switch]$Backgroud
    )

    if ($Backgroud) {
        if ($ArgumentList) {
            return Start-Process -WindowStyle Hidden -FilePath $FilePath -ArgumentList $ArgumentList -PassThru
        }
        else {
            return Start-Process -WindowStyle Hidden -FilePath $FilePath -PassThru
        }
    }

    if (-not $PassThru) {
        if ($ArgumentList) {
            Start-Process -NoNewWindow -Wait -FilePath $FilePath -ArgumentList $ArgumentList
        }
        else {
            Start-Process -NoNewWindow -Wait -FilePath $FilePath
        }
        return
    }

    $stdout = New-TemporaryFile
    $stderr = New-TemporaryFile
    $stdoutContent = ""
    $stderrContent = ""
    try {
        if ($ArgumentList) {
            Start-Process -NoNewWindow -Wait -FilePath $FilePath -ArgumentList $ArgumentList -RedirectStandardOutput $stdout.FullName -RedirectStandardError $stderr.FullName -ErrorAction Ignore
        }
        else {
            Start-Process -NoNewWindow -Wait -FilePath $FilePath -RedirectStandardOutput $stdout.FullName -RedirectStandardError $stderr.FullName -ErrorAction Ignore
        }

        $stdoutContent = Get-Content -Raw $stdout.FullName
        $stderrContent = Get-Content -Raw $stderr.FullName
    }
    catch {
        $stderrContent = $_.Exception.Message
    }

    $stdout.Delete()
    $stderr.Delete()

    if ([string]::IsNullOrEmpty($stderrContent)) {
        if (-not ([string]::IsNullOrEmpty($stdoutContent))) {
            if (($stdoutContent -match 'FATA') -or ($stdoutContent -match 'ERRO')) {
                return @{
                    Ok     = $false
                    Output = $stdoutContent
                }
            }
        }

        return @{
            Ok     = $true
            Output = $stdoutContent
        }
    }

    return @{
        Ok     = $false
        Output = $stderrContent
    }
}

function Judge {
    param(
        [parameter(Mandatory = $true, ValueFromPipeline = $true)] [scriptBlock]$Block,
        [parameter(Mandatory = $false)] [int]$Timeout = 30,
        [parameter(Mandatory = $false)] [switch]$Reverse,
        [parameter(Mandatory = $false)] [switch]$Throw
    )

    $count = $Timeout
    while ($count -gt 0) {
        Start-Sleep -s 1

        if (&$Block) {
            if (-not $Reverse) {
                Start-Sleep -s 5
                break
            }
        }
        elseif ($Reverse) {
            Start-Sleep -s 5
            break
        }

        Start-Sleep -s 1
        $count -= 1
    }

    if ($count -le 0) {
        if ($Throw) {
            throw "Timeout"
        }

        Log-Fatal "Timeout"
    }

}

function Wait-Ready {
    param(
        [parameter(Mandatory = $true)] $Path,
        [parameter(Mandatory = $false)] [int]$Timeout = 30,
        [parameter(Mandatory = $false)] [switch]$Throw
    )

    {
        Test-Path -Path $Path -ErrorAction Ignore
    } | Judge -Throw:$Throw -Timeout $Timeout
}

function Set-WinsConfig {
    # Note: The csi proxy config is intentionally omitted
    $winsConfig =
    @"
white_list:
  processPaths:
    - C:/etc/rancher/wins/powershell.exe
    - C:/etc/wmi-exporter/wmi-exporter.exe
    - C:/etc/windows-exporter/windows-exporter.exe
  proxyPorts:
    - 9796
agentStrictTLSMode: false
debug: false
systemagent:
  workDirectory: C:/etc/rancher/wins/work
  appliedPlanDirectory: C:/etc/rancher/wins/applied
  remoteEnabled: false
  localEnabled: true
  preserveWorkDirectory: false
"@
    Add-Content -Path C:/etc/rancher/wins/config -Value $winsConfig
}

function New-Directory {
    [CmdletBinding()]
    param (
        [Parameter()]
        [string]
        $Path
    )
    if (-not (Test-Path -Path $Path)) {
        New-Item -Path $Path -ItemType Directory | Out-Null
    }
}

function Add-DummyRKE2Service {
    Log-Info "Creating dummy rke2 service"
    $dummyServiceScript = @"
  using System;
  class Hello {
    static void Main() {
        // Intentionally empty
    }
  }
"@
    # Compile the above c# into a binary that can be used as a service.
    # Note: This service will not start, as there is no service handler implemented, however it can still
    # be configured in the same ways that any other functional service can be.
    Add-Type -TypeDefinition $dummyServiceScript -Language CSharp -OutputAssembly "rke2.exe" -OutputType ConsoleApplication
    New-Service -Name "rke2" -BinaryPathName "rke2.exe"
}

function Remove-DummyRKE2Service {
    Log-Info "Removing dummy rke2 service"
    sc.exe delete rke2
}

function Remove-RancherWinsService {
    Log-Info "Removing rancher-wins service"
    $env:CATTLE_AGENT_CONFIG_DIR = "C:/etc/rancher/wins"
    # stop and remove the services
    Stop-Service rancher-wins
    $ret = .\bin\wins.exe srv app run --unregister
    if ($LASTEXITCODE -ne 0) {
        Log-Error $ret
        $false | Should -Be $true
    }

    # Force kill any running processes
    Stop-Process -Name "rancher-wins" -ErrorAction SilentlyContinue
    Log-Info "Removing rancher-wins config file"
    # Remove any existing config files
    Remove-Item -Force -Recurse $env:CATTLE_AGENT_CONFIG_DIR -ErrorAction Ignore
}

function Add-RancherWinsService {
    Log-Info "Resetting the rancher-wins config directory"
    # (Re)create the config file directory
    $env:CATTLE_AGENT_CONFIG_DIR = "C:/etc/rancher/wins"
    # The CATTLE_AGENT_CONFIG_DIR may have been created in other tests
    # we should clear it out to ensure we have a clean slate for this test
    Remove-Item -Force -Recurse $env:CATTLE_AGENT_CONFIG_DIR -ErrorAction Ignore
    New-Directory -Path $env:CATTLE_AGENT_CONFIG_DIR

    # create the config file
    Set-WinsConfig

    # register the service
    Log-Info "Adding rancher-wins service"
    $ret = .\bin\wins.exe srv app run --register
    if ($LASTEXITCODE -ne 0) {
        Log-Error $ret
        $false | Should -Be $true
    }

    # verify
    Get-Service -Name rancher-wins -ErrorAction SilentlyContinue | Should -Not -BeNullOrEmpty
    Log-Info (Get-Service -Name rancher-wins -ErrorAction Ignore)
}

function Add-RKE2WinsDependency {
    sc.exe config rke2 depend= rancher-wins
}

function Get-Permissions {
    param (
        [Parameter(Mandatory=$true)]
        [string]
        $Path
    )

    $exists = Test-Path $Path
    if (-not $exists) {
        throw "Cannot get permissions on path $Path if a file or directory does not exist"
    }

    $acl = Get-Acl $Path

    $owner = $acl.Owner
    $group = $acl.Group
    $permissions = @()
    foreach ($rule in $acl.Access) {
        $permissions += [PSCustomObject]@{
            AccessMask = $rule.FileSystemRights.ToString()
            Type = $rule.AccessControlType
            Identity = $rule.IdentityReference.Value
        }
    }

    return $owner, $group, $permissions
}

function Test-Permissions {
    param (
        [Parameter(Mandatory=$true)]
        [string]
        $Path,

        [Parameter(Mandatory=$true)]
        [string]
        $ExpectedOwner,

        [Parameter(Mandatory=$true)]
        [string]
        $ExpectedGroup,

        [Parameter(Mandatory=$true)]
        [System.Object[]]
        $ExpectedPermissions
    )

    $owner, $group, $permissions = Get-Permissions -Path $Path

    $errors = @()

    if ($owner -ne $ExpectedOwner) {
        $errors += "expected owner $ExpectedOwner, found $owner"
    }

    if ($group -ne $ExpectedGroup) {
        $errors += "expected group $ExpectedGroup, found $group"
    }

    $expected = $ExpectedPermissions | ConvertTo-Json
    $found = $permissions | ConvertTo-Json

    if ($expected -ne $found) {
        $errors += "expected permissions $expected, found $found"
    }

    # Check
    if ($errors.Count -gt 0) {
        $errors_joined = $errors -join "`n- "
        throw "Permissions don't match expectations:`n- $errors_joined"
    }
}

function Ensure-DependencyExistsForService {
    param (
        [Parameter(Mandatory=$true)]
        [string]
        $ServiceName,
        [Parameter(Mandatory=$true)]
        [string]
        $DependencyName
    )

    $dependencies = (Get-Service -Name $ServiceName).ServicesDependedOn
    Log-Info "Checking for $ServiceName service dependency on $DependencyName..."
    $found = $false
    foreach ($dep in $dependencies) {
        Log-Info "found dependency $($DependencyName)"
        if ($dep.Name -eq "rancher-wins") {
            return $true
        }
    }

    return $false
}

Export-ModuleMember -Function Log-Info
Export-ModuleMember -Function Log-Warn
Export-ModuleMember -Function Log-Error
Export-ModuleMember -Function Log-Fatal
Export-ModuleMember -Function Execute-Binary
Export-ModuleMember -Function Judge
Export-ModuleMember -Function Wait-Ready
Export-ModuleMember -Function New-Directory
Export-ModuleMember -Function Set-WinsConfig
Export-ModuleMember -Function Add-DummyRKE2Service
Export-ModuleMember -Function Remove-DummyRKE2Service
Export-ModuleMember -Function Add-RancherWinsService
Export-ModuleMember -Function Remove-RancherWinsService
Export-ModuleMember -Function Get-Permissions
Export-ModuleMember -Function Test-Permissions
Export-ModuleMember -Function Ensure-DependencyExistsForService
Export-ModuleMember -Function Add-RKE2WinsDependency
