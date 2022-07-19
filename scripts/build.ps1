#Requires -Version 5.0

$ErrorActionPreference = 'Stop'

Import-Module -WarningAction Ignore -Name "$PSScriptRoot\utils.psm1"

function Build {
    param (
        [parameter(Mandatory = $true)] [string]$Version,
        [parameter(Mandatory = $true)] [string]$Commit,
        [parameter(Mandatory = $true)] [string]$Output
    )

    $linkFlags = ('-s -w -X github.com/rancher/wins/pkg/defaults.AppVersion={0} -X github.com/rancher/wins/pkg/defaults.AppCommit={1} -extldflags "-static"' -f $Version, $Commit)
    go build -ldflags $linkFlags -o $Output cmd/main.go
    if (-not $?) {
        Log-Fatal "[build] go build failed!"
    }
}

Invoke-Script -File "$PSScriptRoot\version.ps1"

$SRC_PATH = (Resolve-Path "$PSScriptRoot\..").Path
Push-Location $SRC_PATH

Remove-Item -Path "$SRC_PATH\bin\*" -Force -ErrorAction Ignore
$null = New-Item -Type Directory -Path bin -ErrorAction Ignore
$env:GOARCH = $env:ARCH
$env:GOOS = 'windows'
$env:CGO_ENABLED = 0
Write-Host "[build] Building wins version ($env:VERSION) for $env:GOOS/$env:GOARCH"
Build -Version $env:VERSION -Commit $env:COMMIT -Output "$SRC_PATH\bin\wins.exe"
Write-Host "[build] successfully built wins version $env:VERSION for $env:GOOS/$env:GOARCH"

$PACKAGE_DIR = "C:\package"
$ASSETS = @("install.ps1", "bin\wins.exe", "suc\run.ps1")
Write-Host -ForegroundColor Yellow "[build] now staging build artifacts [$ASSETS] from $SRC_PATH"
$null = New-Item -Type Directory -Path C:\package -ErrorAction Ignore
foreach ($item in $ASSETS) {
    if ("$SRC_PATH\$item") {
        if ($item.Contains('\')) {
            $i = $item -replace "^.*?\\"
            Copy-Item -Force -Path $SRC_PATH\$item -Destination "$PACKAGE_DIR\$i"
            Write-Host ('[build] new artifact ({0}\{1})' -f $PACKAGE_DIR, $i)
        } else {
            Copy-Item -Force -Path $SRC_PATH\$item -Destination "$PACKAGE_DIR\$item"
            Write-Host ('[build] new artifact ({0}\{1})' -f $PACKAGE_DIR, $item)
        }
    } else {
        Log-Fatal "[build] build artifact $SRC_PATH\$item is missing"
    }
}
Write-Host -ForegroundColor Green "[build] all required build artifacts have been staged"
Write-Host -ForegroundColor Blue ('[build] Artifacts List in ({0}): {1}' -f $PACKAGE_DIR, (Get-ChildItem $PACKAGE_DIR))
Write-Host -ForegroundColor Green "build.ps1 has completed successfully"
Write-Host "[build] invoking package.ps1 to validate build artifacts"
Invoke-Script -File "$PSScriptRoot\package.ps1"
