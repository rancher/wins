<#
.SYNOPSIS
    Builds the rancher-wins docker image
.DESCRIPTION
    Runs the docker build command for a given windows OS version, docker repository, and tag
.NOTES
    Parameters:
      - ServerCoreVersion: Defines the windows OS version used to build the image. Can be either '1809' or 'ltsc2022'
      - Repo: Dockerhub Repo
      - Tag: Docker Tag

.EXAMPLE
    build-image -ServerCoreVersion "1809" -Repo "rancher" -Tag "v1.2.3"
#>

param (
    [Parameter()]
    [String]
    $ServerCoreVersion,

    [Parameter()]
    [String]
    $Repo,

    [Parameter()]
    [String]
    $Tag
)

if (($ServerCoreVersion -eq "") -or
        (($ServerCoreVersion -ne "ltsc2022") -and ($ServerCoreVersion -ne "1809"))) {
    Write-Host "-ServerCoreVersion must be provided. Accepted values are '1809' and 'ltsc2022'"
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
docker build -f Dockerfile --build-arg SERVERCORE_VERSION=$ServerCoreVersion --build-arg ARCH=amd64 --build-arg MAINTAINERS="harrison.affel@suse.com arvind.iyengar@suse.com" --build-arg REPO=https://github.com/rancher/wins -t $Repo/wins:$Tag-windows-$ServerCoreVersion .
