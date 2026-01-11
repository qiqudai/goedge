$ErrorActionPreference = "Stop"

$baseUrl = $env:CDN_API_BASE
if ([string]::IsNullOrWhiteSpace($baseUrl)) {
    $baseUrl = "http://127.0.0.1:8080"
}

$token = $env:CDN_AGENT_TOKEN
$nodeId = $env:CDN_AGENT_NODE_ID
$adminUser = $env:CDN_ADMIN_USER
$adminPass = $env:CDN_ADMIN_PASS

if ([string]::IsNullOrWhiteSpace($token)) {
    Write-Host "CDN_AGENT_TOKEN is required for WS test."
    exit 1
}
if ([string]::IsNullOrWhiteSpace($nodeId)) {
    Write-Host "CDN_AGENT_NODE_ID is required for WS dispatch test."
    exit 1
}
if ([string]::IsNullOrWhiteSpace($adminUser) -or [string]::IsNullOrWhiteSpace($adminPass)) {
    Write-Host "CDN_ADMIN_USER and CDN_ADMIN_PASS are required for WS dispatch test."
    exit 1
}

$login = Invoke-RestMethod -Method Post -Uri "$baseUrl/api/v1/admin/login" -Body (@{username=$adminUser;password=$adminPass} | ConvertTo-Json) -ContentType "application/json"
$adminToken = $login.token
if ([string]::IsNullOrWhiteSpace($adminToken)) {
    Write-Host "Admin login failed."
    exit 1
}

$body = @{
    node_id = [int64]$nodeId
    task_type = "config_sync"
    payload = ""
    wait_seconds = 8
}
$resp = Invoke-RestMethod -Method Post -Uri "$baseUrl/api/v1/admin/ws/dispatch" -Body ($body | ConvertTo-Json) -Headers @{Authorization="Bearer $adminToken"} -ContentType "application/json"
if ($resp.code -ne 0) {
    Write-Host ("WS dispatch failed: {0}" -f ($resp.msg | Out-String))
    exit 1
}
if (-not $resp.data.connected) {
    Write-Host "WS dispatch failed: node not connected."
    exit 1
}
if ($resp.data.state -and $resp.data.state -ne "success") {
    Write-Host ("WS dispatch task state: {0}" -f $resp.data.state)
    exit 1
}

$packagePayload = '{"packages":[{"package_id":999,"version":1,"config":{"version":1}}]}'
$body = @{
    node_id = [int64]$nodeId
    task_type = "package_sync"
    payload = $packagePayload
    wait_seconds = 8
}
$resp = Invoke-RestMethod -Method Post -Uri "$baseUrl/api/v1/admin/ws/dispatch" -Body ($body | ConvertTo-Json) -Headers @{Authorization="Bearer $adminToken"} -ContentType "application/json"
if ($resp.code -ne 0) {
    Write-Host ("WS dispatch failed: {0}" -f ($resp.msg | Out-String))
    exit 1
}
if (-not $resp.data.connected) {
    Write-Host "WS dispatch failed: node not connected."
    exit 1
}
if ($resp.data.state -and $resp.data.state -ne "success") {
    Write-Host ("WS dispatch task state: {0} error: {1}" -f $resp.data.state, $resp.data.error)
    exit 1
}

Write-Host "WS smoke test passed."
