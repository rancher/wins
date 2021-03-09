ARG SERVERCORE_VERSION

FROM mcr.microsoft.com/powershell:nanoserver-${SERVERCORE_VERSION}

COPY ./bin/wins.exe /Windows/
COPY entry.ps1 entry.ps1
USER ContainerAdministrator

CMD ["pwsh", "-f", "entry.ps1"]

