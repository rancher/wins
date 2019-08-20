$ErrorActionPreference = "Stop"

Import-Module -Name @(
    "$PSScriptRoot\hns.psm1"
    "$PSScriptRoot\utils.psm1"
) -WarningAction Ignore

# clean interferences
try {
    Get-Process -Name "wins" -ErrorAction Ignore | Stop-Process -Force -ErrorAction Ignore
} catch {
    Log-Warn $_.Exception.Message
}

Describe "hns" {

    BeforeEach {
        # start wins server
        Execute-Binary -FilePath "bin\wins.exe" -ArgumentList @('srv', 'app', 'run') -Backgroud | Out-Null
        Wait-Ready -Path //./pipe/rancher_wins

        # create a HNS network
        try {
            New-HNSNetwork -Type "L2Bridge" -AddressPrefix "192.168.255.0/30" -Gateway "192.168.255.1" -Name "test-cbr0" | Out-Null
        } catch {}
        while ($true) {
            $n = Get-HnsNetwork -ErrorAction Ignore | Where-Object {$_.Name -eq "test-cbr0"}
            if ($n) {
                Start-Sleep -Seconds 5
                break
            }
            Start-Sleep -Seconds 1
        }
    }

    AfterEach {
        # clean HNS network
        Get-HnsNetwork -ErrorAction Ignore | Where-Object {$_.Name -eq "test-cbr0"} | Remove-HnsNetwork -ErrorAction Ignore
        while ($true) {
            $n = Get-HnsNetwork -ErrorAction Ignore | Where-Object {$_.Name -eq "test-cbr0"}
            if (-not $n) {
                Start-Sleep -Seconds 5
                break
            }
            Start-Sleep -Seconds 1
        }

        # clean wins server
        Get-Process -Name "wins" -ErrorAction Ignore | Stop-Process -Force -ErrorAction Ignore
    }

    It "get-network by name" {
        # wins.exe cli hns get-network --name "xxx"
        # docker run --name get-network --rm -v //./pipe/rancher_wins://./pipe/rancher_wins -v c:/etc/rancher/wins:c:/etc/rancher/wins wins-cli hns get-network --name test-cbr0
        $ret = Execute-Binary -FilePath "docker.exe" -ArgumentList @("run", "--rm", "-v", "//./pipe/rancher_wins://./pipe/rancher_wins", "-v", "c:/etc/rancher/wins:c:/etc/rancher/wins", "wins-cli", "hns", "get-network", "--name", "test-cbr0") -PassThru
        if (-not $ret.Ok) {
            Log-Error $ret.Output
            $false | Should Be $true
        }

        # verify
        $expectedObj = $ret.Output | ConvertFrom-Json
        $actualObj = Get-HnsNetwork | Where-Object Name -eq "test-cbr0"
        $expectedObj.Subnets[0].AddressCIDR -eq $actualObj.Subnets[0].AddressPrefix | Should Be $true
        $expectedObj.Subnets[0].GatewayAddress -eq $actualObj.Subnets[0].GatewayAddress | Should Be $true
    }

}
