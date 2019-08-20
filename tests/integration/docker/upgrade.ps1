$ErrorActionPreference = "Stop"

# copy to host
Copy-Item -Recurse -Force -Path c:\Windows\wins.exe -Destination c:\etc\rancher\wins\ | Out-Null

# query info
Start-Sleep -s 10
$count = 60
while ($count -gt 0) {
    $ret = wins.exe cli app info
    if ($?) {
        [System.Console]::Out.Write($ret)
        exit 0
    }
    Start-Sleep -s 1
    $count -= 1
}
if ($count -le 0) {
    [System.Console]::Error.Write("Timeout")
    exit 1
}
