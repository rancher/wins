$ErrorActionPreference = "Stop"

Import-Module -Name @(
    "$PSScriptRoot\utils.psm1"
) -WarningAction Ignore

$StartMockRancherHandler = {
    param(
        [Parameter()]
        [string]
        $certs
    )

    $http = New-Object System.Net.HttpListener
    $http.Prefixes.Add("http://localhost:8080/")
    $http.Start()

    while($http.IsListening) {
        $ctx = $http.GetContext()
        if ($ctx.Request.RawUrl -eq "/cacerts") {
            $buf = [System.Text.Encoding]::UTF8.GetBytes($certs)
            $ctx.response.ContentLength64 = $buf.Length
            $ctx.Response.OutputStream.Write($buf, 0, $buf.Length)
            $ctx.Response.OutputStream.Close()
        }
        # A dedicated kill endpoint works around a deadlock
        # that is encountered when Stop-Job is invoked at the same itme
        # that this function is waiting on GetContext()
        if ($ctx.Request.RawUrl -eq "/kill") {
            $ctx.Response.OutputStream.Close()
            exit 0
        }
    }
}

Describe "Install script certificate tests" {
    BeforeEach {
        # Create a test specific copy of the install script
        # as the environment variables being set may differ between tests
        Copy-Item install.ps1 install-certs-test.ps1 -Force

        # note: Simply running the install script does not do anything. During normal provisioning,
        # Rancher will mutate the install script to both add environment variables, and to call
        # the primary function 'Invoke-WinsInstaller'. As this is an integration test, we need to manually
        # update the install script ourselves.
        Add-Content -Path ./install-certs-test.ps1 -Value '$env:CATTLE_REMOTE_ENABLED = "false"'
        Add-Content -Path ./install-certs-test.ps1 -Value '$env:CATTLE_LOCAL_ENABLED = "true"'
        Add-Content -Path install-certs-test.ps1 -Value "`$env:CATTLE_SERVER = `"http://localhost:8080`""
        # reset the agent directory
        Remove-Item "C:/etc/rancher/wins" -Force -ErrorAction Ignore
    }

    AfterEach {
        Cleanup-CertFile
        $env:CATTLE_CA_CHECKSUM = ""
        $env:CATTLE_SERVER = ""
        Log-Info "Running uninstall script"
        try {
            # note: since this script may not be run by an administrator, it's possible that it might fail
            # on trying to delete certain files with ACLs attached to them.
            # If you are running this locally, make sure you run with admin privileges.
            .\uninstall.ps1
        } catch {
            Log-Warn "You need to manually run uninstall.ps1, encountered error: $($_.Exception.Message)"
        }
    }

    It "Imports a single cert properly" {
        Log-Info "TEST: [Imports a single cert properly]"
        $expectedCertificates = 1
        $certData = Setup-CertFiles -length $expectedCertificates
        $certData.ThumbPrints.Length | Should -BeExactly $expectedCertificates

        # Quick sanity check to ensure utility function properly removed
        # certificates from the built-in stores
        Log-Info "Ensuring that certs are not yet imported"
        check-thumbprints -shouldExist $false -thumbs $certData.ThumbPrints

        $checkSum = $certData.Checksum
        Add-Content -Path install-certs-test.ps1 -Value "`$env:CATTLE_CA_CHECKSUM = `"$checkSum`""
        Add-Content -Path install-certs-test.ps1 -Value "Invoke-WinsInstaller"

        invoke-installScript

        Log-Info "Confirming that certs have been properly imported..."
        check-thumbprints -shouldExist $true -thumbs $certData.ThumbPrints

        Log-Info "Properly imported $expectedCertificates certificates"
    }

    It "Imports a chain properly" {
        Log-Info "TEST: [Imports a chain properly]"
        $expectedCertificates = 3
        $certData = Setup-CertFiles -length $expectedCertificates
        $certData.ThumbPrints.Length | Should -BeExactly $expectedCertificates

        # Quick sanity check to ensure utility function properly removed
        # certificates from the built-in stores
        Log-Info "Ensuring that certs are not yet imported"
        check-thumbprints -shouldExist $false -thumbs $certData.ThumbPrints

        $checkSum = $certData.Checksum
        Add-Content -Path install-certs-test.ps1 -Value "`$env:CATTLE_CA_CHECKSUM = `"$checkSum`""
        Add-Content -Path install-certs-test.ps1 -Value "Invoke-WinsInstaller"

        invoke-installScript

        Log-Info "Confirming that certs have been properly imported..."
        check-thumbprints -shouldExist $true -thumbs $certData.ThumbPrints

        Log-Info "Properly imported $expectedCertificates certificates"
    }

    # utility functions only useful for this set of tests
    BeforeAll {
        function invoke-installScript() {
            param (
                [Parameter()]
                [String]
                $certs
            )

            Log-Info "Starting mock Rancher API"
            $job = Start-Job -ScriptBlock $StartMockRancherHandler -ArgumentList $certData.FinalCertBlocks
            Start-Sleep 1
            if ($job.State -ne "Running") {
                # display job output to help debug job failure
                Log-Error "Mock Rancher server failed to start"
                $job | Receive-Job
                $job.State | Should -Be -ExpectedValue "Running"
            }

            Log-Info "Invoking install script"
            .\install-certs-test.ps1
            $installScriptExitCode = $LASTEXITCODE

            Log-Info "Stopping mock server"
            curl.exe -sS http://localhost:8080/kill
            Remove-Job -Id $job.Id -Force

            Log-Info "Install script exited with code $installScriptExitCode"
            $installScriptExitCode | Should -Be -ExpectedValue 0
        }

        function check-thumbprints() {
            param (
                [Parameter()]
                [Boolean]
                $shouldExist,

                [Parameter()]
                [String[]]
                $thumbs
            )

            $expect = 1
            if (-Not $shouldExist) {
                $expect = 0
            }
            $certStore = [System.Security.Cryptography.X509Certificates.X509Store]::new([System.Security.Cryptography.X509Certificates.StoreName]::Root, "LocalMachine")
            $certStore.Open([System.Security.Cryptography.X509Certificates.OpenFlags]::MaxAllowed)
            foreach ($thumbPrint in $thumbs)
            {
                Log-Info "Checking $thumbPrint, expecting $expect instances to exist"
                $found = $certStore.Certificates.Find('FindByThumbprint', $thumbPrint, $false)
                $count = $found.Count
                if ($count -ne $expect)
                {
                    Log-Error "Found unexpected count of cert with thumb print of $thumbPrint, expected $expect, found $count"
                    $found.Count | Should -Be -ExpectedValue $expect
                }
                Log-Info "Found expected number of entries for cert with thumbprint $thumbPrint"
            }
            $certStore.Close()
        }
    }
}
