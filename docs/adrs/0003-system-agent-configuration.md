# 3. System Agent configuration

Date: 2021-10-04

## Status

Accepted

## Context

The system agent requires some configuration. In addition, wins need the ability to enable or disable the system agent functionality to enable backwards compatibility.

## Decision

Two decisions were made. The system agent functionality was added to the wins configuration file as a section called *sa*. The second decision is that if that section doesn't exist, then system agent functionality will be disabled. If, it does exist, then the system agent functionality will be enabled.

## Consequences

This allows maintaining consistent behavior for existing users. Users will not need any additional command line options to alter the behavior. This does require that users of system agent know that the configuration is required. The primary user currently will be Rancher and therefore, will set the configuration correctly during install.