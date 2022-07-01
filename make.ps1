if (!(Test-Path .dapper.exe)) {
    $dapperURL = "https://releases.rancher.com/dapper/latest/dapper-Windows-x86_64.exe"
    Write-Host "no .dapper.exe, downloading $dapperURL"
    curl.exe -sfL -o .dapper.exe $dapperURL
}

if ($args.Count -eq 0) {
    $args = @("ci")
}

if ($args[0] -eq "integration") {
    Write-Host "Running Integration Tests"
    .dapper.exe -f Dockerfile.dapper build
    scripts\integration.ps1
    exit
}

if ($args[0] -eq "build") {
    Write-Host "Building wins"
    Import-Module -WarningAction Ignore -Name "$PSScriptRoot\scripts\utils.psm1"
    Invoke-Expression -Command "$PSScriptRoot\scripts\version.ps1"

    $IMAGE = ('{0}/wins:{1}-windows-{2}-{3}' -f $env:REPO, $env:TAG, $env:SERVERCORE_VERSION, $env:ARCH)
    Write-Host -ForegroundColor Yellow "Starting docker build of $IMAGE`n"

    docker build `
    --build-arg SERVERCORE_VERSION=$env:SERVERCORE_VERSION `
    --build-arg ARCH=$env:ARCH `
    --build-arg VERSION=$env:TAG `
    -t $IMAGE `
    -f Dockerfile .

    if ($LASTEXITCODE -ne 0) {
        exit $LASTEXITCODE
    }
    Write-Host -ForegroundColor Green "Successfully built $IMAGE`n"
    exit
}

if ($args[0] -eq "package") {
    Write-Host "Building and Packaging wins"
    .dapper.exe -f Dockerfile.dapper build
    .dapper.exe -f Dockerfile.dapper package
    exit
}

if ($args[0] -eq "all") {
    Write-Host "Running CI and Integration Tests"
    .dapper.exe -f Dockerfile.dapper ci
    scripts\integration.ps1
    exit
}

if ($args[0] -eq "clean") {
    Remove-Item .dapper.exe
    Remove-Item Dockerfile.dapper* -Exclude "Dockerfile.dapper"
}

if (Test-Path scripts\$($args[0]).ps1) {
    .dapper.exe -f Dockerfile.dapper $($args[0])
    exit
}