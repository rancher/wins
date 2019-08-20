@echo off
if "%1%"=="integration" (
    call :integration
    goto :eof
)
if "%1%"=="all" (
    call :all
    goto :eof
)
if "%1%"=="" (
    set cmd=ci
) else (
    set cmd=%1%
)
call :.dapper
.dapper.exe -f Dockerfile.dapper %cmd%
goto :eof

:.dapper
if not exist .dapper.exe (
    bitsadmin.exe /rawreturn /transfer dappwer-download https://releases.rancher.com/dapper/latest/dapper-Windows-x86_64.exe %~dp0\.dapper.exe
    .dapper.exe -v
)
goto :eof

:integration
call :.dapper
.dapper.exe -f Dockerfile.dapper build
powershell.exe -NoLogo -NonInteractive -File %~dp0\scripts\integration.ps1
goto :eof

:all
call :.dapper
.dapper.exe -f Dockerfile.dapper ci
powershell.exe -NoLogo -NonInteractive -File %~dp0\scripts\integration.ps1
goto :eof