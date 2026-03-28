$ErrorActionPreference = "Stop"

$baseUrl = $env:CDN_API_BASE
if ([string]::IsNullOrWhiteSpace($baseUrl)) {
    $baseUrl = "http://127.0.0.1:8080"
}

$adminUser = $env:CDN_ADMIN_USER
$adminPass = $env:CDN_ADMIN_PASS
$nodeId = $env:CDN_AGENT_NODE_ID
$siteId = $env:CDN_TEST_SITE_ID
if ([string]::IsNullOrWhiteSpace($siteId)) {
    $siteId = "26"
}
$origin = $env:CDN_TEST_ORIGIN
if ([string]::IsNullOrWhiteSpace($origin)) {
    $origin = "172.20.0.1:8080"
}
$testPath = $env:CDN_TEST_PATH
if ([string]::IsNullOrWhiteSpace($testPath)) {
    $testPath = "/health"
}

if ([string]::IsNullOrWhiteSpace($adminUser) -or [string]::IsNullOrWhiteSpace($adminPass)) {
    Write-Host "CDN_ADMIN_USER and CDN_ADMIN_PASS are required."
    exit 1
}
if ([string]::IsNullOrWhiteSpace($nodeId)) {
    Write-Host "CDN_AGENT_NODE_ID is required."
    exit 1
}

function Invoke-Api {
    param(
        [string]$Method,
        [string]$Url,
        [string]$Token = "",
        [object]$Body = $null
    )

    $headers = @{}
    if (-not [string]::IsNullOrWhiteSpace($Token)) {
        $headers["Authorization"] = "Bearer $Token"
    }

    if ($Method -eq "GET") {
        return Invoke-RestMethod -Method $Method -Uri $Url -Headers $headers
    }
    $payload = $null
    if ($Body -ne $null) {
        $payload = $Body | ConvertTo-Json -Depth 10
    }
    return Invoke-RestMethod -Method $Method -Uri $Url -Headers $headers -Body $payload -ContentType "application/json"
}

function Dispatch-Sync {
    param(
        [string]$Token,
        [string]$NodeId
    )
    $body = @{
        node_id = [int64]$NodeId
        task_type = "config_sync"
        payload = ""
        wait_seconds = 8
    }
    $resp = Invoke-Api -Method "POST" -Url "$baseUrl/api/v1/admin/ws/dispatch" -Token $Token -Body $body
    if ($resp.code -ne 0 -or -not $resp.data.connected) {
        throw "WS dispatch failed: $($resp | ConvertTo-Json -Depth 6)"
    }
}

function Invoke-Wsl {
    param([string]$Cmd)
    wsl -d Ubuntu-24.04 -u root -- bash -lc $Cmd
}

function Invoke-WslCurl {
    param(
        [string]$Hostname,
        [string]$Path = "/"
    )
    $cmd = "curl -s -o /dev/null -w '%{http_code} %{size_download}' -H 'Host: $Hostname' http://127.0.0.1$Path"
    $out = Invoke-Wsl $cmd
    return $out.Trim()
}

function Assert-Equal {
    param(
        [string]$Name,
        [string]$Actual,
        [string]$Expected
    )
    if ($Actual -ne $Expected) {
        throw "$Name expected '$Expected' got '$Actual'"
    }
}

function Assert-Status {
    param(
        [string]$Name,
        [string]$Actual,
        [string]$Expected
    )
    $code = ($Actual -split '\s+')[0]
    if ($code -ne $Expected) {
        throw "$Name expected '$Expected' got '$code' (raw '$Actual')"
    }
}

$login = Invoke-Api -Method "POST" -Url "$baseUrl/api/v1/admin/login" -Body @{ username = $adminUser; password = $adminPass }
$adminToken = $login.token
if ([string]::IsNullOrWhiteSpace($adminToken)) {
    Write-Host "Admin login failed."
    exit 1
}

$site = Invoke-Api -Method "GET" -Url "$baseUrl/api/v1/admin/sites/$siteId" -Token $adminToken
$origBackends = $site.data.site.backend
$origProtocol = $site.data.site.backend_protocol
$origState = $site.data.site.state

try {
    $update = @{
        backends = @($origin)
        backend_protocol = "http"
    }
    Invoke-Api -Method "PUT" -Url "$baseUrl/api/v1/admin/sites/$siteId" -Token $adminToken -Body $update | Out-Null
    Dispatch-Sync -Token $adminToken -NodeId $nodeId
    Start-Sleep -Seconds 1

    $results = @{}

    Invoke-Api -Method "PUT" -Url "$baseUrl/api/v1/admin/sites/$siteId" -Token $adminToken -Body @{ state = "locked" } | Out-Null
    Dispatch-Sync -Token $adminToken -NodeId $nodeId
    Start-Sleep -Seconds 1
    $results.locked = Invoke-WslCurl -Hostname "ws-test.local"

    Invoke-Api -Method "PUT" -Url "$baseUrl/api/v1/admin/sites/$siteId" -Token $adminToken -Body @{ state = "conn_limit" } | Out-Null
    Dispatch-Sync -Token $adminToken -NodeId $nodeId
    Start-Sleep -Seconds 1
    $results.conn_limit = Invoke-WslCurl -Hostname "ws-test.local"

    Invoke-Api -Method "PUT" -Url "$baseUrl/api/v1/admin/sites/$siteId" -Token $adminToken -Body @{ state = "expired" } | Out-Null
    Dispatch-Sync -Token $adminToken -NodeId $nodeId
    Start-Sleep -Seconds 1
    $results.timeout = Invoke-WslCurl -Hostname "ws-test.local"

    Invoke-Api -Method "PUT" -Url "$baseUrl/api/v1/admin/sites/$siteId" -Token $adminToken -Body @{ state = "traffic_limit" } | Out-Null
    Dispatch-Sync -Token $adminToken -NodeId $nodeId
    Start-Sleep -Seconds 1
    $results.traffic_limit = Invoke-WslCurl -Hostname "ws-test.local"

    Invoke-Api -Method "PUT" -Url "$baseUrl/api/v1/admin/sites/$siteId" -Token $adminToken -Body @{ state = "running" } | Out-Null
    Dispatch-Sync -Token $adminToken -NodeId $nodeId
    Start-Sleep -Seconds 1
    $results.running = Invoke-WslCurl -Hostname "ws-test.local" -Path $testPath

    $results.domain_invalid = Invoke-WslCurl -Hostname "invalid.local"

    Assert-Status -Name "site_locked" -Actual $results.locked -Expected "451"
    Assert-Status -Name "conn_limit" -Actual $results.conn_limit -Expected "429"
    Assert-Status -Name "timeout" -Actual $results.timeout -Expected "410"
    Assert-Status -Name "traffic_limit" -Actual $results.traffic_limit -Expected "509"
    Assert-Status -Name "running" -Actual $results.running -Expected "200"
    Assert-Status -Name "domain_invalid" -Actual $results.domain_invalid -Expected "404"

    Write-Host "Error page checks passed."
} finally {
    $restore = @{}
    if ($origBackends) { $restore.backends = @($origBackends) }
    if ($origProtocol) { $restore.backend_protocol = $origProtocol }
    if ($origState) { $restore.state = $origState }
    if ($restore.Count -gt 0) {
        Invoke-Api -Method "PUT" -Url "$baseUrl/api/v1/admin/sites/$siteId" -Token $adminToken -Body $restore | Out-Null
        Dispatch-Sync -Token $adminToken -NodeId $nodeId
        Start-Sleep -Seconds 1
    }
    # No local origin process to stop.
}
