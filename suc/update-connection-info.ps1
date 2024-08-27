<#
.SYNOPSIS
    Updates the rancher_connection_info.json file on Windows nodes
.DESCRIPTION
    This script takes sections of the install.ps1 script responsible for for updating the
    rancher2_connection_info.json file. This script is intended to be exclusively used
    by the Rancher SUC plan deployed to Windows nodes. This script must be run in a host
    process container, so that the underlying host can be updated.
#>

function Write-LogInfo {
    Write-Host -NoNewline "INFO: "
    Write-Host ($args -join " ")
}
function Write-LogWarn {
    Write-Host -NoNewline "WARN: "
    Write-Host ($args -join " ")
}
function Write-LogError {
    Write-Host -NoNewline "ERROR: "
    Write-Host ($args -join " ")
}
function Write-LogFatal {
    Write-Host -NoNewline "FATA: "
    Write-Host ($args -join " ")
    exit 255
}

function Write-LogDebug {
    if (-Not $env:CATTLE_WINS_DEBUG) {
        Write-Host -NoNewline "DEBUG: "
        Write-Host ($args -join " ")
    }
}

function Test-CaCheckSum() {
    $caCertsPath = "cacerts"
    $env:RANCHER_CERT = Join-Path -Path $env:CATTLE_AGENT_CONFIG_DIR -ChildPath "ranchercert"
    if (-Not $env:CATTLE_CA_CHECKSUM) {
        return
    }

    curl.exe --insecure -sfL $env:CATTLE_SERVER/$caCertsPath -o $env:RANCHER_CERT
    if (-Not(Test-Path -Path $env:RANCHER_CERT)) {
        Write-Error "The environment variable CATTLE_CA_CHECKSUM is set but there is no CA certificate configured at $( $env:CATTLE_SERVER )/$( $caCertsPath )) "
        exit 1
    }

    if ($LASTEXITCODE -ne 0) {
        Write-Error "Value from $( $env:CATTLE_SERVER )/$( $caCertsPath ) does not look like an x509 certificate, exited with $( $LASTEXITCODE ) "
        Write-Error "Retrieved cacerts:"
        Get-Content $env:RANCHER_CERT
        exit 1
    }
    else {
        Write-LogInfo "Value from $( $env:CATTLE_SERVER )/$( $caCertsPath ) is an x509 certificate"
    }
    $env:CATTLE_SERVER_CHECKSUM = (Get-FileHash -Path $env:RANCHER_CERT -Algorithm SHA256).Hash.ToLower()
    Write-LogInfo "Checking CATTLE_CA_CHECKSUM"
    if ($env:CATTLE_SERVER_CHECKSUM -ne $env:CATTLE_CA_CHECKSUM) {
        Remove-Item -Path $env:RANCHER_CERT -Force
        Write-LogError "Configured cacerts checksum $( $env:CATTLE_SERVER_CHECKSUM ) does not match given -CaCheckSum $( $env:CATTLE_CA_CHECKSUM ) "
        Write-LogError "Please check if the correct certificate is configured at $( $env:CATTLE_SERVER )/$( $caCertsPath ) ."
        exit 1
    }
    Import-Certificate -FilePath $env:RANCHER_CERT -CertStoreLocation Cert:\LocalMachine\Root | Out-Null
}

function Test-RancherConnection {
    $env:RANCHER_SUCCESS = $false
    $retries = 0
    while ($retries -lt 6) {
        $responseCode = $(curl.exe --connect-timeout 60 --max-time 60 -k --write-out "%{http_code}\n" $env:CURL_CAFLAG -sfL "$env:CATTLE_SERVER/healthz")
        switch ( $responseCode ) {
            { $_ -in "ok200", 200 } {
                Write-LogInfo "Successfully tested Rancher connection."
                $env:RANCHER_SUCCESS = $true
                $retries = 99
                break
            }
            default {
                Write-LogError "$responseCode received while testing Rancher connection. Sleeping for 5 seconds and trying again."
                Start-Sleep -Seconds 5
                $retries++
                continue
            }
        }
    }
}

function Test-CaRequired {
    $env:CA_REQUIRED = $false
    if ($env:CATTLE_SERVER) {
        $retries = 0
        while ($retries -lt 6) {
            curl.exe --connect-timeout 60 --max-time 60 -sfL "$env:CATTLE_SERVER/healthz"
            Write-LogDebug "Received curl exit code $EXITCODE"
            switch ($LASTEXITCODE) {
                0 {
                    Write-LogInfo "Determined CA is not necessary to connect to Rancher."
                    $env:CATTLE_CA_CHECKSUM = ""
                    $retries = 99
                    break
                }
                { $_ -in 60, 77, 35 } {
                    Write-LogInfo "Determined CA is necessary to connect to Rancher."
                    $env:CA_REQUIRED = $true
                    $retries = 99
                    break
                }
                default {
                    Write-LogError "Error while connecting to Rancher to verify CA necessity. Sleeping for 5 seconds and trying again. Received error code $LASTEXITCODE"
                    Start-Sleep -Seconds 5
                    $retries++
                    continue
                }
            }
        }
    } else {
        Write-LogWarn "`$env:CATTLE_SERVER was not provided, cannot determine if a CA is required"
    }
}

function Get-RancherConnectionInfo() {
    $retries = 0
    Write-LogInfo "Attempting to get updated Rancher connection info"
    while ($retries -lt 6) {
        $responseCode = $(curl.exe --connect-timeout 60 --max-time 60 --write-out "%{http_code}\n " --ssl-no-revoke -sfL "$env:CATTLE_SERVER/v3/connect/agent" -o $env:CATTLE_AGENT_VAR_DIR/rancher2_connection_info.json -H "Authorization: Bearer $($env:CATTLE_TOKEN)" -H "X-Cattle-Id: $($env:CATTLE_ID)" -H "Content-Type: application/json")
        switch ( $responseCode ) {
            { $_ -in "ok200", 200 } {
                Write-LogInfo "Successfully downloaded Rancher connection information."
                $retries = 99
                break
            }
            default {
                Write-LogError "$responseCode received while downloading Rancher connection information. Sleeping for 5 seconds and trying again."
                Start-Sleep -Seconds 5
                $retries++
                continue
            }
        }
    }
}

function Update-ConnectionInfo()
{
    $env:RKE2_DATA_DIR = "c:\var\lib\rancher"
    $env:CATTLE_AGENT_CONFIG_DIR = "c:\etc\rancher\wins"

    $env:CATTLE_AGENT_VAR_DIR = "$env:RKE2_DATA_DIR\agent"
    $env:CATTLE_ID = Get-Content -Path "$env:CATTLE_AGENT_CONFIG_DIR\cattle-id"

    Test-RancherConnection
    if ($env:CATTLE_CA_CHECKSUM)
    {
        Write-LogDebug "Detected CATTLE_CA_CHECKSUM ($env:CATTLE_CA_CHECKSUM), will confirm CA necessity and validity"
        Test-CaRequired
        if ($env:CA_REQUIRED)
        {
            Test-CaCheckSum
        }
    }

    if ($env:CATTLE_TOKEN)
    {
        Get-RancherConnectionInfo
    } else {
        Write-LogWarn "`$env:CATTLE_TOKEN is not present, will not retrieve connection information from rancher server"
    }
}

Update-ConnectionInfo