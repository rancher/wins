$ErrorActionPreference = "Stop"

# copy to host
Copy-Item -Recurse -Force -Path c:\etc\nginx\* -Destination c:\host\etc\nginx\ | Out-Null

# start process
wins.exe cli prc run --path c:\etc\nginx\nginx.exe --exposes TCP:80 --args=`-g env hello=world`
