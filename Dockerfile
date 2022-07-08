ARG SERVERCORE_VERSION
FROM library/golang:1.17.7 as base
SHELL ["powershell", "-NoLogo", "-Command", "$ErrorActionPreference = 'Stop'; $ProgressPreference = 'SilentlyContinue';"]

RUN pushd c:\; \
    $URL = 'https://github.com/StefanScherer/docker-cli-builder/releases/download/20.10.5/docker.exe'; \
    \
    Write-Host ('Downloading docker from {0} ...' -f $URL); \
    curl.exe -sfL $URL -o c:\Windows\docker.exe; \
    \
    Write-Host 'docker install complete.'; \
    popd;

RUN pushd c:\; \
    $URL = 'https://github.com/golangci/golangci-lint/releases/download/v1.44.0/golangci-lint-1.44.0-windows-amd64.zip'; \
    \
    Write-Host ('Downloading golangci from {0} ...' -f $URL); \
    curl.exe -sfL $URL -o c:\golangci-lint.zip; \
    \
    Write-Host 'Expanding ...'; \
    Expand-Archive -Path c:\golangci-lint.zip -DestinationPath c:\; \
    \
    Write-Host 'Cleaning ...'; \
    Remove-Item -Force -Recurse -Path c:\golangci-lint.zip; \
    \
    Write-Host 'Updating PATH ...'; \
    [Environment]::SetEnvironmentVariable('PATH', ('c:\golangci-lint-1.44.0-windows-amd64\;{0}' -f $env:PATH), [EnvironmentVariableTarget]::Machine); \
    \
    Write-Host 'golangci-lint install complete.'; \
    popd;

# upgrade git
RUN pushd c:\; \
    $URL = 'https://github.com/git-for-windows/git/releases/download/v2.33.0.windows.2/MinGit-2.33.0.2-64-bit.zip'; \
    \
    Write-Host ('Downloading git from {0} ...' -f $URL); \
    [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12; \
    Invoke-WebRequest -UseBasicParsing -OutFile c:\git.zip -Uri $URL; \
    \
    Write-Host 'Expanding ...'; \
    Expand-Archive -Force -Path c:\git.zip -DestinationPath c:\git\.; \
    \
    Write-Host 'Cleaning ...'; \
    Remove-Item -Force -Recurse -Path c:\git.zip; \
    Write-Host 'git install complete.'; \
    popd;

# install ginkgo
RUN pushd c:\; \
    \
    Write-Host ('Updating ginkgo ...'); \
    go install github.com/onsi/ginkgo/ginkgo@latest; \
    go get -u github.com/onsi/gomega/...; \
    Write-Host 'Ginkgo install complete.'; \
    popd;

COPY . /go/wins/
WORKDIR C:/

ARG ACTION
ENV ACTION ${ACTION}
RUN Write-Host "Starting CI Action ($env:ACTION) for wins"; \
    Set-Location C:/go/wins/ ; \
    ./scripts/ci.ps1 "$env:ACTION"

FROM mcr.microsoft.com/windows/servercore:${SERVERCORE_VERSION} as wins
ARG VERSION
ARG MAINTAINERS
ARG REPO

SHELL ["powershell", "-NoLogo", "-Command", "$ErrorActionPreference = 'Stop'; $ProgressPreference = 'SilentlyContinue';"]

ENV VERSION ${VERSION}
ENV MAINTAINERS ${MAINTAINERS}
ENV REPO ${REPO}

LABEL org.opencontainers.image.authors=${MAINTAINERS}
LABEL org.opencontainers.image.url=${REPO}
LABEL org.opencontainers.image.documentation=${REPO}
LABEL org.opencontainers.image.source=${REPO}
LABEL org.label-schema.vcs-url=${REPO}
LABEL org.opencontainers.image.vendor="Rancher Labs"
LABEL org.opencontainers.image.version=${VERSION}

COPY --from=base C:/package/. C:/.
WORKDIR C:/


# Create a symbolic link pwsh.exe that points to powershell.exe for consistency
RUN New-Item -ItemType SymbolicLink -Target "C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe" -Path "C:\Windows\System32\WindowsPowerShell\v1.0\pwsh.exe" ; \
    Copy-Item -Path ./wins.exe -Destination ./Windows/
#USER ContainerAdministrator

ENTRYPOINT [ "powershell", "-Command", "./run.ps1"]



