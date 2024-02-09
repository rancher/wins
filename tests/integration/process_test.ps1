$ErrorActionPreference = "Stop"

Import-Module -Name @(
    "$PSScriptRoot\utils.psm1"
) -WarningAction Ignore

# clean interferences
try {
    Get-Process -Name "rancher-wins-*" -ErrorAction Ignore | Stop-Process -Force -ErrorAction Ignore
    Get-NetFirewallRule -PolicyStore ActiveStore -Name "rancher-wins-*" -ErrorAction Ignore | ForEach-Object { Remove-NetFirewallRule -Name $_.Name -PolicyStore ActiveStore -ErrorAction Ignore } | Out-Null
    Get-Process -Name "wins" -ErrorAction Ignore | Stop-Process -Force -ErrorAction Ignore
}
catch {
    Log-Warn $_.Exception.Message
}

Describe "process" {

    BeforeEach {
        # create nginx dir
        New-Item -ItemType Directory -Path "c:\etc\nginx" -Force -ErrorAction Ignore | Out-Null
    }

    AfterEach {
        # clean nginx dir
        Remove-Item -Path "c:\etc\nginx" -Recurse -Force -ErrorAction Ignore

        # stop nginx process & firewall rules
        Get-Process -Name "rancher-wins-*" -ErrorAction Ignore | Stop-Process -Force -ErrorAction Ignore
        Get-NetFirewallRule -PolicyStore ActiveStore -Name "rancher-wins-*" -ErrorAction Ignore | ForEach-Object { Remove-NetFirewallRule -Name $_.Name -PolicyStore ActiveStore -ErrorAction Ignore } | Out-Null
        Get-Process -Name "wins" -ErrorAction Ignore | Stop-Process -Force -ErrorAction Ignore
    }

    It "run" {
        # generated config
        $config = @{
            whiteList = @{
                processPaths = @(
                    "C:\otherpath"
                    "C:\etc\nginx\nginx.exe"
                )
            }
        }
        $config | ConvertTo-Json -Compress -Depth 32 | Out-File -NoNewline -Encoding utf8 -Force -FilePath "c:\etc\rancher\wins\config"
        $configJson = Get-Content -Raw -Path "c:\etc\rancher\wins\config"
        Log-Info $configJson

        # start wins server
        Execute-Binary -FilePath "bin\wins.exe" -ArgumentList @('srv', 'app', 'run') -Backgroud | Out-Null
        Wait-Ready -Path //./pipe/rancher_wins

        # wins.exe cli prc run --path xxx --exposes xxx
        # docker run --name prc-run --rm -v //./pipe/rancher_wins://./pipe/rancher_wins -v c:/etc/rancher/wins:c:/etc/rancher/wins -v c:/etc/nginx:c:/host/etc/nginx wins-nginx
        Execute-Binary -FilePath "docker.exe" -ArgumentList @("run", "--name", "prc-run", "--rm", "-v", "//./pipe/rancher_wins://./pipe/rancher_wins", "-v", "c:/etc/rancher/wins:c:/etc/rancher/wins", "-v", "c:/etc/nginx:c:/host/etc/nginx", "wins-nginx") -Backgroud
        {
            Wait-Ready -Path "c:\etc\nginx\rancher-wins-nginx.exe" -Throw
        } | Should -Not -Throw

        # verify running
        {
            # should be abled to find processes
            {
                Get-Process -Name "rancher-wins-*" -ErrorAction Ignore
            } | Judge -Throw -Timeout 120
        } | Should -Not -Throw
        $statusCode = $(curl.exe -sL -w "%{http_code}" -o /dev/null http://127.0.0.1)
        $statusCode | Should -Be 200
        Get-NetFirewallRule -PolicyStore ActiveStore -Name "rancher-wins-*-TCP-80" -ErrorAction Ignore | Should -Not -BeNullOrEmpty

        # verify stopping
        Execute-Binary -FilePath "docker.exe" -ArgumentList @("rm", "-f", "prc-run") -PassThru | Out-Null
        {
            # should not be abled to find processes
            {
                Get-Process -Name "rancher-wins-*" -ErrorAction Ignore
            } | Judge -Reverse -Throw
        } | Should -Not -Throw
        Get-NetFirewallRule -PolicyStore ActiveStore -Name "rancher-wins-*-TCP-80" -ErrorAction Ignore | Should -BeNullOrEmpty
    }

    It "run not in whitelist" {
        # generated config
        $config = @{
            white_List = @{
                processPaths = @(
                    "C:\otherpath"
                )
            }
        }
        $config | ConvertTo-Json -Compress -Depth 32 | Out-File -NoNewline -Encoding utf8 -Force -FilePath "c:\etc\rancher\wins\config"
        $configJson = Get-Content -Raw -Path "c:\etc\rancher\wins\config"
        Log-Info $configJson

        # start wins server
        Execute-Binary -FilePath "bin\wins.exe" -ArgumentList @('srv', 'app', 'run') -Backgroud | Out-Null
        Wait-Ready -Path //./pipe/rancher_wins

        # wins.exe cli prc run --path xxx --exposes xxx
        # docker run --name prc-run --rm -v //./pipe/rancher_wins://./pipe/rancher_wins -v c:/etc/rancher/wins:c:/etc/rancher/wins -v c:/etc/nginx:c:/host/etc/nginx wins-nginx
        Execute-Binary -FilePath "docker.exe" -ArgumentList @("run", "--name", "prc-run", "--rm", "-v", "//./pipe/rancher_wins://./pipe/rancher_wins", "-v", "c:/etc/rancher/wins:c:/etc/rancher/wins", "-v", "c:/etc/nginx:c:/host/etc/nginx", "wins-nginx") -Backgroud
        {
            Wait-Ready -Timeout 3 -Path "c:\etc\nginx\rancher-wins-nginx.exe" -Throw
        } | Should -Throw

        # verify
        {
            # should be abled to find processes
            {
                Get-Process -Name "rancher-wins-*" -ErrorAction Ignore
            } | Judge -Timeout 3 -Throw
        } | Should -Throw
        Get-NetFirewallRule -PolicyStore ActiveStore -Name "rancher-wins-*-TCP-80" -ErrorAction Ignore | Should -BeNullOrEmpty
    }
}
