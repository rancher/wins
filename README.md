# Rancher Wins

[![Build Status](https://drone-publish.rancher.io/api/badges/rancher/wins/status.svg?ref=refs/heads/main)](https://drone-pr.rancher.io/rancher/wins)
[![Go Report Card](https://goreportcard.com/badge/github.com/rancher/wins)](https://goreportcard.com/report/github.com/rancher/wins)

`wins` is a way to operate the Windows host inside the Windows container.

## How to use

### Modules

```
> wins.exe -h
NAME:
   rancher-wins - A way to operate the Windows host inside the Windows container

USAGE:
   wins.exe [global options] command [command options] [arguments...]

VERSION:
   ...

DESCRIPTION:
   Rancher Wins Component (...)

COMMANDS:
   srv, server
   cli, client
   up, upgrade  Manage Rancher Wins Application
   help, h      Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --debug        Turn on verbose debug logging
   --quiet        Turn on off all logging
   --help, -h     show help
   --version, -v  print the version

```

#### Server (run on Windows host)

```
> wins.exe srv -h
NAME:
   rancher-wins srv - The server side commands of Rancher Wins

USAGE:
   rancher-wins srv command [command options] [arguments...]

COMMANDS:
     app, application  Manage Rancher Wins Application

OPTIONS:
   --help, -h  show help
```

#### Client (run inside Windows container)

```
> wins.exe cli -h
NAME:
   rancher-wins cli - The client side commands of Rancher Wins

USAGE:
   rancher-wins cli command [command options] [arguments...]

COMMANDS:
     hns               Manage Host Networking Service
     hst, host         Manage Host
     net, network      Manage Network Adapter
     prc, process      Manage Processes
     route             Manage Routes
     app, application  Manage Rancher Wins Application

OPTIONS:
   --help, -h  show help
```

### Examples

``` powershell
# [host] start the wins server
> wins.exe --debug srv app run

# [host] verify the created npipe
> Get-ChildItem //./pipe/ | Where-Object Name -eq "rancher_wins"
```

### Developer Documentation
```powershell
# [host] build local wins and run it as a service for testing/debugging
git clone https://github.com/rancher/wins.git
Set-Location wins
Set-Location $(Get-Location).path
scripts/build.ps1
copy-item bin/wins.exe .
$WINS_PATH = $(Get-Location).Path

New-Service -Name rancher-wins -BinaryPathName "$WINS_PATH\wins.exe --debug srv app run" -DisplayName "Rancher Wins" -StartupType Manual
Set-Service -Name "rancher-wins" -Status Running -PassThru # start the rancher-wins service and output servicecontroller[]
$(Get-Service -name "rancher-wins").Status # verify that the new rancher-wins service is running

# [host] how to replace the wins.exe binary the service uses during active development
Set-Service -Name "rancher-wins" -Status Stopped
$(Get-Service -name "rancher-wins").Status # verify the service stopped
Set-Service -Name "rancher-wins" -Status Running -PassThru # restart the service with the freshly built wins binary

# change the startup type of rancher-wins from automatic to manual
Set-Service -Name "rancher-wins" -StartupType Manual
```

#### Query the host network adapter

``` powershell
# [host] start a container
> $WINS_BIN_PATH=<...>; docker run --rm -it -v //./pipe/rancher_wins://./pipe/rancher_wins -v "$($WINS_BIN_PATH):c:\host\wins" -w c:\host\wins --entrypoint powershell mcr.microsoft.com/windows/servercore:ltsc2019

# [inside container] query the host network adapter
>> .\wins.exe cli network get
{"InterfaceIndex":"7","GatewayAddress":"10.170.0.1","SubnetCIDR":"10.170.0.0/20","HostName":"frank-wins-dev","AddressCIDR":"10.170.15.229/32"}
```

#### Enabling Process and Port Access

To configure wins properly to break out of a container you need to configure a list of processes and ports which are 
granted permission for wins to use. This is done with the `white_list` configuration options.

```
white_list:
  processPaths:
   - c:\path\to\my.exe
  proxyPorts
   - 8888
```

#### Start a process on the host

``` powershell
# [host] download nginx 
> Invoke-WebRequest -UseBasicParsing -OutFile nginx.zip -Uri http://nginx.org/download/nginx-1.21.3.zip

# [host] expand nginx in the current directory
> Expand-Archive -Force -Path nginx.zip -Destination c:\nginx

# [host] start a container
> $WINS_BIN_PATH=<...>; echo "`$NGINX_BIND_DIR=$NGINX_BIND_DIR"; docker run --rm -it -v //./pipe/rancher_wins://./pipe/rancher_wins -v "$($WINS_BIN_PATH):c:\host\wins" -v "c:\nginx:c:\nginx" -w c:\host\wins --entrypoint powershell mcr.microsoft.com/windows/servercore:ltsc2019

# [inside container] start nginx and receive the running output
>> .\wins.exe cli prc run --path c:\nginx\nginx-1.21.3\nginx.exe --exposes TCP:80

# [host] verify the process
> Get-Process rancher-wins-*
> curl.exe 127.0.0.1
```

#### Enabling System Agent functionality

The system agent functionality will only be enabled if the configuration section for the system agent is found in the
config file. To enable it, provide the following configuration section with the required settings. If *remoteEnabled* is
set to `true` then connectionInfoFile will need to be configured.

```YAML
systemagent:
  appliedPlanDirectory: <agent dir>/applied
  connectionInfoFile: <agent dir>/connection.yaml
  localEnabled: <bool>
  localPlanDirectory: <agent dir>/plans
  preserveWorkDirectory: <bool>
  remoteEnabled: <bool>
  workDirectory: <agent dir>/work
```

#### Enabling CSI Proxy functionality

The [CSI Proxy](https://github.com/kubernetes-csi/csi-proxy) will only be enabled if the configuration section is found
in the config file. To enable it, provide the following configuration section with the required settings. The `url`
setting is expected to be formatted for use in a Go's *sprintf* format. An example is provided below for the formatting.
Once enabled Wins will download the CSI Proxy, create the Windows service, and start the service.

```YAML
csi-proxy:
  url: <url to download the CSI Proxy binary>
  version: <version to download>
  kubeletPath: <path to kubelet>
```

Example:

```YAML
csi-proxy:
  url: https://acs-mirror.azureedge.net/csi-proxy/%[1]s/binaries/csi-proxy-%[1]s.tar.gz
  version: v1.1.1
  kubeletPath: c:/etc/kubelet.exe
```

#### Enabling Certificate Support for Wins

Wins now supports consuming a certificate when it is required for pulling the CSI proxy tarball from a Rancher Server. 
Common situations where this is required are airgapped Rancher environments and self-signed Rancher installations. 

```yml
tls-config:
  insecure: <true/false>
  certFilePath: <path to local certificate>
```

Example:

```yml
tls-config:
  insecure: false
  certFilePath: c:/etc/rancher/agent/ranchercert
```

## Build

This project uses magefile to build. The default target is build.

``` powershell
> go run mage.go <target>
```

## Testing

There are not any Docker-in-Docker supported Windows images for now, `rancher/wins` has to separate the validation test
and integration test.

For validation test, which could be embedded into a containerized CI flow, please run the below command in `PowerShell`:

``` powershell
> go run mage.go validate
```

For integration test, please run the below command in `PowerShell`:

``` powershell
> go run mage.go integration
```

> Note: Don't use `bin/wins.exe` after integration testing. Please `.\make.ps1 build` again.

If want both of them, please run the below command in `PowerShell`:

``` powershell
> .\make.ps1 all
```

## License

Copyright (c) 2014-2022 [Rancher Labs, Inc.](http://rancher.com)

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the
License. You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "
AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific
language governing permissions and limitations under the License.