ARG SERVERCORE_VERSION
FROM library/golang:1.17.7 as base
SHELL ["powershell", "-NoLogo", "-Command", "$ErrorActionPreference = 'Stop'; $ProgressPreference = 'SilentlyContinue';"]

RUN pushd c:\; \
    $URL = 'https://github.com/StefanScherer/docker-cli-builder/releases/download/20.10.5/docker.exe'; \
    \
    Write-Host ('Downloading docker from {0} ...' -f $URL); \
    curl.exe -sfL $URL -o c:\Windows\docker.exe; \
    \
    Write-Host 'Complete.'; \
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
    Write-Host 'Complete.'; \
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
    \
    Write-Host 'Complete.'; \
    popd;

# install ginkgo
RUN pushd c:\; \
    \
    Write-Host ('Updating ginkgo ...'); \
    go install github.com/onsi/ginkgo/ginkgo@latest; \
    go get -u github.com/onsi/gomega/...; \
    \
    Write-Host 'Ginkgo install complete.'; \
    popd;
#
#ENTRYPOINT ["powershell", "-NoLogo", "-NonInteractive", "-File", "./scripts/entry.ps1"]
#CMD ["ci"]

COPY . /go/wins/
WORKDIR C:/

RUN Write-Host "current directory is $(pwd)"; \
    Set-Location C:/go/wins/ ; \
    ./scripts/entry.ps1 "ci"

#COPY ./install.ps1 C:/package/install.ps1
#COPY ./suc/run.ps1 C:/package/run.ps1

ENV SERVERCORE_VERSION = ${SERVERCORE_VERSION}
FROM mcr.microsoft.com/windows/servercore:${SERVERCORE_VERSION} as wins
ENV ARCH=amd64

SHELL ["powershell", "-NoLogo", "-Command", "$ErrorActionPreference = 'Stop'; $ProgressPreference = 'SilentlyContinue';"]
COPY --from=base C:/package/. C:/.
WORKDIR C:/

# Create a symbolic link pwsh.exe that points to powershell.exe for consistency
RUN New-Item -ItemType SymbolicLink -Target "C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe" -Path "C:\Windows\System32\WindowsPowerShell\v1.0\pwsh.exe"

COPY ./wins.exe C:/Windows/wins.exe
#COPY ./install.ps1 install.ps1
#COPY ./suc/run.ps1 run.ps1
#USER ContainerAdministrator

ENTRYPOINT [ "powershell", "-Command", "./run.ps1"]



