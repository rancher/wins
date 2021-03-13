#Requires -Version 5.0
$ErrorActionPreference = "Stop"

c:\windows-amd64\helm.exe package -d bin charts\rancher-wins-upgrader
