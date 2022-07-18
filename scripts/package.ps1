#Requires -Version 5.0
$ErrorActionPreference = "Stop"

Invoke-Expression -Command "$PSScriptRoot\version.ps1"

$DIR_PATH = Split-Path -Parent $MyInvocation.MyCommand.Definition
$SRC_PATH = (Resolve-Path "$DIR_PATH\..").Path
$PACKAGE_DIR = "C:\package"

$ASSETS = @("install.ps1", "wins.exe", "run.ps1")
Write-Host -ForegroundColor Yellow "[package] verifying build artifacts [$ASSETS] are present in ($PACKAGE_DIR)"
foreach ($item in $ASSETS) {
    if (-not("$PACKAGE_DIR\$item")) {
        Log-Fatal "[package] required build artifact $PACKAGE_DIR\$item is missing, exiting now"
    }
}
Write-Host -ForegroundColor Green "[package] all required build artifacts are present"
Write-Host -ForegroundColor Green "package.ps1 has completed successfully"
