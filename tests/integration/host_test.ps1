$ErrorActionPreference = "Stop"

Import-Module -Name @(
    "$PSScriptRoot\utils.psm1"
) -WarningAction Ignore

# clean interferences
try {
    Get-Process -Name "wins" -ErrorAction Ignore | Stop-Process -Force -ErrorAction Ignore
}
catch {
    Log-Warn $_.Exception.Message
}

Describe "host" {

    BeforeEach {
        # start wins server
        Execute-Binary -FilePath "bin\wins.exe" -ArgumentList @('srv', 'app', 'run') -Backgroud | Out-Null
        Wait-Ready -Path //./pipe/rancher_wins
    }

    AfterEach {
        # clean wins server
        Get-Process -Name "wins" -ErrorAction Ignore | Stop-Process -Force -ErrorAction Ignore
    }

    It "get version" {
        # wins.exe cli host get-version
        # docker run --rm -v //./pipe/rancher_wins://./pipe/rancher_wins -v c:/etc/rancher/wins:c:/etc/rancher/wins wins-cli host get-version
        $ret = Execute-Binary -FilePath "docker.exe" -ArgumentList @("run", "--rm", "-v", "//./pipe/rancher_wins://./pipe/rancher_wins", "-v", "c:/etc/rancher/wins:c:/etc/rancher/wins", "wins-cli", "host", "get-version") -PassThru
        if (-not $ret.Ok) {
            Log-Error $ret.Output
            $false | Should Be $true
        }

        # verify
        $expectedObj = $ret.Output | ConvertFrom-Json
        $actualObj = Get-ItemProperty 'HKLM:\SOFTWARE\Microsoft\Windows NT\CurrentVersion\' | Select-Object -Property CurrentMajorVersionNumber, CurrentMinorVersionNumber, CurrentBuildNumber, UBR, ReleaseId, BuildLabEx, CurrentBuild
        $actualObj.CurrentMajorVersionNumber -eq $expectedObj.CurrentMajorVersionNumber | Should Be $true
        $actualObj.CurrentMinorVersionNumber -eq $expectedObj.CurrentMinorVersionNumber | Should Be $true
        $actualObj.CurrentBuildNumber -eq $expectedObj.CurrentBuildNumber | Should Be $true
        $actualObj.UBR -eq $expectedObj.UBR | Should Be $true
        $actualObj.ReleaseId -eq $expectedObj.ReleaseId | Should Be $true
        $actualObj.BuildLabEx -eq $expectedObj.BuildLabEx | Should Be $true
        $actualObj.CurrentBuild -eq $expectedObj.CurrentBuild | Should Be $true
    }

}
