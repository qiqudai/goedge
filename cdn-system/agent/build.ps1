$ErrorActionPreference = "Stop"

$scriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
Push-Location $scriptDir

try {
Write-Host "Building cdn-agent..."
$mainFile = "E:/cdn/goedge/cdn-system/agent/main.go"
$mainContent = Get-Content -Raw $mainFile
$versionPattern = 'var\s+Version\s*=\s*"([^"]+)"'
$currentVersion = ""
if ($mainContent -match $versionPattern) {
    $currentVersion = $Matches[1]
}

$nextVersion = "1.0.0"
if ($currentVersion -match '^(\d+)\.(\d+)\.(\d+)$') {
    $major = [int]$Matches[1]
    $minor = [int]$Matches[2]
    $patch = [int]$Matches[3] + 1
    $nextVersion = "$major.$minor.$patch"
}

if ($mainContent -match $versionPattern) {
    $mainContent = [regex]::Replace($mainContent, $versionPattern, "var Version = `"$nextVersion`"")
} else {
    $mainContent = $mainContent + "`r`n`r`nvar Version = `"$nextVersion`"`r`n"
}
Set-Content -Path $mainFile -Value $mainContent -Encoding UTF8

$buildVersion = $nextVersion
$wslCmd = "cd ""/mnt/e/cdn/goedge/cdn-system/agent"" && GO111MODULE=on go build -ldflags ""-X=main.Version=$buildVersion"" -o ""/mnt/e/cdn/goedge/cdn-system/agent/cdn-agent"" ."
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
