# Rancher Wins

[![Build Status](https://drone-publish.rancher.io/api/badges/rancher/wins/status.svg?ref=refs/heads/main)](https://drone-pr.rancher.io/rancher/wins)
[![Go Report Card](https://goreportcard.com/badge/github.com/rancher/wins)](https://goreportcard.com/report/github.com/rancher/wins)

`wins` embeds Rancher system-agent and manages CSI Proxy service lifecycle on Windows nodes.

## Release Lines

+ `release/v0.4.x`, `v0.4.x` 
  + Currently in maintenance mode, intended for use by Rancher versions <= 2.9.x
+ `main`, `v0.5.x`
  + Currently accepting new features, intended for use by Rancher versions >= v2.10.x

While Rancher versions <= v2.9.x are supported, CVEs must be addressed in both rancher-wins release lines and bumped in the relevant Rancher branches.   

## How to use

Wins now focuses on two runtime capabilities:

- Embedding Rancher system-agent plan execution on Windows.
- Enabling and maintaining the CSI Proxy Windows service when configured.

### Modules

```
> wins.exe -h
NAME:
   rancher-wins - Embedded system-agent and CSI proxy manager for Windows nodes

USAGE:
   wins.exe [global options] command [command options] [arguments...]

VERSION:
   ...

DESCRIPTION:
   Rancher Wins component for embedded system-agent and CSI proxy service management

COMMANDS:
   srv, server
   stackdump
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

### Examples

``` powershell
# [host] start the wins server
> wins.exe --debug srv app run
```

### Developer Documentation
```powershell
# [host] build local wins and run it as a service for testing/debugging
git clone https://github.com/rancher/wins.git
Set-Location wins
Set-Location $(Get-Location).path

# build the project using the magefile
go run mage.go build

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

#### Enabling System Agent functionality

System-agent functionality is enabled only when the `systemagent` configuration section is present.
To enable it, provide the following configuration section with the required settings. If *remoteEnabled* is
set to `true`, then `connectionInfoFile` must be configured.

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

The [CSI Proxy](https://github.com/kubernetes-csi/csi-proxy) is enabled only when the `csi-proxy` configuration section is present.
To enable it, provide the following configuration section with the required settings. The `url`
setting is expected to be formatted for Go `sprintf`. An example is provided below.
Once enabled, Wins downloads CSI Proxy, creates the Windows service, and ensures it is running.

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

> Note: Don't use `bin/wins.exe` after integration testing. Please `go run mage.go build` again.

If want both of them, please run the below command in `PowerShell`:

``` powershell
> go run mage.go TestAll
```

## License

Copyright (c) 2014-2023 [Rancher Labs, Inc.](http://rancher.com)

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the
License. You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "
AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific
language governing permissions and limitations under the License.