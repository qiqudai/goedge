$ErrorActionPreference = "Stop"

$baseUrl = $env:CDN_API_BASE
if ([string]::IsNullOrWhiteSpace($baseUrl)) {
    $baseUrl = "http://127.0.0.1:8080"
}

$adminUser = $env:CDN_ADMIN_USER
$adminPass = $env:CDN_ADMIN_PASS
$userUser = $env:CDN_USER_USER
$userPass = $env:CDN_USER_PASS
$agentToken = $env:CDN_AGENT_TOKEN
$agentNodeId = $env:CDN_AGENT_NODE_ID

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

    try {
        if ($Method -eq "GET") {
            $resp = Invoke-WebRequest -UseBasicParsing -Method $Method -Uri $Url -Headers $headers
        } else {
            $payload = $null
            if ($Body -ne $null) {
                $payload = $Body | ConvertTo-Json -Depth 10
            }
            $resp = Invoke-WebRequest -UseBasicParsing -Method $Method -Uri $Url -Headers $headers -Body $payload -ContentType "application/json"
        }
        return @{ ok = $true; status = $resp.StatusCode }
    } catch {
        if ($_.Exception.Response -ne $null) {
            return @{ ok = $false; status = $_.Exception.Response.StatusCode.Value__ }
        }
        return @{ ok = $false; status = 0 }
    }
}

function Login {
    param(
        [string]$Role,
        [string]$Username,
        [string]$Password
    )
    if ([string]::IsNullOrWhiteSpace($Username) -or [string]::IsNullOrWhiteSpace($Password)) {
        return ""
    }
    $body = @{ username = $Username; password = $Password }
    $resp = Invoke-Api -Method "POST" -Url "$baseUrl/api/v1/$Role/login" -Body $body
    if (-not $resp.ok) {
        return ""
    }
    try {
        $json = Invoke-RestMethod -Method "POST" -Uri "$baseUrl/api/v1/$Role/login" -Body ($body | ConvertTo-Json) -ContentType "application/json"
        return $json.token
    } catch {
        return ""
    }
}

$results = @()

function Add-Check {
    param(
        [string]$Name,
        [string]$Method,
        [string]$Url,
        [string]$Token = ""
    )
    $results += [PSCustomObject]@{
        Name = $Name
        Method = $Method
        Url = $Url
        Token = $Token
        Ok = $false
        Status = 0
    }
}

Add-Check -Name "health" -Method "GET" -Url "$baseUrl/health"

$adminToken = Login -Role "admin" -Username $adminUser -Password $adminPass
if (-not [string]::IsNullOrWhiteSpace($adminToken)) {
    $adminGets = @(
        "/api/v1/admin/global_config",
        "/api/v1/admin/config_items",
        "/api/v1/admin/nodes",
        "/api/v1/admin/node-groups",
        "/api/v1/admin/regions",
        "/api/v1/admin/dns/providers",
        "/api/v1/admin/cname_domains",
        "/api/v1/admin/monitor_config",
        "/api/v1/admin/logs/login",
        "/api/v1/admin/logs/operation",
        "/api/v1/admin/logs/access",
        "/api/v1/admin/logs/backup",
        "/api/v1/admin/logs/mail",
        "/api/v1/admin/logs/block/current",
        "/api/v1/admin/logs/block/stats",
        "/api/v1/admin/logs/block/history",
        "/api/v1/admin/messages",
        "/api/v1/admin/stats/basic",
        "/api/v1/admin/stats/quality",
        "/api/v1/admin/stats/origin",
        "/api/v1/admin/stats/ranking",
        "/api/v1/admin/stats/node_traffic",
        "/api/v1/admin/dashboard",
        "/api/v1/admin/packages",
        "/api/v1/admin/plans",
        "/api/v1/admin/user_plans",
        "/api/v1/admin/orders",
        "/api/v1/admin/announcements",
        "/api/v1/admin/system_info",
        "/api/v1/admin/api_key",
        "/api/v1/admin/domains",
        "/api/v1/admin/users",
        "/api/v1/admin/sites",
        "/api/v1/admin/site_groups",
        "/api/v1/admin/site_defaults",
        "/api/v1/admin/user_packages",
        "/api/v1/admin/certs",
        "/api/v1/admin/dnsapi",
        "/api/v1/admin/forwards",
        "/api/v1/admin/forward_groups",
        "/api/v1/admin/forward_defaults",
        "/api/v1/admin/tasks",
        "/api/v1/admin/tasks/usage",
        "/api/v1/admin/rules/cc/groups",
        "/api/v1/admin/rules/cc/matchers",
        "/api/v1/admin/rules/cc/filters",
        "/api/v1/admin/rules/acl"
    )
    foreach ($path in $adminGets) {
        Add-Check -Name "admin GET $path" -Method "GET" -Url ($baseUrl + $path) -Token $adminToken
    }
}

$userToken = Login -Role "user" -Username $userUser -Password $userPass
if (-not [string]::IsNullOrWhiteSpace($userToken)) {
    $userGets = @(
        "/api/v1/user/profile",
        "/api/v1/user/domains",
        "/api/v1/user/config_items",
        "/api/v1/user/orders",
        "/api/v1/user/messages",
        "/api/v1/user/message_sub",
        "/api/v1/user/api_key",
        "/api/v1/user/sites",
        "/api/v1/user/certs",
        "/api/v1/user/tasks",
        "/api/v1/user/tasks/usage",
        "/api/v1/user/plans",
        "/api/v1/user/user_packages",
        "/api/v1/user/site_groups",
        "/api/v1/user/site_defaults",
        "/api/v1/user/dns/providers",
        "/api/v1/user/dnsapi",
        "/api/v1/user/rules/cc/groups",
        "/api/v1/user/rules/cc/matchers",
        "/api/v1/user/rules/cc/filters",
        "/api/v1/user/rules/acl",
        "/api/v1/user/logs/access",
        "/api/v1/user/logs/block/current",
        "/api/v1/user/logs/block/stats",
        "/api/v1/user/logs/block/history",
        "/api/v1/user/stats/basic",
        "/api/v1/user/stats/quality",
        "/api/v1/user/stats/origin",
        "/api/v1/user/stats/ranking",
        "/api/v1/user/usage",
        "/api/v1/user/forwards",
        "/api/v1/user/forward_groups",
        "/api/v1/user/forward_defaults"
    )
    foreach ($path in $userGets) {
        Add-Check -Name "user GET $path" -Method "GET" -Url ($baseUrl + $path) -Token $userToken
    }
}

if (-not [string]::IsNullOrWhiteSpace($agentToken) -and -not [string]::IsNullOrWhiteSpace($agentNodeId)) {
    $agentGets = @(
        "/api/v1/agent/config?node_id=$agentNodeId",
        "/api/v1/agent/tasks",
        "/api/v1/agent/l2/nodes"
    )
    foreach ($path in $agentGets) {
        Add-Check -Name "agent GET $path" -Method "GET" -Url ($baseUrl + $path) -Token $agentToken
    }
}

$failures = 0
foreach ($check in $results) {
    $resp = Invoke-Api -Method $check.Method -Url $check.Url -Token $check.Token
    $check.Ok = $resp.ok
    $check.Status = $resp.status
    if (-not $check.Ok) {
        $failures++
    }
}

$results | Format-Table -AutoSize Name, Method, Status, Ok

if ($failures -gt 0) {
    Write-Host ("Smoke tests failed: {0}" -f $failures)
    exit 1
}

Write-Host "Smoke tests passed."
