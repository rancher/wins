# 4. CSI Proxy and Configuration

Date: 2021-11-22

## Status

Approved

## Context

To enable storage with Windows nodes, the [CSI Proxy](https://github.com/kubernetes-csi/csi-proxy) was created to allow CSI implementations to run as daemonsets on Windows nodes. Implementing support for installation and configuration of the CSI Proxy within Wins allows multiple projects to benefit from having the ability to have the CSI Proxy available, including both RKE and RKE2.

## Decision

After some discussion, it was decided that Wins would be responsible for downloading the specified version of CSI Proxy from the current upstream location based on the version that is specified. The ability to override the upstream URL will be provided. Once the binary is available Wins will be responsible for creating the Windows Service for CSI Proxy and ensuring that it is running. Wins will also have the ability to manage the lifecycle of the CSI Proxy service since configuration changes will need to be picked up from Wins. The proposed configuration is below:

```yaml
csi-proxy:
  url: <location of the binaries>
  version: <version>
```

The presence of the configuration will enable the CSI Proxy configuration.

## Consequences

This allows maintaining consistent behavior for existing users. Users will not need any additional command line options to alter the behavior. This does require that users of CSI Proxy know that the configuration is required. The primary user currently will be Rancher and therefore, will set the configuration correctly during install.