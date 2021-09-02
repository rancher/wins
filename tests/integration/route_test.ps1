$ErrorActionPreference = "Stop"

Import-Module -Name @(
    "$PSScriptRoot\utils.psm1"
) -WarningAction Ignore

# clean interferences
try {
    Get-NetRoute -PolicyStore ActiveStore | Where-Object { ($_.DestinationPrefix -eq "7.7.7.7/32") } | ForEach-Object { Remove-NetRoute -Confirm:$false -InterfaceIndex $_.ifIndex -DestinationPrefix $_.DestinationPrefix -NextHop $_.NextHop -PolicyStore ActiveStore -ErrorAction Stop | Out-Null }
    Get-Process -Name "wins" -ErrorAction Ignore | Stop-Process -Force -ErrorAction Ignore
}
catch {
    Log-Warn $_.Exception.Message
}

Describe "route" {

    BeforeEach {
        # start wins server
        Execute-Binary -FilePath "bin\wins.exe" -ArgumentList @('srv', 'app', 'run') -Backgroud | Out-Null
        Wait-Ready -Path //./pipe/rancher_wins
    }

    AfterEach {
        # clean wins server
        Get-Process -Name "wins" -ErrorAction Ignore | Stop-Process -Force -ErrorAction Ignore

        # clean route
        Get-NetRoute -PolicyStore ActiveStore | Where-Object { ($_.DestinationPrefix -eq "7.7.7.7/32") } | ForEach-Object { Remove-NetRoute -Confirm:$false -InterfaceIndex $_.ifIndex -DestinationPrefix $_.DestinationPrefix -NextHop $_.NextHop -PolicyStore ActiveStore -ErrorAction Stop | Out-Null }
    }

    It "add" {
        # wins.exe cli route add --addresses "7.7.7.7"
        # docker run --rm -v //./pipe/rancher_wins://./pipe/rancher_wins -v c:/etc/rancher/wins:c:/etc/rancher/wins wins-cli route add
        $ret = Execute-Binary -FilePath "docker.exe" -ArgumentList @("run", "--rm", "-v", "//./pipe/rancher_wins://./pipe/rancher_wins", "-v", "c:/etc/rancher/wins:c:/etc/rancher/wins", "wins-cli", "route", "add", "--addresses", "7.7.7.7") -PassThru
        if (-not $ret.Ok) {
            Log-Error $ret.Output
            $false | Should Be $true
        }

        # verify
        Start-Sleep -Seconds 5
        Get-NetRoute -PolicyStore ActiveStore | Where-Object { ($_.DestinationPrefix -eq "7.7.7.7/32") } | Measure-Object | Select-Object -ExpandProperty Count | Should Be 1
    }

}
