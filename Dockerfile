ARG SERVERCORE_VERSION
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
WORKDIR C:/

COPY ./artifacts C:/
# staging for backwards compatibility
# Create a symbolic link pwsh.exe that points to powershell.exe for consistency
RUN New-Item -ItemType SymbolicLink -Target "C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe" -Path "C:\Windows\System32\WindowsPowerShell\v1.0\pwsh.exe" ; \
    Copy-Item wins.exe -Destination C:/Windows

USER ContainerAdministrator
ENTRYPOINT [ "powershell", "-Command", "./run.ps1"]



