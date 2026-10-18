# Build the Windows x64 binary next to this script.
$ErrorActionPreference = "Stop"

$env:GOPROXY = "https://goproxy.cn,direct"
$env:GOTOOLCHAIN = "local"
$env:CGO_ENABLED = "0"

go build -trimpath -ldflags "-s -w -X main.version=0.3.1-ui" -o mahjong-helper-ui.exe .
Write-Host "Built mahjong-helper-ui.exe"
