#Requires -Version 5.0
$ErrorActionPreference = 'Stop'

trap {
    Write-Host -NoNewline -ForegroundColor Red "[ERROR]: "
    Write-Host -ForegroundColor Red "$_"

    popd
    exit 1
}

$SCRIPT_PATH = ("{0}\{1}.ps1" -f $PSScriptRoot, $Args[0])
if (Test-Path $SCRIPT_PATH -ErrorAction Ignore)
{
    Invoke-Expression -Command $SCRIPT_PATH
    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }
    exit 0
}

Start-Process -Wait -FilePath $Args[0] -ArgumentList $Args[1..$Args.Length]
