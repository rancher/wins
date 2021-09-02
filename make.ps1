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