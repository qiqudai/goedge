$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Push-Location $scriptDir

try {
    Write-Host "Building cdn-agent..."
    $wslCmd = 'cd "/mnt/e/cdn/goedge/cdn-system/agent" && GO111MODULE=on go build -o "/mnt/e/cdn/goedge/cdn-system/agent/cdn-agent" .'
    & wsl -- bash -lc $wslCmd

    $src = "E:/cdn/goedge/cdn-system/agent/cdn-agent"
    $dstDir = "E:/cdn/goedge/cdn-system/build/linux-amd64"
    $dst = "$dstDir/cdn-agent"

    if (!(Test-Path -Path $src)) {
        throw "Build output not found: $src"
    }

    if (!(Test-Path -Path $dstDir)) {
        New-Item -ItemType Directory -Path $dstDir -Force | Out-Null
    }

    Write-Host "Copying build artifact to $dst"
    Copy-Item -Path $src -Destination $dst -Force

    Write-Host "Done."
} finally {
    Pop-Location
}
