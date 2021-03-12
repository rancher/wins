ARG SERVERCORE_VERSION

FROM mcr.microsoft.com/windows/servercore:${SERVERCORE_VERSION} as download
ENV ARCH=amd64

SHELL ["powershell", "-NoLogo", "-Command", "$ErrorActionPreference = 'Stop'; $ProgressPreference = 'SilentlyContinue';"]

# Create a symbolic link pwsh.exe that points to powershell.exe for consistency
RUN New-Item -ItemType SymbolicLink -Target "C:\Windows\System32\WindowsPowerShell\v1.0\powershell.exe" -Path "C:\Windows\System32\WindowsPowerShell\v1.0\pwsh.exe"

COPY ./wins.exe /Windows/
#USER ContainerAdministrator

CMD ["wins.exe", "--version"]
