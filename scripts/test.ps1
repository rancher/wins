#Requires -Version 5.0

$ErrorActionPreference = 'Stop'

Import-Module -WarningAction Ignore -Name "$PSScriptRoot\utils.psm1"

Invoke-Script -File "$PSScriptRoot\version.ps1"

$SRC_PATH = (Resolve-Path "$PSScriptRoot\..").Path
Push-Location $SRC_PATH

Get-Command -ErrorAction Ignore -Name @("x86_64-w64-mingw32-gcc.exe", "x86_64-w64-mingw32-g++.exe") | Out-Null
if (-not $?) {
    Log-Info 'Installing gcc ...'
    New-Item -Type Directory -Path c:\cygwin64 -Force -ErrorAction Ignore | Out-Null

    $URL = 'https://cygwin.com/setup-x86_64.exe'
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
    Invoke-WebRequest -UseBasicParsing -OutFile c:\cygwin64\setup.exe -Uri $URL

    $PACKAGES = 'mingw64-x86_64-gcc-core,mingw64-x86_64-gcc-g++'
    Start-Process -NoNewWindow -Wait -FilePath 'c:\cygwin64\setup.exe' -ArgumentList ('-q -d -X -s {0} -D -L -R {1} -l {2} -P {3}' -f 'http://cygwin.mirrors.pair.com/', 'C:/cygwin64', $env:TEMP, $PACKAGES)

    Log-Info 'Updating PATH ...'
    [Environment]::SetEnvironmentVariable('PATH', ('c:\cygwin64\bin\;c:\cygwin64\sbin\;{0}' -f $env:PATH), [EnvironmentVariableTarget]::Machine)
    $env:PATH = ('c:\cygwin64\bin\;c:\cygwin64\sbin\;{0}' -f $env:PATH)

    Log-Info 'Complete .'
}

$env:CXX = 'x86_64-w64-mingw32-g++'
$env:CC = 'x86_64-w64-mingw32-gcc'
$env:GOARCH = $env:ARCH
$env:GOOS = 'windows'
$env:CGO_ENABLED = 1
$LINKFLAGS = ('-X github.com/rancher/wins/pkg/defaults.AppVersion={0} -X github.com/rancher/wins/pkg/defaults.AppCommit={1} -linkmode external' -f $env:VERSION, $env:COMMIT)
ginkgo --ldflags $LINKFLAGS --randomizeAllSpecs --randomizeSuites --noisyPendings=false --noisySkippings=false --cover --coverpkg=github.com/rancher/wins/... --trace --race -r tests/validation/
if (-not $?) {
    Log-Fatal "validation test failed"
}

Pop-Location
