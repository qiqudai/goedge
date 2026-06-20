using System.Text.Json;
using System.Text.RegularExpressions;
using System.Threading;
using Cnn.Agent.Config;
using Cnn.Agent.Plugin;
using Cnn.Common.Contracts.Agent;
using Cnn.Common.Contracts.Admin;

namespace Cnn.Agent.Security;

public interface ISecurityDecisionService
{
    SecurityDecision Evaluate(HttpContext context, EdgeDomainDto? domain);
}

public sealed class SecurityDecisionService : ISecurityDecisionService
{
    private readonly EdgeConfigStore _edgeConfigStore;
    private readonly WafMatcher _wafMatcher;
    private readonly CcEngine _ccEngine;
    private readonly IPluginHost _pluginHost;
    private readonly object _lock = new();
    private EdgeConfigDto? _lastConfigRef;
    private SecurityConfigSnapshot _current = SecurityConfigSnapshot.Empty;

    public SecurityDecisionService(EdgeConfigStore edgeConfigStore, WafMatcher wafMatcher, CcEngine ccEngine, IPluginHost pluginHost)
    {
        _edgeConfigStore = edgeConfigStore;
        _wafMatcher = wafMatcher;
        _ccEngine = ccEngine;
        _pluginHost = pluginHost;
    }

    public SecurityDecision Evaluate(HttpContext context, EdgeDomainDto? domain)
    {
        if (domain == null)
        {
            return SecurityDecision.Allow();
        }

        var snapshot = GetSnapshot();

        if (_wafMatcher.TryEvaluate(context, snapshot.Waf, out var wafDecision))
        {
            return wafDecision;
        }

        if (_ccEngine.TryEvaluate(context, domain, snapshot, out var ccDecision))
        {
            return ccDecision;
        }

        try
        {
            var pluginDecision = _pluginHost.Evaluate(context);
            if (pluginDecision is { Handled: true })
            {
                if (pluginDecision.Allowed)
                {
                    return SecurityDecision.Allow();
                }

                return SecurityDecision.Block(
                    statusCode: pluginDecision.StatusCode,
                    ruleType: "plugin",
                    ruleId: null,
                    reason: pluginDecision.Reason);
            }
        }
        catch
        {
            // plugin errors must not break main request path
        }

        return SecurityDecision.Allow();
    }

    private SecurityConfigSnapshot GetSnapshot()
    {
        var config = _edgeConfigStore.Current;
        if (config == null)
        {
            return SecurityConfigSnapshot.Empty;
        }

        if (ReferenceEquals(config, _lastConfigRef))
        {
            return Volatile.Read(ref _current);
        }

        lock (_lock)
        {
            if (ReferenceEquals(config, _lastConfigRef))
            {
                return _current;
            }

            var next = BuildSnapshot(config);
            Volatile.Write(ref _current, next);
            _lastConfigRef = config;
            return next;
        }
    }

    private static SecurityConfigSnapshot BuildSnapshot(EdgeConfigDto config)
    {
        var waf = CompileWaf(config.Waf);
        var ccRules = CompileCcRules(config);
        return new SecurityConfigSnapshot(config.Version, waf, ccRules);
    }

    private static WafCompiledConfig CompileWaf(WafConfigDto? waf)
    {
        if (waf == null || !waf.Enable)
        {
            return WafCompiledConfig.Disabled;
        }

        var whiteIps = MergeLists(SplitToList(waf.WhitelistIps), waf.AccessControl?.WhiteIp);
        var blackIps = MergeLists(SplitToList(waf.BlacklistIps), waf.AccessControl?.BlackIp);
        var regionBlock = ToList(waf.AccessControl?.RegionBlock);
        var whiteUa = ToList(waf.AccessControl?.WhiteUa);
        var blackUa = ToList(waf.AccessControl?.BlackUa);
        var whiteUrl = ToList(waf.AccessControl?.WhiteUrl);
        var blackUrl = ToList(waf.AccessControl?.BlackUrl);
        var syntactic = waf.Syntactic;

        return new WafCompiledConfig(
            Enabled: true,
            SqlInjectionEnabled: syntactic?.SqlInjection ?? false,
            XssEnabled: syntactic?.Xss ?? false,
            ScannerEnabled: syntactic?.Scanner ?? false,
            BlockEmptyUa: waf.AccessControl?.BlockEmptyUa ?? false,
            RegionBlockCountries: regionBlock,
            WhiteIps: whiteIps,
            BlackIps: blackIps,
            WhiteUaKeywords: whiteUa,
            BlackUaKeywords: blackUa,
            WhiteUrlKeywords: whiteUrl,
            BlackUrlKeywords: blackUrl);
    }

    private static IReadOnlyDictionary<long, IReadOnlyList<CcRuleCompiled>> CompileCcRules(EdgeConfigDto config)
    {
        var ruleGroups = new Dictionary<long, IReadOnlyList<CcRuleCompiled>>();
        if (config.CcRules == null || config.CcRules.Count == 0)
        {
            return ruleGroups;
        }

        foreach (var (groupId, items) in config.CcRules)
        {
            if (items == null || items.Count == 0)
            {
                continue;
            }

            var compiled = new List<CcRuleCompiled>(items.Count);
            for (var i = 0; i < items.Count; i++)
            {
                var item = items[i];
                var matcher = CompileMatcher(item.MatcherId, config.CcMatchers);
                var filter = CompileFilter(item.FilterId, config.CcFilters);
                var fallbackAction = filter?.Type;
                var action = string.IsNullOrWhiteSpace(item.Action) ? (fallbackAction ?? "block") : item.Action!.Trim().ToLowerInvariant();

                compiled.Add(new CcRuleCompiled
                {
                    RuleId = $"{groupId}:{i}",
                    Enabled = item.Enabled,
                    Action = action,
                    Matcher = matcher,
                    Filter = filter
                });
            }

            ruleGroups[groupId] = compiled;
        }

        return ruleGroups;
    }

    private static CcMatcherCompiled CompileMatcher(long? matcherId, Dictionary<long, EdgeCCMatcherDto>? source)
    {
        var id = matcherId.GetValueOrDefault();
        if (id <= 0 || source == null || !source.TryGetValue(id, out var matcher))
        {
            return CcMatcherCompiled.MatchAll;
        }

        if (string.IsNullOrWhiteSpace(matcher.Data))
        {
            return CcMatcherCompiled.MatchAll;
        }

        try
        {
            using var doc = JsonDocument.Parse(matcher.Data);
            var clauses = new List<CcMatcherClause>();

            var root = doc.RootElement;
            if (root.ValueKind == JsonValueKind.Object && TryGetProperty(root, "rules", out var rulesElement))
            {
                ExtractMatcherClauses(rulesElement, clauses);
            }
            else
            {
                ExtractMatcherClauses(root, clauses);
            }

            if (clauses.Count == 0)
            {
                return CcMatcherCompiled.MatchAll;
            }

            return new CcMatcherCompiled(clauses);
        }
        catch
        {
            return CcMatcherCompiled.MatchAll;
        }
    }

    private static void ExtractMatcherClauses(JsonElement element, List<CcMatcherClause> clauses)
    {
        if (element.ValueKind == JsonValueKind.Array)
        {
            foreach (var item in element.EnumerateArray())
            {
                ExtractMatcherClauses(item, clauses);
            }
            return;
        }

        if (element.ValueKind != JsonValueKind.Object)
        {
            return;
        }

        if (TryGetProperty(element, "conditions", out var conditions) && conditions.ValueKind == JsonValueKind.Array)
        {
            foreach (var condition in conditions.EnumerateArray())
            {
                if (TryBuildClause(condition, out var clause))
                {
                    clauses.Add(clause);
                }
            }
            return;
        }

        if (TryBuildClause(element, out var directClause))
        {
            clauses.Add(directClause);
        }
    }

    private static bool TryBuildClause(JsonElement element, out CcMatcherClause clause)
    {
        clause = new CcMatcherClause();
        if (element.ValueKind != JsonValueKind.Object)
        {
            return false;
        }

        var item = GetString(element, "item", "key", "type", "field");
        var op = GetString(element, "operator", "op", "match");
        var value = GetString(element, "value", "val", "pattern");
        var header = GetString(element, "header", "header_name", "name");

        if (string.IsNullOrWhiteSpace(item) && !string.IsNullOrWhiteSpace(header))
        {
            item = "header";
        }

        if (string.IsNullOrWhiteSpace(item))
        {
            return false;
        }

        op = string.IsNullOrWhiteSpace(op) ? "contains" : op.Trim().ToLowerInvariant();
        value = value?.Trim() ?? string.Empty;

        Regex? regex = null;
        if (op is "regex" or "re" or "~" && !string.IsNullOrWhiteSpace(value))
        {
            try
            {
                regex = new Regex(value, RegexOptions.IgnoreCase | RegexOptions.Compiled, TimeSpan.FromMilliseconds(50));
            }
            catch
            {
                regex = null;
            }
        }

        clause = new CcMatcherClause
        {
            Item = item.Trim().ToLowerInvariant(),
            Operator = op,
            Value = value,
            HeaderName = string.IsNullOrWhiteSpace(header) ? null : header.Trim(),
            Regex = regex
        };
        return true;
    }

    private static CcFilterCompiled? CompileFilter(long? filterId, Dictionary<long, EdgeCCFilterDto>? source)
    {
        var id = filterId.GetValueOrDefault();
        if (id <= 0 || source == null || !source.TryGetValue(id, out var filter))
        {
            return null;
        }

        var withinSecond = Math.Max(1, filter.WithinSecond);
        var maxReq = Math.Max(0, filter.MaxReq);
        var maxReqPerUri = Math.Max(0, filter.MaxReqPerUri);
        var type = string.IsNullOrWhiteSpace(filter.Type) ? "block" : filter.Type.Trim().ToLowerInvariant();
        return new CcFilterCompiled(type, withinSecond, maxReq, maxReqPerUri);
    }

    private static IReadOnlyList<string> SplitToList(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return Array.Empty<string>();
        }

        var value = raw.Trim();
        if (value.StartsWith("["))
        {
            try
            {
                var json = JsonSerializer.Deserialize<List<string>>(value);
                if (json != null)
                {
                    return NormalizeList(json);
                }
            }
            catch
            {
                // ignore
            }
        }

        return NormalizeList(value.Split([',', ';', '|', ' ', '\n', '\r', '\t'], StringSplitOptions.RemoveEmptyEntries));
    }

    private static IReadOnlyList<string> MergeLists(IReadOnlyList<string> first, IReadOnlyList<string>? second)
    {
        if ((first == null || first.Count == 0) && (second == null || second.Count == 0))
        {
            return Array.Empty<string>();
        }

        var result = new List<string>();
        var seen = new HashSet<string>(StringComparer.OrdinalIgnoreCase);

        if (first != null)
        {
            foreach (var value in first)
            {
                if (!string.IsNullOrWhiteSpace(value) && seen.Add(value))
                {
                    result.Add(value);
                }
            }
        }

        if (second != null)
        {
            foreach (var value in second)
            {
                var normalized = value?.Trim();
                if (!string.IsNullOrWhiteSpace(normalized) && seen.Add(normalized))
                {
                    result.Add(normalized);
                }
            }
        }

        return result;
    }

    private static IReadOnlyList<string> ToList(IReadOnlyList<string>? value)
    {
        if (value == null || value.Count == 0)
        {
            return Array.Empty<string>();
        }

        return NormalizeList(value);
    }

    private static IReadOnlyList<string> NormalizeList(IEnumerable<string> source)
    {
        var result = new List<string>();
        var seen = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        foreach (var entry in source)
        {
            var item = entry?.Trim();
            if (string.IsNullOrWhiteSpace(item))
            {
                continue;
            }

            if (seen.Add(item))
            {
                result.Add(item);
            }
        }

        return result;
    }

    private static bool TryGetProperty(JsonElement element, string name, out JsonElement value)
    {
        foreach (var property in element.EnumerateObject())
        {
            if (string.Equals(property.Name, name, StringComparison.OrdinalIgnoreCase))
            {
                value = property.Value;
                return true;
            }
        }

        value = default;
        return false;
    }

    private static string? GetString(JsonElement element, params string[] keys)
    {
        foreach (var key in keys)
        {
            if (!TryGetProperty(element, key, out var value))
            {
                continue;
            }

            var parsed = ReadString(value);
            if (!string.IsNullOrWhiteSpace(parsed))
            {
                return parsed;
            }
        }

        return null;
    }

    private static string? ReadString(JsonElement value)
    {
        return value.ValueKind switch
        {
            JsonValueKind.String => value.GetString(),
            JsonValueKind.Number => value.ToString(),
            JsonValueKind.True => "true",
            JsonValueKind.False => "false",
            JsonValueKind.Array => string.Join(",", value.EnumerateArray().Select(ReadString).Where(s => !string.IsNullOrWhiteSpace(s))),
            _ => null
        };
    }
}
