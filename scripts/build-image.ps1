<#
.SYNOPSIS
    Builds the rancher-wins docker image
.DESCRIPTION
    Runs the docker build command for a given windows OS version, docker repository, and tag
.NOTES
    Parameters:
      - NanoServerVersion: Defines the windows OS version used to build the image. Can be either 'ltsc2019' or 'ltsc2022'
      - Repo: Dockerhub Repo
      - Tag: Docker Tag

.EXAMPLE
    build-image -NanoServerVersion "ltsc2019" -Repo "myRepo" -Tag "v1.2.3"
#>

param (
    [Parameter()]
    [String]
    $NanoServerVersion,

    [Parameter()]
    [String]
    $Repo,

    [Parameter()]
    [String]
    $Tag
)

if (($NanoServerVersion -eq "") -or
        (($NanoServerVersion -ne "ltsc2022") -and ($NanoServerVersion -ne "ltsc2019"))) {
    Write-Host "-NanoServerVersion must be provided. Accepted values are 'ltsc2019' and 'ltsc2022'"
}

if ($Repo -eq "") {
    Write-Host "Repo paramter is empty, defaulting to 'rancher'"
    $Repo = "rancher"
}

if ($Tag -eq "") {
    Write-Host "Tag parameter is empty"
    exit 1
}

# Don't run this command from the scripts directory, always use the parent directory (i.e ./scripts/build-image)
docker build -f Dockerfile --build-arg NANOSERVER_VERSION=$NanoServerVersion --build-arg ARCH=amd64 --build-arg MAINTAINERS="harrison.affel@suse.com" --build-arg REPO=https://github.com/rancher/wins -t $Repo/wins:$Tag-windows-$NanoServerVersion .
