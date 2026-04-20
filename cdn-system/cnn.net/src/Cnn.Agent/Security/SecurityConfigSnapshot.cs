using System.Text.RegularExpressions;

namespace Cnn.Agent.Security;

public sealed class SecurityConfigSnapshot
{
    public static readonly SecurityConfigSnapshot Empty = new(
        version: 0,
        waf: WafCompiledConfig.Disabled,
        ccRulesByGroupId: new Dictionary<long, IReadOnlyList<CcRuleCompiled>>());

    public SecurityConfigSnapshot(
        long version,
        WafCompiledConfig waf,
        IReadOnlyDictionary<long, IReadOnlyList<CcRuleCompiled>> ccRulesByGroupId)
    {
        Version = version;
        Waf = waf;
        CcRulesByGroupId = ccRulesByGroupId;
    }

    public long Version { get; }
    public WafCompiledConfig Waf { get; }
    public IReadOnlyDictionary<long, IReadOnlyList<CcRuleCompiled>> CcRulesByGroupId { get; }
}

public sealed record WafCompiledConfig(
    bool Enabled,
    bool SqlInjectionEnabled,
    bool XssEnabled,
    bool ScannerEnabled,
    bool BlockEmptyUa,
    IReadOnlyList<string> RegionBlockCountries,
    IReadOnlyList<string> WhiteIps,
    IReadOnlyList<string> BlackIps,
    IReadOnlyList<string> WhiteUaKeywords,
    IReadOnlyList<string> BlackUaKeywords,
    IReadOnlyList<string> WhiteUrlKeywords,
    IReadOnlyList<string> BlackUrlKeywords)
{
    public static readonly WafCompiledConfig Disabled = new(
        Enabled: false,
        SqlInjectionEnabled: false,
        XssEnabled: false,
        ScannerEnabled: false,
        BlockEmptyUa: false,
        RegionBlockCountries: Array.Empty<string>(),
        WhiteIps: Array.Empty<string>(),
        BlackIps: Array.Empty<string>(),
        WhiteUaKeywords: Array.Empty<string>(),
        BlackUaKeywords: Array.Empty<string>(),
        WhiteUrlKeywords: Array.Empty<string>(),
        BlackUrlKeywords: Array.Empty<string>());
}

public sealed class CcRuleCompiled
{
    public string RuleId { get; init; } = string.Empty;
    public bool Enabled { get; init; } = true;
    public string Action { get; init; } = "block";
    public CcMatcherCompiled Matcher { get; init; } = CcMatcherCompiled.MatchAll;
    public CcFilterCompiled? Filter { get; init; }
}

public sealed class CcMatcherCompiled
{
    public static readonly CcMatcherCompiled MatchAll = new(Array.Empty<CcMatcherClause>());

    public CcMatcherCompiled(IReadOnlyList<CcMatcherClause> clauses)
    {
        Clauses = clauses;
    }

    public IReadOnlyList<CcMatcherClause> Clauses { get; }
}

public sealed class CcMatcherClause
{
    public string Item { get; init; } = string.Empty;
    public string Operator { get; init; } = "contains";
    public string? HeaderName { get; init; }
    public string Value { get; init; } = string.Empty;
    public Regex? Regex { get; init; }
}

public sealed record CcFilterCompiled(
    string Type,
    int WithinSecond,
    int MaxReq,
    int MaxReqPerUri);
