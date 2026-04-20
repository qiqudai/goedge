using System.Collections.Concurrent;
using System.Diagnostics;
using System.Net;
using Microsoft.AspNetCore.Http.Features;
using Cnn.Agent.Diagnostics;
using Cnn.Agent.Logs;
using Cnn.Common.Contracts.Agent;

namespace Cnn.Agent.Security;

public sealed class SiteSecurityMiddleware
{
    private readonly RequestDelegate _next;
    private readonly IEdgeDomainResolver _resolver;
    private readonly ISecurityDecisionService _securityDecisionService;
    private readonly IDebugSessionService _debugSessionService;
    private readonly ILogEventWriter _logWriter;
    private readonly ILogger<SiteSecurityMiddleware> _logger;
    private readonly ConcurrentDictionary<string, int> _activeConnectionsByHost = new(StringComparer.OrdinalIgnoreCase);

    public SiteSecurityMiddleware(
        RequestDelegate next,
        IEdgeDomainResolver resolver,
        ISecurityDecisionService securityDecisionService,
        IDebugSessionService debugSessionService,
        ILogEventWriter logWriter,
        ILogger<SiteSecurityMiddleware> logger)
    {
        _next = next;
        _resolver = resolver;
        _securityDecisionService = securityDecisionService;
        _debugSessionService = debugSessionService;
        _logWriter = logWriter;
        _logger = logger;
    }

    public async Task InvokeAsync(HttpContext context)
    {
        var started = Stopwatch.GetTimestamp();
        var host = context.Request.Host.Host?.Trim().ToLowerInvariant() ?? string.Empty;
        var traceId = context.TraceIdentifier ?? string.Empty;
        var method = context.Request.Method;
        var path = context.Request.Path.ToString();
        var query = context.Request.QueryString.ToString();
        var clientIp = NormalizeIp(context.Connection.RemoteIpAddress);

        if (_debugSessionService.TryAllowEvent("security", context, out var debugSession))
        {
            _ = _logWriter.TryWrite(new LogEvent(
                DateTimeOffset.UtcNow,
                LogChannels.Debug,
                "debug",
                "security_debug_probe",
                traceId,
                DebugLogSanitizer.Sanitize(new Dictionary<string, object?>
                {
                    ["session_id"] = debugSession,
                    ["host"] = host,
                    ["client_ip"] = clientIp,
                    ["method"] = method,
                    ["path"] = path,
                    ["query"] = query
                })));
        }

        EdgeDomainDto? domain = null;
        if (!string.IsNullOrWhiteSpace(host) && _resolver.TryResolve(host, out var resolved))
        {
            domain = resolved;
        }

        var originalBody = context.Response.Body;
        RateLimitedWriteStream? rateLimitedBody = null;
        var incConnCounter = false;
        try
        {
            if (domain != null)
            {
                ApplyHttpsHeaders(context, domain);

                if (TryHandleStatusBlock(context, domain, traceId, clientIp))
                {
                    return;
                }

                if (TryHandleHttpsForceRedirect(context, domain))
                {
                    return;
                }

                if (TryHandleIpPolicyBlock(context, domain, traceId, clientIp))
                {
                    return;
                }

                if (TryHandleHotlinkBlock(context, domain, traceId, clientIp))
                {
                    return;
                }

                if (TryHandleWebSocketPolicyBlock(context, domain, traceId, clientIp))
                {
                    return;
                }

                if (TryApplyBodyLimit(context, domain, traceId, clientIp))
                {
                    return;
                }

                if (TryApplyRateLimit(context, domain, out var limitedBody))
                {
                    rateLimitedBody = limitedBody;
                    context.Response.Body = limitedBody;
                }

                ApplyCorsHeaders(context, domain);
                if (IsCorsPreflight(context))
                {
                    context.Response.StatusCode = StatusCodes.Status204NoContent;
                    return;
                }

                var decision = _securityDecisionService.Evaluate(context, domain);
                if (!decision.Allowed)
                {
                    context.Response.StatusCode = decision.StatusCode;
                    if (!string.IsNullOrWhiteSpace(decision.ErrorPageKey))
                    {
                        context.Items["error_page_key"] = decision.ErrorPageKey;
                    }

                    _ = _logWriter.TryWrite(BuildSecurityEvent(
                        traceId,
                        decision.RuleType ?? "security",
                        host,
                        clientIp,
                        context.Request.Path,
                        new Dictionary<string, object?>
                        {
                            ["rule_id"] = decision.RuleId,
                            ["reason"] = decision.Reason,
                            ["status"] = decision.StatusCode,
                            ["error_page_key"] = decision.ErrorPageKey
                        }));
                    return;
                }

                var connLimit = domain.ConnLimit.GetValueOrDefault();
                if (connLimit > 0)
                {
                    var active = _activeConnectionsByHost.AddOrUpdate(host, 1, static (_, current) => current + 1);
                    if (active > connLimit)
                    {
                        _activeConnectionsByHost.AddOrUpdate(host, 0, static (_, current) => Math.Max(0, current - 1));
                        context.Response.StatusCode = StatusCodes.Status429TooManyRequests;
                        _ = _logWriter.TryWrite(BuildSecurityEvent(
                            traceId,
                            "conn_limit",
                            host,
                            clientIp,
                            context.Request.Path,
                            new Dictionary<string, object?>
                            {
                                ["limit"] = connLimit,
                                ["active"] = active
                            }));
                        return;
                    }

                    incConnCounter = true;
                }
            }

            await _next(context);
        }
        finally
        {
            if (rateLimitedBody != null)
            {
                context.Response.Body = originalBody;
                await rateLimitedBody.DisposeAsync();
            }

            if (incConnCounter && !string.IsNullOrWhiteSpace(host))
            {
                _activeConnectionsByHost.AddOrUpdate(host, 0, static (_, current) => Math.Max(0, current - 1));
            }

            var elapsedMs = Stopwatch.GetElapsedTime(started).TotalMilliseconds;
            var fields = DebugLogSanitizer.Sanitize(new Dictionary<string, object?>
            {
                ["host"] = host,
                ["client_ip"] = clientIp,
                ["method"] = method,
                ["path"] = path,
                ["query"] = query,
                ["status"] = context.Response.StatusCode,
                ["latency_ms"] = Math.Round(elapsedMs, 3),
                ["matched_domain"] = domain?.Name,
                ["https"] = context.Request.IsHttps
            });

            _ = _logWriter.TryWrite(new LogEvent(
                DateTimeOffset.UtcNow,
                LogChannels.Access,
                "information",
                "http_access",
                traceId,
                fields));
        }
    }

    private bool TryHandleStatusBlock(HttpContext context, EdgeDomainDto domain, string traceId, string clientIp)
    {
        var status = domain.Status?.Trim().ToLowerInvariant();
        if (string.IsNullOrWhiteSpace(status))
        {
            return false;
        }

        if (status is "running" or "active")
        {
            return false;
        }

        context.Response.StatusCode = status is "stop" ? StatusCodes.Status503ServiceUnavailable : StatusCodes.Status403Forbidden;
        _ = _logWriter.TryWrite(BuildSecurityEvent(
            traceId,
            "domain_status",
            context.Request.Host.Host,
            clientIp,
            context.Request.Path,
            new Dictionary<string, object?>
            {
                ["status"] = status
            }));
        return true;
    }

    private static bool TryHandleHttpsForceRedirect(HttpContext context, EdgeDomainDto domain)
    {
        if (!domain.HttpsForce.GetValueOrDefault() || context.Request.IsHttps)
        {
            return false;
        }

        var host = context.Request.Host.Host;
        var path = context.Request.Path + context.Request.QueryString;
        var httpsPort = ResolveHttpsPort(domain.HttpsRedirectPort);
        var redirect = httpsPort == 443 ? $"https://{host}{path}" : $"https://{host}:{httpsPort}{path}";
        context.Response.Redirect(redirect, permanent: false);
        return true;
    }

    private bool TryHandleIpPolicyBlock(HttpContext context, EdgeDomainDto domain, string traceId, string clientIp)
    {
        var ip = context.Connection.RemoteIpAddress;
        if (ip == null)
        {
            return false;
        }

        if (IpRuleMatcher.IsMatch(ip, domain.WhiteIps))
        {
            return false;
        }

        if (IpRuleMatcher.IsMatch(ip, domain.BlackIps))
        {
            context.Response.StatusCode = StatusCodes.Status403Forbidden;
            _ = _logWriter.TryWrite(BuildSecurityEvent(
                traceId,
                "ip_blacklist",
                context.Request.Host.Host,
                clientIp,
                context.Request.Path,
                new Dictionary<string, object?>()));
            return true;
        }

        if (IsRegionBlocked(context, domain.RegionBlock, out var countryCode))
        {
            context.Response.StatusCode = StatusCodes.Status403Forbidden;
            _ = _logWriter.TryWrite(BuildSecurityEvent(
                traceId,
                "region_block",
                context.Request.Host.Host,
                clientIp,
                context.Request.Path,
                new Dictionary<string, object?>
                {
                    ["country"] = countryCode
                }));
            return true;
        }

        var acl = domain.AclRules;
        if (acl == null || acl.Count == 0)
        {
            return false;
        }

        foreach (var rule in acl)
        {
            if (!IpRuleMatcher.IsMatch(ip, rule.Ip))
            {
                continue;
            }

            var action = rule.Action?.Trim().ToLowerInvariant();
            if (action is "deny" or "block")
            {
                context.Response.StatusCode = StatusCodes.Status403Forbidden;
                _ = _logWriter.TryWrite(BuildSecurityEvent(
                    traceId,
                    "acl_block",
                    context.Request.Host.Host,
                    clientIp,
                    context.Request.Path,
                    new Dictionary<string, object?>
                    {
                        ["acl_ip"] = rule.Ip,
                        ["acl_action"] = action
                    }));
                return true;
            }

            if (action == "allow")
            {
                return false;
            }
        }

        var defaultAction = domain.AclDefaultAction?.Trim().ToLowerInvariant();
        if (defaultAction is "deny" or "block")
        {
            context.Response.StatusCode = StatusCodes.Status403Forbidden;
            _ = _logWriter.TryWrite(BuildSecurityEvent(
                traceId,
                "acl_default_block",
                context.Request.Host.Host,
                clientIp,
                context.Request.Path,
                new Dictionary<string, object?>
                {
                    ["acl_default_action"] = defaultAction
                }));
            return true;
        }

        return false;
    }

    private bool TryHandleHotlinkBlock(HttpContext context, EdgeDomainDto domain, string traceId, string clientIp)
    {
        var hotlink = domain.Hotlink;
        if (hotlink == null || !hotlink.Enable)
        {
            return false;
        }

        var referer = context.Request.Headers.Referer.ToString();
        var origin = context.Request.Headers.Origin.ToString();

        var sourceHost = ExtractHost(referer);
        if (string.IsNullOrWhiteSpace(sourceHost))
        {
            sourceHost = ExtractHost(origin);
        }

        if (string.IsNullOrWhiteSpace(sourceHost))
        {
            if (hotlink.AllowEmpty.GetValueOrDefault())
            {
                return false;
            }

            context.Response.StatusCode = StatusCodes.Status403Forbidden;
            _ = _logWriter.TryWrite(BuildSecurityEvent(
                traceId,
                "hotlink_block_empty",
                context.Request.Host.Host,
                clientIp,
                context.Request.Path,
                new Dictionary<string, object?>()));
            return true;
        }

        var allowed = ResolveAllowedDomains(hotlink).ToHashSet(StringComparer.OrdinalIgnoreCase);
        var requestHost = context.Request.Host.Host;
        allowed.Add(requestHost);
        if (allowed.Contains(sourceHost))
        {
            return false;
        }

        context.Response.StatusCode = StatusCodes.Status403Forbidden;
        _ = _logWriter.TryWrite(BuildSecurityEvent(
            traceId,
            "hotlink_block",
            context.Request.Host.Host,
            clientIp,
            context.Request.Path,
            new Dictionary<string, object?>
            {
                ["source_host"] = sourceHost
            }));
        return true;
    }

    private bool TryHandleWebSocketPolicyBlock(HttpContext context, EdgeDomainDto domain, string traceId, string clientIp)
    {
        if (domain.EnableWebsocket.GetValueOrDefault(true))
        {
            return false;
        }

        if (!IsWebSocketUpgradeRequest(context.Request))
        {
            return false;
        }

        context.Response.StatusCode = StatusCodes.Status403Forbidden;
        _ = _logWriter.TryWrite(BuildSecurityEvent(
            traceId,
            "websocket_disabled",
            context.Request.Host.Host,
            clientIp,
            context.Request.Path,
            new Dictionary<string, object?>()));
        return true;
    }

    private bool TryApplyBodyLimit(HttpContext context, EdgeDomainDto domain, string traceId, string clientIp)
    {
        var bodyLimit = domain.BodyLimit.GetValueOrDefault();
        if (bodyLimit <= 0)
        {
            return false;
        }

        var maxRequestFeature = context.Features.Get<IHttpMaxRequestBodySizeFeature>();
        if (maxRequestFeature != null && !maxRequestFeature.IsReadOnly)
        {
            maxRequestFeature.MaxRequestBodySize = bodyLimit;
        }

        var contentLength = context.Request.ContentLength;
        if (!contentLength.HasValue || contentLength.Value <= bodyLimit)
        {
            return false;
        }

        context.Response.StatusCode = StatusCodes.Status413PayloadTooLarge;
        _ = _logWriter.TryWrite(BuildSecurityEvent(
            traceId,
            "body_limit_exceeded",
            context.Request.Host.Host,
            clientIp,
            context.Request.Path,
            new Dictionary<string, object?>
            {
                ["body_limit"] = bodyLimit,
                ["content_length"] = contentLength.Value
            }));
        return true;
    }

    private static bool IsWebSocketUpgradeRequest(HttpRequest request)
    {
        if (!request.Headers.TryGetValue("Connection", out var connectionHeader)
            || !request.Headers.TryGetValue("Upgrade", out var upgradeHeader))
        {
            return false;
        }

        var connection = connectionHeader.ToString();
        var upgrade = upgradeHeader.ToString();
        return connection.Contains("upgrade", StringComparison.OrdinalIgnoreCase)
               && string.Equals(upgrade.Trim(), "websocket", StringComparison.OrdinalIgnoreCase);
    }

    private static bool TryApplyRateLimit(HttpContext context, EdgeDomainDto domain, out RateLimitedWriteStream rateLimitedBody)
    {
        rateLimitedBody = null!;
        var limitRate = domain.LimitRate.GetValueOrDefault();
        if (limitRate <= 0 || IsWebSocketUpgradeRequest(context.Request))
        {
            return false;
        }

        rateLimitedBody = new RateLimitedWriteStream(context.Response.Body, limitRate);
        return true;
    }

    private static IEnumerable<string> ResolveAllowedDomains(EdgeHotlinkConfigDto config)
    {
        if (config.Domains != null)
        {
            foreach (var item in config.Domains)
            {
                var normalized = NormalizeHost(item);
                if (!string.IsNullOrWhiteSpace(normalized))
                {
                    yield return normalized;
                }
            }
        }

        if (!string.IsNullOrWhiteSpace(config.Value))
        {
            foreach (var part in config.Value.Split([',', ';', ' ', '\n', '\r', '\t'], StringSplitOptions.RemoveEmptyEntries))
            {
                var normalized = NormalizeHost(part);
                if (!string.IsNullOrWhiteSpace(normalized))
                {
                    yield return normalized;
                }
            }
        }
    }

    private static void ApplyCorsHeaders(HttpContext context, EdgeDomainDto domain)
    {
        var cors = domain.Cors;
        if (cors == null || !cors.Enable)
        {
            return;
        }

        var origin = context.Request.Headers.Origin.ToString().Trim();
        var allowOrigin = ResolveCorsAllowOrigin(cors, origin);
        if (string.IsNullOrWhiteSpace(allowOrigin))
        {
            return;
        }

        context.Response.Headers["Access-Control-Allow-Origin"] = allowOrigin;
        context.Response.Headers["Vary"] = "Origin";

        if (!string.IsNullOrWhiteSpace(cors.AllowMethods))
        {
            context.Response.Headers["Access-Control-Allow-Methods"] = cors.AllowMethods;
        }

        if (!string.IsNullOrWhiteSpace(cors.AllowHeaders))
        {
            context.Response.Headers["Access-Control-Allow-Headers"] = cors.AllowHeaders;
        }

        if (!string.IsNullOrWhiteSpace(cors.ExposeHeaders))
        {
            context.Response.Headers["Access-Control-Expose-Headers"] = cors.ExposeHeaders;
        }

        if (cors.AllowCredentials.GetValueOrDefault())
        {
            context.Response.Headers["Access-Control-Allow-Credentials"] = "true";
        }

        if (!string.IsNullOrWhiteSpace(cors.MaxAge))
        {
            context.Response.Headers["Access-Control-Max-Age"] = cors.MaxAge;
        }
    }

    private static void ApplyHttpsHeaders(HttpContext context, EdgeDomainDto domain)
    {
        if (!context.Request.IsHttps || !domain.HttpsHsts.GetValueOrDefault())
        {
            return;
        }

        if (context.Response.Headers.ContainsKey("Strict-Transport-Security"))
        {
            return;
        }

        // Keep defaults explicit and deterministic for agent-side behavior.
        context.Response.Headers["Strict-Transport-Security"] = "max-age=31536000; includeSubDomains";
    }

    private static bool IsCorsPreflight(HttpContext context)
    {
        return HttpMethods.IsOptions(context.Request.Method) &&
               context.Request.Headers.ContainsKey("Origin") &&
               context.Request.Headers.ContainsKey("Access-Control-Request-Method");
    }

    private static string ResolveCorsAllowOrigin(EdgeCorsConfigDto cors, string requestOrigin)
    {
        var allowed = cors.AllowOrigin?.Trim();
        if (string.IsNullOrWhiteSpace(allowed))
        {
            return string.Empty;
        }

        if (allowed == "*")
        {
            if (cors.AllowCredentials.GetValueOrDefault() && !string.IsNullOrWhiteSpace(requestOrigin))
            {
                return requestOrigin;
            }

            return "*";
        }

        var allowedItems = allowed.Split([',', ';', ' ', '\n', '\r', '\t'], StringSplitOptions.RemoveEmptyEntries);
        if (allowedItems.Length == 0)
        {
            return string.Empty;
        }

        if (string.IsNullOrWhiteSpace(requestOrigin))
        {
            return string.Empty;
        }

        foreach (var item in allowedItems)
        {
            if (string.Equals(item.Trim(), requestOrigin, StringComparison.OrdinalIgnoreCase))
            {
                return requestOrigin;
            }
        }

        return string.Empty;
    }

    private static int ResolveHttpsPort(string? raw)
    {
        if (!string.IsNullOrWhiteSpace(raw) && int.TryParse(raw, out var parsed) && parsed > 0 && parsed <= 65535)
        {
            return parsed;
        }

        return 443;
    }

    private static string NormalizeIp(IPAddress? ip)
    {
        if (ip == null)
        {
            return string.Empty;
        }

        if (ip.IsIPv4MappedToIPv6)
        {
            return ip.MapToIPv4().ToString();
        }

        return ip.ToString();
    }

    private static bool IsRegionBlocked(HttpContext context, IReadOnlyList<string>? blockedCountries, out string countryCode)
    {
        countryCode = ResolveCountryCode(context);
        if (string.IsNullOrWhiteSpace(countryCode) || blockedCountries == null || blockedCountries.Count == 0)
        {
            return false;
        }

        foreach (var raw in blockedCountries)
        {
            var country = raw?.Trim();
            if (string.IsNullOrWhiteSpace(country))
            {
                continue;
            }

            if (string.Equals(country, countryCode, StringComparison.OrdinalIgnoreCase))
            {
                return true;
            }
        }

        return false;
    }

    private static string ResolveCountryCode(HttpContext context)
    {
        if (context.Request.Headers.TryGetValue("CF-IPCountry", out var cf))
        {
            var value = cf.ToString().Trim();
            if (value.Length > 0)
            {
                return value.ToUpperInvariant();
            }
        }

        if (context.Request.Headers.TryGetValue("X-Country-Code", out var xCountry))
        {
            var value = xCountry.ToString().Trim();
            if (value.Length > 0)
            {
                return value.ToUpperInvariant();
            }
        }

        if (context.Request.Headers.TryGetValue("X-Geo-Country", out var geo))
        {
            var value = geo.ToString().Trim();
            if (value.Length > 0)
            {
                return value.ToUpperInvariant();
            }
        }

        return string.Empty;
    }

    private static string ExtractHost(string? url)
    {
        if (string.IsNullOrWhiteSpace(url))
        {
            return string.Empty;
        }

        if (!Uri.TryCreate(url.Trim(), UriKind.Absolute, out var uri))
        {
            return string.Empty;
        }

        return NormalizeHost(uri.Host);
    }

    private static string NormalizeHost(string? host)
    {
        if (string.IsNullOrWhiteSpace(host))
        {
            return string.Empty;
        }

        return host.Trim().TrimEnd('.').ToLowerInvariant();
    }

    private static LogEvent BuildSecurityEvent(
        string traceId,
        string eventName,
        string? host,
        string clientIp,
        PathString path,
        IReadOnlyDictionary<string, object?> fields)
    {
        var payload = new Dictionary<string, object?>(StringComparer.OrdinalIgnoreCase)
        {
            ["host"] = host,
            ["client_ip"] = clientIp,
            ["path"] = path.ToString()
        };

        foreach (var (key, value) in fields)
        {
            payload[key] = value;
        }

        return new LogEvent(
            DateTimeOffset.UtcNow,
            LogChannels.Security,
            "warning",
            eventName,
            traceId,
            DebugLogSanitizer.Sanitize(payload));
    }
}
