function Log-Info {
    Write-Host -NoNewline -ForegroundColor Blue "INFO "
    Write-Host -ForegroundColor Gray ("{0,-44}" -f ($args -join " "))
}

function Log-Warn {
    Write-Host -NoNewline -ForegroundColor DarkYellow "WARN "
    Write-Host -ForegroundColor Gray ("{0,-44}" -f ($args -join " "))
}

function Log-Error {
    Write-Host -NoNewline -ForegroundColor DarkRed "ERRO "
    Write-Host -ForegroundColor Gray ("{0,-44}" -f ($args -join " "))
}

function Log-Fatal {
    Write-Host -NoNewline -ForegroundColor DarkRed "FATA "
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
    $winsConfig =
    @"
white_list:
  processPaths:
    - C:/etc/rancher/wins/powershell.exe
    - C:/etc/rancher/wins/wins-upgrade.exe
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
csi-proxy:
  url: https://acs-mirror.azureedge.net/csi-proxy/%[1]s/binaries/csi-proxy-%[1]s.tar.gz
  version: v1.1.3
  kubeletPath: fake
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


Export-ModuleMember -Function Log-Info
Export-ModuleMember -Function Log-Warn
Export-ModuleMember -Function Log-Error
Export-ModuleMember -Function Log-Fatal
Export-ModuleMember -Function Execute-Binary
Export-ModuleMember -Function Judge
Export-ModuleMember -Function Wait-Ready
Export-ModuleMember -Function New-Directory
Export-ModuleMember -Function Set-WinsConfig