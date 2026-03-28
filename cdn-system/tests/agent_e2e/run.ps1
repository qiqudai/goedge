param(
    [string]$ApiBase = "http://127.0.0.1:8080",
    [string]$AdminUser = "admin",
    [string]$AdminPass = "123456",
    [string]$AgentToken = "",
    [string]$NodeId = "",
    [switch]$StartAgent,
    [string]$WslDistro = "Ubuntu-24.04",
    [string]$AgentBin = "/usr/local/goedge/nodes/agent-linux",
    [string]$AgentConfig = "/usr/local/goedge/nodes/agent.json",
    [int]$AgentWaitSeconds = 3,
    [string]$WorkDir = "/usr/local/goedge/nodes/edge-node",
    [string]$NginxBin = "/usr/local/goedge/nodes/edge-node/openresty/nginx/sbin/nginx",
    [bool]$VerifyPersistence = $true,
    [bool]$VerifyNginx = $true,
    [bool]$CheckAgentLog = $true
)

$ErrorActionPreference = "Stop"

if ([string]::IsNullOrWhiteSpace($AgentToken) -or [string]::IsNullOrWhiteSpace($NodeId)) {
    Write-Host "AgentToken and NodeId are required."
    exit 1
}

if ($StartAgent) {
    $cmd = "nohup $AgentBin -config $AgentConfig >/tmp/agent_e2e.out 2>&1 &"
    wsl -d $WslDistro -u root -- bash -lc $cmd | Out-Null
    Start-Sleep -Seconds $AgentWaitSeconds
}

function Invoke-Wsl {
    param([string]$Command)
    wsl -d $WslDistro -u root -- bash -lc $Command
    if ($LASTEXITCODE -ne 0) {
        throw "WSL command failed ($LASTEXITCODE): $Command"
    }
}

function Test-WslFile {
    param(
        [string]$Path,
        [bool]$Required = $true
    )
    $cmd = "test -s '$Path'"
    wsl -d $WslDistro -u root -- bash -lc $cmd | Out-Null
    if ($LASTEXITCODE -ne 0) {
        if ($Required) {
            throw "Missing or empty file: $Path"
        }
        Write-Warning "Missing or empty optional file: $Path"
    }
}

$here = Split-Path -Parent $MyInvocation.MyCommand.Path
Push-Location $here
try {
    $env:GO111MODULE = "off"
    go run .\main.go `
        -api $ApiBase `
        -admin-user $AdminUser `
        -admin-pass $AdminPass `
        -agent-token $AgentToken `
        -node-id $NodeId
} finally {
    Pop-Location
}

if ($VerifyPersistence) {
    Write-Host "Verifying persisted config files in WSL..."
    Test-WslFile "$WorkDir/conf/cdn_config.json"
    Test-WslFile "$WorkDir/conf/nginx.conf"
    Test-WslFile "$WorkDir/conf/dynamic/http.conf"
    Test-WslFile "$WorkDir/conf/dynamic/main.conf"
    Test-WslFile "$WorkDir/conf/dynamic/events.conf"
    Test-WslFile "$WorkDir/conf/resources.json" $false
    Test-WslFile "$WorkDir/conf/error_pages.json" $false
    Test-WslFile "$WorkDir/conf/default_config.json" $false
    Test-WslFile "$WorkDir/conf/cc_rules.json" $false
    Test-WslFile "$WorkDir/conf/cc_matchers.json" $false
    Test-WslFile "$WorkDir/conf/cc_filters.json" $false
}

if ($VerifyNginx) {
    Write-Host "Verifying nginx config test in WSL..."
    Invoke-Wsl "$NginxBin -p $WorkDir -t -c conf/nginx.conf"
}

if ($CheckAgentLog) {
    Write-Host "Checking agent log for reload errors..."
    $logCmd = "if [ -f /tmp/agent_e2e.out ]; then if tail -n 200 /tmp/agent_e2e.out | grep -E '\\[Error\\] Reload Nginx Failed|Reload after rollback failed' >/dev/null; then exit 2; fi; fi"
    wsl -d $WslDistro -u root -- bash -lc $logCmd
    if ($LASTEXITCODE -eq 2) {
        throw "Agent log reports nginx reload failures. Check /tmp/agent_e2e.out"
    }
    if ($LASTEXITCODE -ne 0) {
        throw "WSL log check failed: $LASTEXITCODE"
    }
}
