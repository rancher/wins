<#
.SYNOPSIS 
    Installs Rancher Wins to create Windows Worker Nodes.
.DESCRIPTION 
    Run the script to install all Rancher Wins related needs.
.NOTES
    Environment variables:
      System Agent Variables
      - CATTLE_AGENT_LOGLEVEL (default: debug)
      - CATTLE_AGENT_CONFIG_DIR (default: C:/etc/rancher/agent)
      - CATTLE_AGENT_VAR_DIR (default: C:/var/lib/rancher/agent)
      Rancher 2.6+ Variables
      - CATTLE_SERVER
      - CATTLE_TOKEN
      - CATTLE_CA_CHECKSUM
      - CATTLE_ROLE_CONTROLPLANE=false
      - CATTLE_ROLE_ETCD=false
      - CATTLE_ROLE_WORKER=false
      - CATTLE_LABELS
      - CATTLE_TAINTS
      Advanced Environment Variables
      - CATTLE_AGENT_BINARY_URL (default: latest GitHub release)
      - CATTLE_PRESERVE_WORKDIR (default: false)
      - CATTLE_REMOTE_ENABLED (default: true)
      - CATTLE_LOCAL_ENABLED (default: false)
      - CATTLE_ID (default: autogenerate)
      - CATTLE_AGENT_BINARY_LOCAL (default: false)
      - CATTLE_AGENT_BINARY_LOCAL_LOCATION (default: )
      - CSI_PROXY_URL (default: )
      - CSI_PROXY_VERSION (default: )
      - CSI_PROXY_KUBELET_PATH (default: )
.EXAMPLE

#>
#Make sure this params matches the CmdletBinding below
param (
    [Parameter()]
    [String]
    $Address,
    [Parameter()]
    [String]
    $CaChecksum,
    [Parameter()]
    [String]
    $InternalAddress,
    [Parameter()]
    [String]
    $Label,
    [Parameter()]
    [String]
    $NodeName,
    [Parameter()]
    [String]
    $Server,
    [Parameter()]
    [String]
    $Taint,
    [Parameter()]
    [String]
    $Token,
    [Parameter()]
    [Switch]
    $Worker,
    [Parameter()]
    [Switch]
    $StrictTlsVerification
)
$ErrorActionPreference = 'Stop'
Set-StrictMode -Version Latest

$FALLBACK = "v0.4.15"

function Invoke-WinsInstaller {
    [CmdletBinding()]
    param (
        [Parameter()]
        [String]
        $Address,
        [Parameter()]
        [String]
        $CaChecksum,
        [Parameter()]
        [String]
        $InternalAddress,
        [Parameter()]
        [String]
        $Label,
        [Parameter()]
        [String]
        $NodeName,
        [Parameter()]
        [String]
        $Server,
        [Parameter()]
        [String]
        $Taint,
        [Parameter()]
        [String]
        $Token,
        [Parameter()]
        [Switch]
        $Worker
    )

    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls -bor [Net.SecurityProtocolType]::Tls11 -bor [Net.SecurityProtocolType]::Tls12

    function Write-LogInfo {
        Write-Host -NoNewline -ForegroundColor Blue "INFO: "
        Write-Host -ForegroundColor Gray ("{0,-44}" -f ($args -join " "))
    }
    function Write-LogWarn {
        Write-Host -NoNewline -ForegroundColor DarkYellow "WARN: "
        Write-Host -ForegroundColor Gray ("{0,-44}" -f ($args -join " "))
    }
    function Write-LogError {
        Write-Host -NoNewline -ForegroundColor DarkRed "ERROR: "
        Write-Host -ForegroundColor Gray ("{0,-44}" -f ($args -join " "))
    }
    function Write-LogFatal {
        Write-Host -NoNewline -ForegroundColor DarkRed "FATA: "
        Write-Host -ForegroundColor Gray ("{0,-44}" -f ($args -join " "))
        exit 255
    }

    function Get-StringHash {
        [CmdletBinding()]
        param (
            [Parameter()]
            [string]
            $Value
        )
        $stringAsStream = [System.IO.MemoryStream]::new()
        $writer = [System.IO.StreamWriter]::new($stringAsStream)
        $writer.write($Value)
        $writer.Flush()
        $stringAsStream.Position = 0
        return (Get-FileHash -InputStream $stringAsStream -Algorithm SHA256).Hash.ToLower()
    }

    function Get-Args {
        if ($Address) {
            $env:CATTLE_ADDRESS = $Address
        }

        if ($CaChecksum) {
            $env:CATTLE_CA_CHECKSUM = $CaChecksum
        }

        if ($InternalAddress) {
            $env:CATTLE_INTERNAL_ADDRESS = $InternalAddress
        }

        if ($Label) {
            if ($env:CATTLE_LABELS) {
                $env:CATTLE_LABELS += ",$Label"
            }
            else {
                $env:CATTLE_LABELS = $Label
            }
        }

        if ($NodeName) {
            $env:CATTLE_NODE_NAME = $NodeName
        }

        if ($Server) {
            $env:CATTLE_SERVER = $Server
        }

        if ($Taint) {
            if ($env:CATTLE_TAINTS) {
                $env:CATTLE_TAINTS += ",$Taint"
            }
            else {
                $env:CATTLE_TAINTS = $Taint
            }
        }

        if ($Token) {
            $env:CATTLE_TOKEN = $Token
        }

        if ($Worker) {
            $env:CATTLE_ROLE_WORKER = "true"
        }
    }

    function Set-Path {
        $env:PATH += ";C:\var\lib\rancher\rke2\bin;C:\usr\local\bin"
        $environment = [System.Environment]::GetEnvironmentVariable("Path", "Machine")
        $environment = $environment.Insert($environment.Length, ";C:\var\lib\rancher\rke2\bin;C:\usr\local\bin")
        [System.Environment]::SetEnvironmentVariable("Path", $environment, "Machine")
    }

    function Set-Environment {
        $env:CURL_CAFLAG = "--ssl-no-revoke"

        if (-Not $env:CATTLE_ROLE_CONTROLPLANE) {
            $env:CATTLE_ROLE_CONTROLPLANE = "false"
        }

        if (-Not $env:CATTLE_ROLE_ETCD) {
            $env:CATTLE_ROLE_ETCD = "false"
        }

        if (-Not $env:CATTLE_ROLE_WORKER) {
            $env:CATTLE_ROLE_WORKER = "false"
        }

        if (-Not $env:CATTLE_REMOTE_ENABLED) {
            $env:CATTLE_REMOTE_ENABLED = "true"
        }
        else {
            $env:CATTLE_REMOTE_ENABLED = $env:CATTLE_REMOTE_ENABLED.ToLower()
        }

        if (-Not $env:CATTLE_LOCAL_ENABLED) {
            $env:CATTLE_LOCAL_ENABLED = "false"
        } else {
            $env:CATTLE_LOCAL_ENABLED = $env:CATTLE_LOCAL_ENABLED.ToLower()
        }

        if (-Not $env:CATTLE_PRESERVE_WORKDIR) {
            $env:CATTLE_PRESERVE_WORKDIR = "false"
        }
        else {
            $env:CATTLE_PRESERVE_WORKDIR = $env:CATTLE_PRESERVE_WORKDIR.ToLower()
        }

        if (-Not $env:CATTLE_AGENT_LOGLEVEL) {
            $env:CATTLE_AGENT_LOGLEVEL = "debug"
        }
        else {
            $env:CATTLE_AGENT_LOGLEVEL = $env:CATTLE_AGENT_LOGLEVEL.ToLower()
        }

        if (-Not $env:STRICT_VERIFY) {
            $env:STRICT_VERIFY = "false"
        }
        if ($StrictTlsVerification -eq $true) {
            $env:STRICT_VERIFY = "true"
        }

        if ($env:CATTLE_AGENT_BINARY_LOCAL -eq "true") {
            if (-Not $env:CATTLE_AGENT_BINARY_LOCAL_LOCATION) {
                Write-LogFatal "No local binary location was specified"
            }
            $env:BINARY_SOURCE = "local"
        }
        else {
            $env:BINARY_SOURCE = "remote"
            if (-Not $env:CATTLE_AGENT_BINARY_URL -and $env:CATTLE_AGENT_BINARY_BASE_URL) {
                $env:CATTLE_AGENT_BINARY_URL = "$env:CATTLE_AGENT_BINARY_BASE_URL/wins.exe"
            }

            if (-Not $env:CATTLE_AGENT_BINARY_URL) {
                $rateLimit = $(curl.exe --connect-timeout 60 --max-time 300 $env:CURL_CAFLAG -sfL "https://api.github.com/rate_limit") | ConvertFrom-Json
               
                if ($rateLimit.rate.remaining -eq 0) {
                    Write-LogInfo "Error contacting GitHub to retrieve the latest version, falling back to version: $FALLBACK"
                    $env:VERSION = $FALLBACK
                }
                else {
                    try {
                        $env:VERSION = $(curl.exe --connect-timeout 60 $env:CURL_CAFLAG -sfL "https://api.github.com/repos/rancher/wins/releases/latest" | ConvertFrom-Json).tag_name
                    }
                    catch {
                        Write-LogInfo "Error contacting GitHub to retrieve the latest version, falling back to version: $FALLBACK"
                        $env:VERSION = $FALLBACK
                    }
                }

                $env:CATTLE_AGENT_BINARY_URL = "https://github.com/rancher/wins/releases/download/$env:VERSION/wins.exe"
                $env:BINARY_SOURCE = "upstream"
            }
        }

        if ($env:CATTLE_REMOTE_ENABLED -eq "true") {
            if (-Not $env:CATTLE_TOKEN) {
                Write-LogFatal "Environment variable CATTLE_TOKEN was not set. Will not retrieve a remote connection configuration from Rancher2"
            }
            if (-Not $env:CATTLE_SERVER) {
                Write-LogFatal "Environment variable CATTLE_SERVER was not set"
            }
        }

        if (($env:CATTLE_REMOTE_ENABLED -eq "true") -and ($env:CATTLE_LOCAL_ENABLED -eq "true")){
            Write-LogFatal "Both CATTLE_LOCAL_ENABLED and CATTLE_REMOTE_ENABLED were enabled, exiting as only one can be enabled"
        }

        if (($env:CATTLE_REMOTE_ENABLED -eq "false") -and ($env:CATTLE_LOCAL_ENABLED -eq "false")){
            Write-LogFatal "Neither CATTLE_LOCAL_ENABLED nor CATTLE_REMOTE_ENABLED were enabled, exiting as one must be enabled"
        }

        if (-Not $env:CATTLE_AGENT_CONFIG_DIR) {
            $env:CATTLE_AGENT_CONFIG_DIR = "C:/etc/rancher/wins"
            Write-LogInfo "Using default agent configuration directory $( $env:CATTLE_AGENT_CONFIG_DIR )"
        }
        if (-Not (Test-Path $env:CATTLE_AGENT_CONFIG_DIR)) {
            New-Item -Path $env:CATTLE_AGENT_CONFIG_DIR -ItemType Directory -Force | Out-Null
        }

        # copy powershell for wins
        Copy-Item $($(Get-Command powershell).Source) "$env:CATTLE_AGENT_CONFIG_DIR/powershell.exe"

        if (-Not $env:CATTLE_AGENT_VAR_DIR) {          
            $env:CATTLE_AGENT_VAR_DIR = "C:/var/lib/rancher/agent"
            Write-LogInfo "Using default agent var directory $( $env:CATTLE_AGENT_VAR_DIR )"
        }
        if (-Not (Test-Path $env:CATTLE_AGENT_VAR_DIR)) {
            New-Item -Path $env:CATTLE_AGENT_VAR_DIR -ItemType Directory -Force | Out-Null
        }

        if (-Not $env:CATTLE_AGENT_BIN_PREFIX) {
            $env:CATTLE_AGENT_BIN_PREFIX = "c:/usr/local"
        }
        if (-Not (Test-Path $env:CATTLE_AGENT_BIN_PREFIX)) {
            New-Item -Path $env:CATTLE_AGENT_BIN_PREFIX -ItemType Directory -Force | Out-Null
        }

        $env:CATTLE_ADDRESS = Get-Address -Value $env:CATTLE_ADDRESS
        $env:CATTLE_INTERNAL_ADDRESS = Get-Address -Value $env:CATTLE_INTERNAL_ADDRESS
    }

    function Test-Architecture() {
        if ($env:PROCESSOR_ARCHITECTURE -ne "AMD64") {
            Write-LogFatal "Unsupported architecture $( $env:PROCESSOR_ARCHITECTUR )"
        }
    }

    function Invoke-WinsAgentDownload() {
        if (-Not (Test-Path "$env:CATTLE_AGENT_BIN_PREFIX/bin")) {
            New-Item -Path "$env:CATTLE_AGENT_BIN_PREFIX/bin" -ItemType Directory -Force | Out-Null
        }
        
        if ($env:CATTLE_AGENT_BINARY_LOCAL -eq "true") {
            Write-LogInfo "Using local Wins installer from $($env:CATTLE_AGENT_BINARY_LOCAL_LOCATION)"
            Copy-Item -Path $env:CATTLE_AGENT_BINARY_LOCAL_LOCATION -Destination "$env:CATTLE_AGENT_BIN_PREFIX/bin/wins.exe"
        }
        else {
            Write-LogInfo "Downloading Wins from $($env:CATTLE_AGENT_BINARY_URL)"
            if ($env:BINARY_SOURCE -ne "upstream") {
                $env:CURL_BIN_CAFLAG = $env:CURL_CAFLAG
            }
            else {
                $env:CURL_BIN_CAFLAG = ""
            }

            $retries = 0  
            while ($retries -lt 6) {
                $responseCode = $(curl.exe --connect-timeout 60 --max-time 300 --write-out "%{http_code}\n" $env:CURL_BIN_CAFLAG -sfL "$($env:CATTLE_AGENT_BINARY_URL)" -o "$env:CATTLE_AGENT_BIN_PREFIX/bin/wins.exe")
                
                switch ( $responseCode ) {
                    { "ok200", 200 } {
                        Write-LogInfo "Successfully downloaded the wins binary." 
                        $retries = 99
                        break
                    }
                    default {
                        Write-LogError "$responseCode received while downloading the wins binary. Sleeping for 5 seconds and trying again." 
                        Start-Sleep -Seconds 5
                        $retries++
                        continue
                    }
                }
            }
        }
        if (-Not (Test-Path "$env:CATTLE_AGENT_BIN_PREFIX/bin/wins.exe")) {
            Write-LogFatal "Wins.exe doesn't appear to have been installed."
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
        if ($env:CATTLE_SERVER -and ($env:CATTLE_REMOTE_ENABLED -eq "true")) {
            $retries = 0  
            while ($retries -lt 6) {
                $responseCode = $(curl.exe --connect-timeout 60 --max-time 60 --write-out "%{http_code}\n" $env:CURL_CAFLAG -sfL "$env:CATTLE_SERVER/healthz")
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
            if (!$env:RANCHER_SUCCESS) {
                Write-LogFatal "Error connecting to Rancher. Perhaps -CaCheckSum needs to be set?"
            }
        }
    }

    function Test-CaRequired {
        $env:CA_REQUIRED = $false
        if ($env:CATTLE_SERVER -and ($env:CATTLE_REMOTE_ENABLED -eq "true")) {
            $retries = 0  
            while ($retries -lt 6) {
                curl.exe --connect-timeout 60 --max-time 60 -sfL "$env:CATTLE_SERVER/healthz"
                switch ($LASTEXITCODE) {
                    0 {
                        Write-LogInfo "Determined CA is not necessary to connect to Rancher." 
                        $env:CATTLE_CA_CHECKSUM = ""
                        $retries = 99
                        break
                    }                    
                    { $_ -in 60, 77 } {
                        Write-LogInfo "Determined CA is necessary to connect to Rancher."
                        $env:CA_REQUIRED = $true
                        $retries = 99
                        break
                    }
                    default {
                        Write-LogError "Error while connecting to Rancher to verify CA necessity. Sleeping for 5 seconds and trying again."
                        Start-Sleep -Seconds 5
                        $retries++
                        continue
                    }
                }
            }
        }
    }

    function Get-RancherConnectionInfo() {
        if ($env:CATTLE_REMOTE_ENABLED -eq "true") {
            $retries = 0              
            while ($retries -lt 6) {
                $responseCode = $(curl.exe --connect-timeout 60 --max-time 60 --write-out "%{http_code}\n" $env:CURL_CAFLAG -sfL "$env:CATTLE_SERVER/v3/connect/agent" -o $env:CATTLE_AGENT_VAR_DIR/rancher2_connection_info.json -H "Authorization: Bearer $($env:CATTLE_TOKEN)" -H "X-Cattle-Node-Name: $($env:CATTLE_NODE_NAME)" -H "X-Cattle-Id: $($env:CATTLE_ID)" -H "X-Cattle-Role-Worker: $($env:CATTLE_ROLE_WORKER)" -H "X-Cattle-Labels: $($env:CATTLE_LABELS)" -H "X-Cattle-Taints: $($env:CATTLE_TAINTS)" -H "X-Cattle-Address: $($env:CATTLE_ADDRESS)" -H "X-Cattle-Internal-Address: $($env:CATTLE_INTERNAL_ADDRESS)" -H "Content-Type: application/json")
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
            Set-RestrictedPermissions -Path $env:CATTLE_AGENT_VAR_DIR/rancher2_connection_info.json
        }
    }

    function Set-WinsConfig() {
        $winsConfig =
        @"
white_list:
  processPaths:
    - $($env:CATTLE_AGENT_CONFIG_DIR)/powershell.exe
    - C:/etc/wmi-exporter/wmi-exporter.exe
    - C:/etc/windows-exporter/windows-exporter.exe
  proxyPorts:
    - 9796
"@
        # Overwrite the existing contents of the file using 'Set-Content'
        Set-Content -Path $env:CATTLE_AGENT_CONFIG_DIR/config -Value $winsConfig

        $agentConfig = 
        @"
agentStrictTLSMode: $(($env:STRICT_VERIFY).ToString().ToLower())
systemagent:
  workDirectory: $($env:CATTLE_AGENT_VAR_DIR)/work
  appliedPlanDirectory: $($env:CATTLE_AGENT_VAR_DIR)/applied
  remoteEnabled: $($env:CATTLE_REMOTE_ENABLED)
  localEnabled: $($env:CATTLE_LOCAL_ENABLED)
  preserveWorkDirectory: $($env:CATTLE_PRESERVE_WORKDIR)
"@
        Add-Content -Path $env:CATTLE_AGENT_CONFIG_DIR/config -Value $agentConfig
        if ($env:CATTLE_REMOTE_ENABLED -eq "true") {
            Add-Content -Path $env:CATTLE_AGENT_CONFIG_DIR/config -Value "  connectionInfoFile: $env:CATTLE_AGENT_VAR_DIR/rancher2_connection_info.json"
        }
        if ((Test-Path -Path $env:RANCHER_CERT) -and ($env:CA_REQUIRED -eq "true")) {
            $tlsConfig =
            @"
            tls-config:
                certFilePath: $($($env:RANCHER_CERT).Replace("\\","/"))
"@
            Add-Content -Path $env:CATTLE_AGENT_CONFIG_DIR/config -Value $tlsConfig
        }
        Set-RestrictedPermissions -Path $env:CATTLE_AGENT_CONFIG_DIR/config
    }

    function Set-CsiProxyConfig() {
        $proxyConfig = 
        @"
csi-proxy:
  url: $($env:CSI_PROXY_URL)
  version: $($env:CSI_PROXY_VERSION)
  kubeletPath: $($env:CSI_PROXY_KUBELET_PATH)
"@
        Add-Content -Path $env:CATTLE_AGENT_CONFIG_DIR/config -Value $proxyConfig
    }

    function Stop-Agent() { 
        [CmdletBinding()]
        param (
            [Parameter()]
            [string]
            $ServiceName
        )       
        Write-LogInfo "Checking if $ServiceName service exists"
        if ((Get-Service -Name $ServiceName -ErrorAction SilentlyContinue)) {
            Write-LogInfo "$ServiceName service found, stopping now"
            Stop-Service -Name $ServiceName
            while ((Get-Service -Name $ServiceName).Status -ne 'Stopped') {
                Write-LogInfo "Waiting for $ServiceName service to stop"
                Start-Sleep -s 5
            }
        }
        else {
            Write-LogInfo "$ServiceName isn't installed, continuing"
        }
    }

    function New-CattleId() {
        if (-Not $env:CATTLE_ID) {
            Write-LogInfo "Generating Cattle ID"

            if (Test-Path -Path "$($env:CATTLE_AGENT_CONFIG_DIR)/cattle-id") {
                $env:CATTLE_ID = Get-Content -Path "$($env:CATTLE_AGENT_CONFIG_DIR)/cattle-id"
                Write-LogInfo "Cattle ID was already detected as $($env:CATTLE_ID). Not generating a new one."
                return
            }
            $stream = [IO.MemoryStream]::new([Text.Encoding]::UTF8.GetBytes($env:COMPUTERNAME))
            $env:CATTLE_ID = (Get-FileHash -InputStream $stream -Algorithm SHA256).Hash.ToLower().Substring(0, 62)
            Set-Content -Path "$($env:CATTLE_AGENT_CONFIG_DIR)/cattle-id" -Value $env:CATTLE_ID
            return
        }
        Write-LogInfo "Not generating Cattle ID"
    }

    function Get-Address() {
        [CmdletBinding()]
        param (
            [Parameter()]
            [String]
            $Value
        )
        if (!$Value) {
            # If nothing is given, return empty (it will be automatically determined later if empty)
            return ""
        }
        # If given address is a network interface on the system, retrieve configured IP on that interface (only the first configured IP is taken)
        elseif (Get-NetAdapter -Name $Value -ErrorAction SilentlyContinue) {
            return $(Get-NetIpConfiguration | Where-Object { $null -ne $_.IPv4DefaultGateway -and $_.NetAdapter.Status -ne "Disconnected" }).IPv4Address.IPAddress
        }
        # Loop through cloud provider options to get IP from metadata, if not found return given value
        else {
            switch ($Value) {
                awslocal { return curl.exe --connect-timeout 60 --max-time 60 -s http://169.254.169.254/latest/meta-data/local-ipv4 }
                awspublic { return curl.exe --connect-timeout 60 --max-time 60 -s http://169.254.169.254/latest/meta-data/public-ipv4 }
                doprivate { return curl.exe --connect-timeout 60 --max-time 60 -s http://169.254.169.254/metadata/v1/interfaces/private/0/ipv4/address }
                dopublic { return curl.exe --connect-timeout 60 --max-time 60 -s http://169.254.169.254/metadata/v1/interfaces/public/0/ipv4/address }
                azprivate { return curl.exe --connect-timeout 60 --max-time 60 -s -H Metadata:true "http://169.254.169.254/metadata/instance/network/interface/0/ipv4/ipAddress/0/privateIpAddress?api-version=2017-08-01&format=text" }
                azpublic { return curl.exe --connect-timeout 60 --max-time 60 -s -H Metadata:true "http://169.254.169.254/metadata/instance/network/interface/0/ipv4/ipAddress/0/publicIpAddress?api-version=2017-08-01&format=text" }
                gceinternal { return curl.exe --connect-timeout 60 --max-time 60 -H "Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/instance/network-interfaces/0/ip }
                gceexternal { return curl.exe --connect-timeout 60 --max-time 60 -H "Metadata-Flavor: Google" http://metadata.google.internal/computeMetadata/v1/instance/network-interfaces/0/access-configs/0/external-ip }
                packetlocal { return curl.exe --connect-timeout 60 --max-time 60 -s https://metadata.packet.net/2009-04-04/meta-data/local-ipv4 }
                packetpublic { return curl.exe --connect-timeout 60 --max-time 60 -s https://metadata.packet.net/2009-04-04/meta-data/public-ipv4 }
                ipify { return curl.exe --connect-timeout 60 --max-time 60 -s https://api.ipify.org }
                default {
                    return $Value
                }
            }          
        }
    }

    function Copy-WinsForCharts() {
        $winsForChartsPath = "c:/windows"
        if (-Not (Test-Path $winsForChartsPath)) {
            New-Item $winsForChartsPath -ItemType Directory
        }
        Copy-Item -Path "$env:CATTLE_AGENT_BIN_PREFIX/bin/wins.exe" -Destination "$winsForChartsPath/wins.exe" -Force
    }

    function Confirm-WindowsFeatures {
        [CmdletBinding()]
        param (
            [Parameter(Mandatory = $true)]
            [String[]]
            $RequiredFeatures
        )
        foreach ($feature in $RequiredFeatures) {
            $f = Get-WindowsFeature -Name $feature
            if (-not $f.Installed) {
                Write-LogFatal "Windows feature: '$feature' is not installed. Please run: Install-WindowsFeature -Name $feature"
            }
            else {
                Write-LogInfo "Windows feature: '$feature' is installed. Installation will proceed."
            }
        }
    }

    function Set-RestrictedPermissions {
        [CmdletBinding()]
        param (
            [Parameter(Mandatory=$true)]
            [string]
            $Path,
            [Parameter(Mandatory=$false)]
            [Switch]
            $Directory
        )

        $Owner = "BUILTIN\Administrators"
        $Group = "NT AUTHORITY\SYSTEM"

        $acl = Get-Acl $Path

        # cleanup existing rules by removing both explicit and inherited rules.
        foreach ($rule in $acl.GetAccessRules($true, $true, [System.Security.Principal.SecurityIdentifier])) {
            $acl.RemoveAccessRule($rule) | Out-Null
        }

        $acl.SetAccessRuleProtection($true, $false)
        $acl.SetOwner((New-Object System.Security.Principal.NTAccount($Owner)))
        $acl.SetGroup((New-Object System.Security.Principal.NTAccount($Group)))

        Set-FileSystemAccessRule -Directory $Directory -acl $acl

        Set-Acl -Path $Path -AclObject $acl
    }

    function Set-FileSystemAccessRule() {
        [CmdletBinding()]
        param (
            [Parameter(Mandatory=$true)]
            [Boolean]
            $Directory,
            [Parameter(Mandatory=$false)]
            [System.Security.AccessControl.ObjectSecurity]
            $acl
        )
        $users = @(
            $acl.Owner,
            $acl.Group
        )
        # Note that the function signature for files and directories
        # intentionally differ.
        $FullPath = Resolve-Path $Path
        if ($Directory -eq $true) {
            Write-LogInfo "Setting restricted ACL on $FullPath directory"
            foreach ($user in $users) {
                $rule = New-Object System.Security.AccessControl.FileSystemAccessRule(
                $user,
                [System.Security.AccessControl.FileSystemRights]::FullControl,
                [System.Security.AccessControl.InheritanceFlags]'ObjectInherit,ContainerInherit',
                [System.Security.AccessControl.PropagationFlags]::None,
                [System.Security.AccessControl.AccessControlType]::Allow
                )
                $acl.AddAccessRule($rule)
            }
        } else {
            Write-LogInfo "Setting restricted ACL on $FullPath"
            foreach ($user in $users) {
                $rule = New-Object System.Security.AccessControl.FileSystemAccessRule(
                $user,
                [System.Security.AccessControl.FileSystemRights]::FullControl,
                [System.Security.AccessControl.AccessControlType]::Allow
                )
                $acl.AddAccessRule($rule)
            }
        }
    }

    function Invoke-WinsAgentInstall() {
        $serviceName = "rancher-wins"
        Get-Args
        Set-Environment
        Set-RestrictedPermissions -Path $env:CATTLE_AGENT_CONFIG_DIR -Directory
        Set-RestrictedPermissions -Path $env:CATTLE_AGENT_VAR_DIR -Directory
        Set-Path
        Test-CaCheckSum

        if ($env:CATTLE_CA_CHECKSUM) {
            Test-CaRequired
        }

        Test-RancherConnection
        Stop-Agent -ServiceName $serviceName
        Invoke-WinsAgentDownload
        Copy-WinsForCharts
        Set-WinsConfig

        if($env:CSI_PROXY_URL -and $env:CSI_PROXY_VERSION -and $env:CSI_PROXY_KUBELET_PATH) {
            Set-CsiProxyConfig
        }

        if ($env:CATTLE_TOKEN) {
            New-CattleId
            Get-RancherConnectionInfo
        }                   

        $newEnv = @()
        $PROXY_ENV_INFO = Get-ChildItem env: | Where-Object { $_.Name -Match "^(NO|HTTP|HTTPS)_PROXY" } | ForEach-Object { "$($_.Name)=$($_.Value)" }
        if ($PROXY_ENV_INFO) {
            netsh winhttp set proxy $env:HTTPS_PROXY
            Set-ItemProperty -path "HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings" ProxyEnable -value 1
            Set-ItemProperty -path "HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings" ProxyServer -value "https=$env:HTTPS_PROXY;http=$env:HTTP_PROXY"
            Set-ItemProperty -path "HKCU:\Software\Microsoft\Windows\CurrentVersion\Internet Settings" ProxyOverride -value $env:NO_PROXY.Replace(',',';')

            $newEnv += $PROXY_ENV_INFO
            if(Test-Path -Path HKLM:SYSTEM\CurrentControlSet\Services\$serviceName) {
                Set-ItemProperty HKLM:SYSTEM\CurrentControlSet\Services\$serviceName -Name Environment -Value $newEnv
            }
            else {
                New-Item HKLM:SYSTEM\CurrentControlSet\Services\$serviceName
                New-ItemProperty HKLM:SYSTEM\CurrentControlSet\Services\$serviceName -Name Environment -PropertyType MultiString -Value $newEnv
            }
        }
                
        try {
            Write-LogInfo "Checking if $serviceName service exists."
            Get-Service -Name $serviceName
        }
        catch {
            Write-LogInfo "$serviceName service not found, enabling agent service."
            Push-Location c:\usr\local\bin
            wins.exe srv app run --register
            Pop-Location
            Start-Sleep -s 5
        }

        try
        {
            Write-LogInfo "Starting $serviceName service."
            Start-Service -Name $serviceName
        } catch {
            Write-LogInfo "$serviceName failed to start. Check the $serviceName logs for more information"
            Write-LogInfo "Command: Get-WinEvent -ProviderName $serviceName | select-object TimeCreated,Message | Format-Table -wrap"
            exit 1
        }
        while ((Get-Service $serviceName).Status -ne 'Running') {
            Write-LogInfo "Waiting for $serviceName service to start."
            Start-Sleep -s 5
        }
    }

    Confirm-WindowsFeatures -RequiredFeatures @("Containers")
    Invoke-WinsAgentInstall
}
