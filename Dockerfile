ARG NANOSERVER_VERSION
FROM mcr.microsoft.com/windows/nanoserver:${NANOSERVER_VERSION} as wins
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

COPY ./artifacts/wins-suc.exe C:/
COPY ./suc/update-connection-info.ps1 C:/

USER ContainerAdministrator
ENTRYPOINT [ "wins-suc.exe" ]
