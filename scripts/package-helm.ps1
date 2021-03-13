#Requires -Version 5.0
$ErrorActionPreference = "Stop"

$SRC_PATH = (Resolve-Path "$PSScriptRoot\..").Path
Push-Location $SRC_PATH

Remove-Item -Path "$SRC_PATH\dist\*" -Force -ErrorAction Ignore
$null = New-Item -Type Directory -Path dist -ErrorAction Ignore

c:\windows-amd64\helm.exe package -d dist charts\rancher-wins-upgrader

Pop-Location
