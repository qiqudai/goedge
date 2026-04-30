using System.Text;
using System.Text.Json;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Agent;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Admin;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Agent;

public interface IEdgeConfigService
{
    Task<ServiceResult<EdgeConfigDto>> GenerateAsync(string nodeId, CancellationToken cancellationToken);
}

public sealed partial class EdgeConfigService : IEdgeConfigService
{
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true
    };

    private readonly ISqlSugarClient _db;
    private readonly IGlobalConfigService _globalConfigService;
    private readonly ISystemConfigService _systemConfigService;
    private readonly ICryptoService _cryptoService;
    private readonly ILogger<EdgeConfigService>? _logger;

    public EdgeConfigService(
        ISqlSugarClient db,
        IGlobalConfigService globalConfigService,
        ISystemConfigService systemConfigService,
        ICryptoService cryptoService,
        ILogger<EdgeConfigService>? logger = null)
    {
        _db = db;
        _globalConfigService = globalConfigService;
        _systemConfigService = systemConfigService;
        _cryptoService = cryptoService;
        _logger = logger;
    }

    public async Task<ServiceResult<EdgeConfigDto>> GenerateAsync(string nodeId, CancellationToken cancellationToken)
    {
        nodeId = nodeId?.Trim() ?? string.Empty;
        if (string.IsNullOrWhiteSpace(nodeId))
        {
            return ServiceResult<EdgeConfigDto>.Fail(ErrorCodes.MissingParam);
        }

        var node = await FindNodeAsync(nodeId);
        if (node == null)
        {
            return ServiceResult<EdgeConfigDto>.Fail(ErrorCodes.NotFound);
        }

        var globalResult = await _globalConfigService.GetAsync(cancellationToken);
        if (!globalResult.Success)
        {
            return ServiceResult<EdgeConfigDto>.Fail(globalResult.ErrorCode, globalResult.MessageKey);
        }

        var global = globalResult.Data ?? new GlobalConfigDto();
        var errorPages = NormalizeErrorPages(global.ErrorPages);

        var config = new EdgeConfigDto
        {
            NodeId = node.Id.ToString(),
            NodeLevel = node.Level ?? 0,
            NodeBandwidthLimit = node.BwLimit,
            Domains = new List<EdgeDomainDto>(),
            Upstreams = new List<EdgeUpstreamDto>(),
            Waf = global.Waf,
            Resources = global.Resources,
            ErrorPages = errorPages.Count == 0 ? null : errorPages,
            DefaultConfig = global.DefaultConfig
        };

        var systemCfg = await _systemConfigService.LoadSystemConfigAsync(cancellationToken);
        if (systemCfg.TryGetValue("https_cert", out var cert))
        {
            config.FallbackCertData = cert?.Trim();
        }
        if (systemCfg.TryGetValue("https_key", out var key))
        {
            config.FallbackKeyData = key?.Trim();
        }

        var ccData = await LoadAllCcDataAsync();
        config.CcRules = ccData.Rules;
        config.CcMatchers = ccData.Matchers;
        config.CcFilters = ccData.Filters;

        await PopulateNodeConfigAsync(config, node, systemCfg, cancellationToken);

        config.Version = ComputeVersion(config);
        _logger?.LogInformation(
            "edge config generated node_id={NodeId} version={Version} domains={Domains} upstreams={Upstreams} streams={Streams} fallback_cert={FallbackCert}",
            config.NodeId ?? nodeId,
            config.Version,
            config.Domains?.Count ?? 0,
            config.Upstreams?.Count ?? 0,
            config.Streams?.Count ?? 0,
            !string.IsNullOrWhiteSpace(config.FallbackCertData) && !string.IsNullOrWhiteSpace(config.FallbackKeyData));
        return ServiceResult<EdgeConfigDto>.Ok(config);
    }

    private async Task<Node?> FindNodeAsync(string nodeId)
    {
        if (int.TryParse(nodeId, out var id) && id > 0)
        {
            var byId = await _db.Queryable<Node>().Where(n => n.Id == id).FirstAsync();
            if (byId != null)
            {
                return byId;
            }
        }

        return await _db.Queryable<Node>()
            .Where(n => n.Name == nodeId || n.Host == nodeId || n.Ip == nodeId)
            .FirstAsync();
    }

    private static long ComputeVersion(EdgeConfigDto config)
    {
        var original = config.Version;
        config.Version = 0;
        var json = JsonSerializer.Serialize(config, JsonOptions);
        config.Version = original;

        const ulong offsetBasis = 14695981039346656037;
        const ulong prime = 1099511628211;
        var hash = offsetBasis;
        foreach (var b in Encoding.UTF8.GetBytes(json))
        {
            hash ^= b;
            hash *= prime;
        }

        // Agent validates config version as > 0. Normalize hash into positive non-zero range.
        var version = unchecked((long)(hash & 0x7FFFFFFFFFFFFFFFUL));
        return version == 0 ? 1 : version;
    }

    private sealed record CcData(
        Dictionary<long, List<EdgeCCRuleItemDto>> Rules,
        Dictionary<long, EdgeCCMatcherDto> Matchers,
        Dictionary<long, EdgeCCFilterDto> Filters);

    private async Task<CcData> LoadAllCcDataAsync()
    {
        var rules = await _db.Queryable<CcRule>()
            .Where(r => r.Enable == true)
            .ToListAsync();

        var ccRules = new Dictionary<long, List<EdgeCCRuleItemDto>>();
        foreach (var rule in rules)
        {
            var items = new List<EdgeCCRuleItemDto>();
            foreach (var entry in ParseCcRuleData(rule.Data))
            {
                var matcherId = ParseEntryId(entry, "matcher", "matcher_id");
                var filterId = ParseEntryId(entry, "filter1", "filter1_id", "filter_id");
                var action = ParseEntryString(entry, "action");
                var enabled = ParseEntryBool(entry, true, "state", "is_on", "enabled");
                items.Add(new EdgeCCRuleItemDto
                {
                    MatcherId = matcherId == 0 ? null : matcherId,
                    FilterId = filterId == 0 ? null : filterId,
                    Action = string.IsNullOrWhiteSpace(action) ? null : action,
                    Enabled = enabled
                });
            }

            ccRules[rule.Id] = items;
        }

        var matchers = await _db.Queryable<CcMatch>()
            .Where(m => m.Enable == true)
            .ToListAsync();
        var ccMatchers = new Dictionary<long, EdgeCCMatcherDto>();
        foreach (var matcher in matchers)
        {
            ccMatchers[matcher.Id] = new EdgeCCMatcherDto
            {
                Id = matcher.Id,
                Data = matcher.Data
            };
        }

        var filters = await _db.Queryable<CcFilter>()
            .Where(f => f.Enable == true)
            .ToListAsync();
        var ccFilters = new Dictionary<long, EdgeCCFilterDto>();
        foreach (var filter in filters)
        {
            ccFilters[filter.Id] = new EdgeCCFilterDto
            {
                Id = filter.Id,
                Type = filter.Type,
                WithinSecond = filter.WithinSecond ?? 0,
                MaxReq = filter.MaxReq ?? 0,
                MaxReqPerUri = filter.MaxReqPerUri ?? 0,
                Extra = filter.Extra
            };
        }

        return new CcData(ccRules, ccMatchers, ccFilters);
    }

    private static IEnumerable<Dictionary<string, JsonElement>> ParseCcRuleData(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return Array.Empty<Dictionary<string, JsonElement>>();
        }

        try
        {
            var list = JsonSerializer.Deserialize<List<Dictionary<string, JsonElement>>>(raw, JsonOptions);
            if (list != null)
            {
                return list;
            }
        }
        catch
        {
        }

        try
        {
            var wrapper = JsonSerializer.Deserialize<CcRuleWrapper>(raw, JsonOptions);
            if (wrapper?.Rules != null)
            {
                return wrapper.Rules;
            }
        }
        catch
        {
        }

        return Array.Empty<Dictionary<string, JsonElement>>();
    }

    private static long ParseEntryId(Dictionary<string, JsonElement> entry, params string[] keys)
    {
        foreach (var key in keys)
        {
            if (!TryGetEntry(entry, key, out var value))
            {
                continue;
            }

            var id = ParseId(value);
            if (id <= 0)
            {
                continue;
            }

            return id;
        }

        return 0;
    }

    private static string ParseEntryString(Dictionary<string, JsonElement> entry, string key)
    {
        return TryGetEntry(entry, key, out var value) ? ParseString(value) : string.Empty;
    }

    private static bool ParseEntryBool(Dictionary<string, JsonElement> entry, bool defaultValue, params string[] keys)
    {
        foreach (var key in keys)
        {
            if (TryGetEntry(entry, key, out var value))
            {
                return ParseBool(value, defaultValue);
            }
        }

        return defaultValue;
    }

    private static bool TryGetEntry(Dictionary<string, JsonElement> entry, string key, out JsonElement value)
    {
        if (entry.TryGetValue(key, out value))
        {
            return true;
        }

        foreach (var pair in entry)
        {
            if (string.Equals(pair.Key, key, StringComparison.OrdinalIgnoreCase))
            {
                value = pair.Value;
                return true;
            }
        }

        value = default;
        return false;
    }

    private static long ParseId(JsonElement value)
    {
        switch (value.ValueKind)
        {
            case JsonValueKind.Number:
                if (value.TryGetInt64(out var numeric))
                {
                    return numeric;
                }
                break;
            case JsonValueKind.String:
                if (long.TryParse(value.GetString(), out var parsed))
                {
                    return parsed;
                }
                break;
        }

        return 0;
    }

    private static string ParseString(JsonElement value)
    {
        return value.ValueKind switch
        {
            JsonValueKind.String => value.GetString()?.Trim() ?? string.Empty,
            JsonValueKind.Number => value.ToString(),
            JsonValueKind.True => "true",
            JsonValueKind.False => "false",
            _ => value.ToString()
        };
    }

    private static bool ParseBool(JsonElement value, bool defaultValue)
    {
        switch (value.ValueKind)
        {
            case JsonValueKind.True:
                return true;
            case JsonValueKind.False:
                return false;
            case JsonValueKind.Number:
                if (value.TryGetInt64(out var numeric))
                {
                    return numeric != 0;
                }
                break;
            case JsonValueKind.String:
                var raw = value.GetString();
                if (string.IsNullOrWhiteSpace(raw))
                {
                    return defaultValue;
                }
                var normalized = raw.Trim().ToLowerInvariant();
                return normalized is "1" or "true" or "yes" or "on";
        }

        return defaultValue;
    }

    private static Dictionary<string, string> NormalizeErrorPages(Dictionary<string, string>? pages)
    {
        if (pages == null || pages.Count == 0)
        {
            return new Dictionary<string, string>();
        }

        var normalized = new Dictionary<string, string>();
        void CopyIfPresent(string sourceKey, string targetKey)
        {
            if (pages.TryGetValue(sourceKey, out var value) && !string.IsNullOrWhiteSpace(value))
            {
                normalized[targetKey] = value;
            }
        }

        foreach (var key in new[]
        {
            "400",
            "403",
            "502",
            "504",
            "traffic_limit",
            "site_locked",
            "domain_invalid",
            "conn_limit",
            "timeout",
            "ip"
        })
        {
            CopyIfPresent(key, key);
        }

        var fallbacks = new Dictionary<string, string>
        {
            ["p400"] = "400",
            ["p403"] = "403",
            ["p502"] = "502",
            ["p504"] = "504",
            ["p512"] = "timeout",
            ["p513"] = "traffic_limit",
            ["p514"] = "site_locked",
            ["p515"] = "conn_limit",
            ["access_ip_not_allow"] = "ip",
            ["host_not_found"] = "domain_invalid"
        };

        foreach (var pair in fallbacks)
        {
            if (!normalized.ContainsKey(pair.Value))
            {
                CopyIfPresent(pair.Key, pair.Value);
            }
        }

        return normalized;
    }

    private sealed class CcRuleWrapper
    {
        public List<Dictionary<string, JsonElement>>? Rules { get; set; }
    }
}
