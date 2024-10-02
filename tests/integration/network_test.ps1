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

function ConvertTo-MaskLength {
    param(
        [Parameter(Mandatory = $true, Position = 0)]
        [Net.IPAddress] $subnetMask
    )

    $bits = "$($subnetMask.GetAddressBytes() | % {
        [Convert]::ToString($_, 2)
    } )" -replace "[\s0]"

    return $bits.Length
}

function ConvertTo-DecimalIP {
    param(
        [Parameter(Mandatory = $true, Position = 0)]
        [Net.IPAddress] $ipAddress
    )

    $i = 3
    $decimalIP = 0

    $ipAddress.GetAddressBytes() | % {
        $decimalIP += $_ * [Math]::Pow(256, $i)
        $i--
    }

    return [UInt32]$decimalIP
}

function ConvertTo-DottedIP {
    param(
        [Parameter(Mandatory = $true, Position = 0)]
        [Uint32] $ipAddress
    )

    $dottedIP = $(for ($i = 3; $i -gt -1; $i--) {
            $base = [Math]::Pow(256, $i)
            $remainder = $ipAddress % $base
            ($ipAddress - $remainder) / $base
            $ipAddress = $remainder
        })

    return [String]::Join(".", $dottedIP)
}

Describe "network" {

    BeforeEach {
        # start wins server
        Execute-Binary -FilePath "bin\wins.exe" -ArgumentList @('srv', 'app', 'run') -Backgroud | Out-Null
        Wait-Ready -Path //./pipe/rancher_wins
    }

    AfterEach {
        # clean wins server
        Get-Process -Name "wins" -ErrorAction Ignore | Stop-Process -Force -ErrorAction Ignore
    }

    It "get default adapter" {
        # wins.exe cli network get
        # docker run --rm -v //./pipe/rancher_wins://./pipe/rancher_wins -v c:/etc/rancher/wins:c:/etc/rancher/wins wins-cli network get
        New-Directory "c:/etc/rancher/pipe"

        $ret = Execute-Binary -FilePath "docker.exe" -ArgumentList @("run", "--rm", "-v", "//./pipe/rancher_wins://./pipe/rancher_wins", "-v", "c:/etc/rancher/pipe:c:/etc/rancher/pipe", "wins-cli", "network", "get") -PassThru
        if (-not $ret.Ok) {
            Log-Error $ret.Output
            $false | Should -Be $true
        }

        # verify
        $defaultNetIndex = (Get-NetIPAddress -AddressFamily IPv4 -ErrorAction Ignore | Get-NetAdapter -ErrorAction Ignore | Get-NetRoute -DestinationPrefix "0.0.0.0/0" -ErrorAction Ignore | Select-Object -ExpandProperty ifIndex -First 1)
        $expectedObj = $ret.Output | ConvertFrom-Json
        $actaulObj = (Get-WmiObject -Class Win32_NetworkAdapterConfiguration -Filter "IPEnabled=True and InterfaceIndex=$defaultNetIndex" | Select-Object -Property DefaultIPGateway, DNSHostName, InterfaceIndex, IPAddress, IPSubnet)
        $actaulObjSubnetMask = ConvertTo-MaskLength $actaulObj.IPSubnet[0]
        $actaulObjSubnetAddr = ConvertTo-DottedIP ((ConvertTo-DecimalIP $actaulObj.IPAddress[0]) -band (ConvertTo-DecimalIP $actaulObj.IPSubnet[0]))
        $expectedObj.GatewayAddress -eq $actaulObj.DefaultIPGateway[0] | Should -Be $true
        $expectedObj.InterfaceIndex -eq $actaulObj.InterfaceIndex | Should -Be $true
        $expectedObj.SubnetCIDR -eq "$actaulObjSubnetAddr/$actaulObjSubnetMask" | Should -Be $true
    }

}
