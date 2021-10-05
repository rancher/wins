# 2. Add System Agent support to Wins

Date: 2021-10-04

## Status

Accepted

## Context

Currently, RKE2 uses [`rancher/system-agent`](https://github.com/rancher/system-agent) to install and upgrade RKE2 using plans. However, this capability doesn't currently exist for Windows hosts. System Agent functionality needs some capabilities of Wins to achieve its goal. This would require that both the system agent service and wins service to be running on a system. Alternatively, we could consume the system agent as a package in the wins service reducing the need to having both installed in a system.

## Decision

The decision was to consume the system agent functionality as a package in wins.

## Consequences

This decision reduces the overall services running on a system and allows leveraging the existing wins upgrader and capabilities to provide the system agent functionality.
