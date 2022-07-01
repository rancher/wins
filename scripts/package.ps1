#Requires -Version 5.0
$ErrorActionPreference = "Stop"

Invoke-Expression -Command "$PSScriptRoot\version.ps1"

$DIR_PATH = Split-Path -Parent $MyInvocation.MyCommand.Definition
$SRC_PATH = (Resolve-Path "$DIR_PATH\..").Path

$ASSETS = @("install.ps1", "bin\wins.exe", "suc\run.ps1")
Write-Host -ForegroundColor Yellow "checking for build artifact [$ASSETS] in $SRC_PATH"
foreach ($item in $ASSETS) {
    if (-not ("$SRC_PATH\$item")) {
        Write-Error "required build artifact is missing: $item"
        throw
    }
}
Write-Host -ForegroundColor Green "all required build artifacts are present"

Write-Host -ForegroundColor Yellow "staging artifacts for multi-stage build"
$null = New-Item -Type Directory -Path C:\package -ErrorAction Ignore
$null = New-Item -Type Directory -Path C:\package\suc -ErrorAction Ignore
$null = New-Item -Type Directory -Path C:\package\bin -ErrorAction Ignore
Copy-Item -Force -Path $SRC_PATH\install.ps1 -Destination C:\package\install.ps1
Copy-Item -Force -Path $SRC_PATH\suc\run.ps1 -Destination C:\package\suc\run.ps1
Copy-Item -Force -Path $SRC_PATH\bin\wins.exe -Destination C:\package\bin\wins.exe
Write-Host -ForegroundColor Green "artifacts have been successfully staged"
#Set-Location -Path $SRC_PATH\package
Write-Host -ForegroundColor Green "package.ps1 has completed successfully."
