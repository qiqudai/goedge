$ErrorActionPreference = "Stop"

$baseUrl = $env:CDN_API_BASE
if ([string]::IsNullOrWhiteSpace($baseUrl)) {
    $baseUrl = "http://127.0.0.1:8080"
}

$adminUser = $env:CDN_ADMIN_USER
$adminPass = $env:CDN_ADMIN_PASS
if ([string]::IsNullOrWhiteSpace($adminUser) -or [string]::IsNullOrWhiteSpace($adminPass)) {
    Write-Host "CDN_ADMIN_USER and CDN_ADMIN_PASS are required."
    exit 1
}

$domain = $env:CDN_TEST_DOMAIN
if ([string]::IsNullOrWhiteSpace($domain)) {
    $domain = "test.665305.cc"
}

$distro = $env:CDN_WSL_DISTRO
if ([string]::IsNullOrWhiteSpace($distro)) {
    $distro = "Ubuntu-24.04"
}

$resolveIp = $env:CDN_TLS_RESOLVE_IP
if ([string]::IsNullOrWhiteSpace($resolveIp)) {
    $resolveIp = "127.0.0.1"
}

$agentConfigPath = $env:CDN_AGENT_CONFIG_PATH
if ([string]::IsNullOrWhiteSpace($agentConfigPath)) {
    $agentConfigPath = "\\wsl$\$distro\usr\local\goedge\nodes\edge-node\conf\cdn_config.json"
}

function Invoke-Wsl {
    param(
        [string]$Command
    )
    $wrapped = "$Command 2>&1"
    $output = & wsl.exe -d $distro -- bash -lc $wrapped
    return @{
        Output = ($output -join "`n")
        ExitCode = $LASTEXITCODE
    }
}

function Get-AdminToken {
    $body = @{ username = $adminUser; password = $adminPass } | ConvertTo-Json
    $resp = Invoke-RestMethod -Method Post -Uri "$baseUrl/api/v1/admin/login" -Body $body -ContentType "application/json"
    if ($resp.token) {
        return $resp.token
    }
    if ($resp.data -and $resp.data.token) {
        return $resp.data.token
    }
    return ""
}

function Extract-List {
    param(
        [object]$Resp
    )
    if ($Resp.list) {
        return $Resp.list
    }
    if ($Resp.data -and $Resp.data.list) {
        return $Resp.data.list
    }
    return @()
}

$failures = 0
function Assert-True {
    param(
        [bool]$Condition,
        [string]$Message
    )
    if ($Condition) {
        Write-Host "OK: $Message"
        return
    }
    Write-Host "FAIL: $Message"
    $script:failures++
}

$token = Get-AdminToken
if ([string]::IsNullOrWhiteSpace($token)) {
    Write-Host "Admin login failed."
    exit 1
}

Write-Host "Checking cert status for $domain..."
$encodedDomain = [System.Uri]::EscapeDataString($domain)
$certUrl = "$baseUrl/api/v1/admin/certs?page=1&pageSize=50&search_field=domain&keyword=$encodedDomain"
$certResp = Invoke-RestMethod -Method Get -Uri $certUrl -Headers @{Authorization = "Bearer $token"}
$certList = Extract-List $certResp
$cert = $certList | Where-Object { $_.domain -like "*$domain*" } | Select-Object -First 1
Assert-True ($null -ne $cert) "cert found in admin list"
if ($null -ne $cert) {
    Assert-True ($cert.state -eq "ready") "cert state is ready"
    Assert-True ($cert.enable -eq $true) "cert is enabled"
    Assert-True (-not [string]::IsNullOrWhiteSpace($cert.cert)) "cert PEM is present"
    Assert-True (-not [string]::IsNullOrWhiteSpace($cert.key)) "key PEM is present"
    Assert-True ($null -ne $cert.expire_time) "expire time is present"
}

$taskId = 0
if ($cert -and $cert.task_id) {
    $taskId = [int64]$cert.task_id
}

if ($taskId -gt 0) {
    Write-Host "Checking task $taskId..."
    $taskUrl = "$baseUrl/api/v1/admin/tasks?page=1&pageSize=500&type=issue_cert"
    $taskResp = Invoke-RestMethod -Method Get -Uri $taskUrl -Headers @{Authorization = "Bearer $token"}
    $taskList = Extract-List $taskResp
    $task = $taskList | Where-Object { $_.id -eq $taskId } | Select-Object -First 1
    Assert-True ($null -ne $task) "issue_cert task found"
    if ($null -ne $task) {
        Assert-True ($task.state -eq "success") "task state is success"
    }
} else {
    Write-Host "SKIP: no task_id on cert."
}

Write-Host "Checking agent config sync..."
if (-not (Test-Path $agentConfigPath)) {
    Assert-True $false "agent config path exists: $agentConfigPath"
} else {
    $config = Get-Content -Raw -Path $agentConfigPath | ConvertFrom-Json
    $domainEntry = $config.domains | Where-Object { $_.name -eq $domain } | Select-Object -First 1
    Assert-True ($null -ne $domainEntry) "agent config contains domain"
    if ($null -ne $domainEntry) {
        $certData = $domainEntry.ssl_cert_data
        if ([string]::IsNullOrWhiteSpace($certData)) {
            $certData = $domainEntry.ssl_cert_path
        }
        $keyData = $domainEntry.ssl_key_data
        if ([string]::IsNullOrWhiteSpace($keyData)) {
            $keyData = $domainEntry.ssl_key_path
        }
        Assert-True (-not [string]::IsNullOrWhiteSpace($certData)) "agent has cert data or path"
        Assert-True (-not [string]::IsNullOrWhiteSpace($keyData)) "agent has key data or path"
        Assert-True ($domainEntry.https_http2 -eq $true) "agent https_http2 enabled"
    }
}

Write-Host "Checking TLS handshake via WSL curl..."
$curlCommand = "curl -vkI --connect-timeout 5 --max-time 15 --resolve ${domain}:443:${resolveIp} https://${domain}"
$curlResult = Invoke-Wsl -Command $curlCommand
Assert-True ($curlResult.ExitCode -eq 0) "curl exit code is 0"
if ($curlResult.ExitCode -eq 0) {
    $curlOutput = $curlResult.Output
    $hasHttp2 = ($curlOutput -match "ALPN: server accepted h2") -or ($curlOutput -match "HTTP/2")
    $hasSubject = ($curlOutput -match ("subject:.*" + [regex]::Escape($domain))) -or
        ($curlOutput -match ("subjectAltName:.*" + [regex]::Escape($domain)))
    Assert-True $hasHttp2 "TLS negotiated HTTP/2"
    Assert-True $hasSubject "TLS subject/SAN contains domain"
}

if ($failures -gt 0) {
    Write-Host ("Verification failed: {0}" -f $failures)
    exit 1
}

Write-Host "Verification passed."
