using System.Globalization;
using Cnn.Common.Contracts.Agent;

namespace Cnn.Agent.Proxy;

public sealed class ProxyConfigValidator
{
    public ProxyApplyResult Validate(EdgeConfigDto? config)
    {
        if (config == null)
        {
            return ProxyApplyResult.Fail(0, "config is null");
        }

        // Go control-plane may generate signed 64-bit hash versions that can be negative.
        // Version ordering is handled by ConfigVersionTracker; validator should not reject by sign.

        var upstreamMap = new Dictionary<string, EdgeUpstreamDto>(StringComparer.OrdinalIgnoreCase);
        foreach (var upstream in config.Upstreams)
        {
            if (string.IsNullOrWhiteSpace(upstream.Id))
            {
                return ProxyApplyResult.Fail(config.Version, "upstream id is required");
            }

            if (upstream.Targets == null || upstream.Targets.Count == 0)
            {
                return ProxyApplyResult.Fail(config.Version, $"upstream {upstream.Id} has no targets");
            }

            foreach (var target in upstream.Targets)
            {
                if (string.IsNullOrWhiteSpace(target.Addr))
                {
                    return ProxyApplyResult.Fail(config.Version, $"upstream {upstream.Id} has empty target addr");
                }

                var normalized = NormalizeAddress(target.Addr);
                if (!Uri.TryCreate(normalized, UriKind.Absolute, out _))
                {
                    return ProxyApplyResult.Fail(config.Version, $"invalid target addr: {target.Addr}");
                }
            }

            upstreamMap[upstream.Id.Trim()] = upstream;
        }

        foreach (var domain in config.Domains)
        {
            if (string.IsNullOrWhiteSpace(domain.Name))
            {
                return ProxyApplyResult.Fail(config.Version, "domain name is required");
            }

            if (string.IsNullOrWhiteSpace(domain.UpstreamKey))
            {
                return ProxyApplyResult.Fail(config.Version, $"domain {domain.Name} missing upstream_key");
            }

            if (!upstreamMap.ContainsKey(domain.UpstreamKey.Trim()))
            {
                return ProxyApplyResult.Fail(config.Version, $"domain {domain.Name} upstream not found: {domain.UpstreamKey}");
            }

            var runtimeError = ValidateDomainRuntime(domain);
            if (!string.IsNullOrWhiteSpace(runtimeError))
            {
                return ProxyApplyResult.Fail(config.Version, $"domain {domain.Name} {runtimeError}");
            }
        }

        return ProxyApplyResult.Ok(config.Version);
    }

    private static string NormalizeAddress(string addr)
    {
        var trimmed = addr.Trim();
        if (trimmed.Contains("://", StringComparison.Ordinal))
        {
            return trimmed;
        }

        return "http://" + trimmed;
    }

    private static string? ValidateDomainRuntime(EdgeDomainDto domain)
    {
        if (!string.IsNullOrWhiteSpace(domain.OriginProtocol))
        {
            var protocol = domain.OriginProtocol.Trim().ToLowerInvariant();
            // Go control-plane can emit "follow" to inherit origin scheme.
            if (protocol is not ("http" or "https" or "follow"))
            {
                return $"has invalid origin_protocol: {domain.OriginProtocol}";
            }
        }

        if (!IsValidPortOrEmpty(domain.OriginHttpPort))
        {
            return $"has invalid origin_http_port: {domain.OriginHttpPort}";
        }

        if (!IsValidPortOrEmpty(domain.OriginHttpsPort))
        {
            return $"has invalid origin_https_port: {domain.OriginHttpsPort}";
        }

        if (!IsValidDurationOrEmpty(domain.ProxyConnectTimeout))
        {
            return $"has invalid proxy_connect_timeout: {domain.ProxyConnectTimeout}";
        }

        if (!IsValidDurationOrEmpty(domain.ProxyReadTimeout))
        {
            return $"has invalid proxy_read_timeout: {domain.ProxyReadTimeout}";
        }

        if (!IsValidDurationOrEmpty(domain.ProxySendTimeout))
        {
            return $"has invalid proxy_send_timeout: {domain.ProxySendTimeout}";
        }

        if (!IsValidProxyHttpVersionOrEmpty(domain.ProxyHttpVersion))
        {
            return $"has invalid proxy_http_version: {domain.ProxyHttpVersion}";
        }

        if (domain.BodyLimit.HasValue && domain.BodyLimit.Value < 0)
        {
            return $"has invalid body_limit: {domain.BodyLimit}";
        }

        if (domain.LimitRate.HasValue && domain.LimitRate.Value < 0)
        {
            return $"has invalid limit_rate: {domain.LimitRate}";
        }

        if (!IsValidHealthPolicyOrEmpty(domain.UpstreamActiveHealthCheckPolicy, active: true))
        {
            return $"has invalid upstream_active_health_check_policy: {domain.UpstreamActiveHealthCheckPolicy}";
        }

        if (!IsValidHealthPolicyOrEmpty(domain.UpstreamPassiveHealthCheckPolicy, active: false))
        {
            return $"has invalid upstream_passive_health_check_policy: {domain.UpstreamPassiveHealthCheckPolicy}";
        }

        if (!IsValidAvailableDestinationsPolicyOrEmpty(domain.UpstreamAvailableDestinationsPolicy))
        {
            return $"has invalid upstream_available_destinations_policy: {domain.UpstreamAvailableDestinationsPolicy}";
        }

        if (!IsValidDurationOrEmpty(domain.UpstreamActiveHealthCheckInterval))
        {
            return $"has invalid upstream_active_health_check_interval: {domain.UpstreamActiveHealthCheckInterval}";
        }

        if (!IsValidDurationOrEmpty(domain.UpstreamActiveHealthCheckTimeout))
        {
            return $"has invalid upstream_active_health_check_timeout: {domain.UpstreamActiveHealthCheckTimeout}";
        }

        if (!IsValidHealthCheckPathOrEmpty(domain.UpstreamActiveHealthCheckPath))
        {
            return $"has invalid upstream_active_health_check_path: {domain.UpstreamActiveHealthCheckPath}";
        }

        if (!IsValidDurationOrEmpty(domain.UpstreamPassiveHealthCheckReactivation))
        {
            return $"has invalid upstream_passive_health_check_reactivation: {domain.UpstreamPassiveHealthCheckReactivation}";
        }

        if (domain.UpstreamActiveHealthCheckThreshold.HasValue && domain.UpstreamActiveHealthCheckThreshold.Value <= 0)
        {
            return $"has invalid upstream_active_health_check_threshold: {domain.UpstreamActiveHealthCheckThreshold}";
        }

        if (domain.UpstreamPassiveHealthCheckRateLimit.HasValue &&
            (domain.UpstreamPassiveHealthCheckRateLimit.Value <= 0 || domain.UpstreamPassiveHealthCheckRateLimit.Value >= 1))
        {
            return $"has invalid upstream_passive_health_check_rate_limit: {domain.UpstreamPassiveHealthCheckRateLimit}";
        }

        return null;
    }

    private static bool IsValidPortOrEmpty(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return true;
        }

        return int.TryParse(raw.Trim(), NumberStyles.Integer, CultureInfo.InvariantCulture, out var port)
               && port > 0
               && port <= 65535;
    }

    private static bool IsValidDurationOrEmpty(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return true;
        }

        var value = raw.Trim().ToLowerInvariant();
        if (value.EndsWith("ms", StringComparison.Ordinal))
        {
            return double.TryParse(value[..^2], NumberStyles.Float, CultureInfo.InvariantCulture, out var ms) && ms > 0;
        }

        if (value.EndsWith("s", StringComparison.Ordinal))
        {
            return double.TryParse(value[..^1], NumberStyles.Float, CultureInfo.InvariantCulture, out var sec) && sec > 0;
        }

        if (value.EndsWith("m", StringComparison.Ordinal))
        {
            return double.TryParse(value[..^1], NumberStyles.Float, CultureInfo.InvariantCulture, out var minute) && minute > 0;
        }

        if (value.EndsWith("h", StringComparison.Ordinal))
        {
            return double.TryParse(value[..^1], NumberStyles.Float, CultureInfo.InvariantCulture, out var hour) && hour > 0;
        }

        if (double.TryParse(value, NumberStyles.Float, CultureInfo.InvariantCulture, out var secNumeric) && secNumeric > 0)
        {
            return true;
        }

        return TimeSpan.TryParse(value, CultureInfo.InvariantCulture, out var parsed) && parsed > TimeSpan.Zero;
    }

    private static bool IsValidProxyHttpVersionOrEmpty(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return true;
        }

        var normalized = raw.Trim().ToLowerInvariant();
        return normalized is
            "1"
            or "1.0"
            or "http/1"
            or "http/1.0"
            or "1.1"
            or "http/1.1"
            or "2"
            or "2.0"
            or "http/2"
            or "h2"
            or "3"
            or "3.0"
            or "http/3"
            or "h3";
    }

    private static bool IsValidHealthPolicyOrEmpty(string? raw, bool active)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return true;
        }

        var normalized = raw.Trim().ToLowerInvariant().Replace("_", string.Empty, StringComparison.Ordinal);
        if (active)
        {
            return normalized is "consecutivefailures";
        }

        return normalized is "transportfailurerate";
    }

    private static bool IsValidAvailableDestinationsPolicyOrEmpty(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return true;
        }

        var normalized = raw.Trim().ToLowerInvariant().Replace("_", string.Empty, StringComparison.Ordinal);
        return normalized is "healthyandunknown" or "healthyorpanic";
    }

    private static bool IsValidHealthCheckPathOrEmpty(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return true;
        }

        var value = raw.Trim();
        if (value.Contains("://", StringComparison.Ordinal))
        {
            return false;
        }

        if (value.Contains('#', StringComparison.Ordinal))
        {
            return false;
        }

        foreach (var ch in value)
        {
            if (char.IsWhiteSpace(ch))
            {
                return false;
            }
        }

        if (!value.StartsWith("/", StringComparison.Ordinal))
        {
            value = "/" + value;
        }

        return Uri.TryCreate(value, UriKind.Relative, out var uri) && !uri.IsAbsoluteUri;
    }
}
