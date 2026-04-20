using System.Text.Json;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Deletion;

internal static class SiteRuleUsageInspector
{
    public static async Task<IReadOnlyList<RuleSiteUsage>> FindCcRuleUsagesAsync(
        ISqlSugarClient db,
        long ruleId,
        CancellationToken cancellationToken)
    {
        return await FindUsagesAsync(
            db,
            ruleId,
            directSql: """
SELECT
  s.id,
  s.domain AS primary_domain,
  u.name AS username
FROM site s
JOIN `user` u ON u.id = s.uid
WHERE s.cc_default_rule = @ruleId
ORDER BY s.id ASC
""",
            settingsMatcher: raw => TryExtractNestedId(raw, "security", new[] { "default_rule" }, out var currentId) && currentId == ruleId);
    }

    public static async Task<IReadOnlyList<RuleSiteUsage>> FindAclRuleUsagesAsync(
        ISqlSugarClient db,
        long ruleId,
        CancellationToken cancellationToken)
    {
        return await FindUsagesAsync(
            db,
            ruleId,
            directSql: """
SELECT
  s.id,
  s.domain AS primary_domain,
  u.name AS username
FROM site s
JOIN `user` u ON u.id = s.uid
WHERE s.acl = @ruleId
ORDER BY s.id ASC
""",
            settingsMatcher: raw => TryExtractNestedId(raw, "access", new[] { "acl" }, out var currentId) && currentId == ruleId);
    }

    private static async Task<IReadOnlyList<RuleSiteUsage>> FindUsagesAsync(
        ISqlSugarClient db,
        long ruleId,
        string directSql,
        Func<string?, bool> settingsMatcher)
    {
        if (ruleId <= 0)
        {
            return Array.Empty<RuleSiteUsage>();
        }

        var usages = new Dictionary<long, RuleSiteUsage>();

        var directRefs = await db.Ado.SqlQueryAsync<RuleSiteUsage>(directSql, new { ruleId });
        foreach (var item in directRefs)
        {
            usages[item.Id] = item;
        }

        var settingsRows = await db.Queryable<Config>()
            .Where(c => c.Type == "site_settings" && c.ScopeName == "site" && c.ScopeId != null)
            .ToListAsync();

        var siteIds = settingsRows
            .Where(row => row.ScopeId.HasValue && settingsMatcher(row.Value))
            .Select(row => row.ScopeId!.Value)
            .Distinct()
            .ToList();

        if (siteIds.Count > 0)
        {
            var configRefs = await db.Ado.SqlQueryAsync<RuleSiteUsage>(
                $"""
SELECT
  s.id,
  s.domain AS primary_domain,
  u.name AS username
FROM site s
JOIN `user` u ON u.id = s.uid
WHERE s.id IN ({string.Join(',', siteIds)})
ORDER BY s.id ASC
""");

            foreach (var item in configRefs)
            {
                usages[item.Id] = item;
            }
        }

        return usages.Values.OrderBy(x => x.Id).ToList();
    }

    private static bool TryExtractNestedId(string? raw, string sectionName, IReadOnlyList<string> keys, out long id)
    {
        id = 0;
        if (string.IsNullOrWhiteSpace(raw))
        {
            return false;
        }

        try
        {
            using var doc = JsonDocument.Parse(raw);
            if (doc.RootElement.ValueKind != JsonValueKind.Object)
            {
                return false;
            }

            if (!doc.RootElement.TryGetProperty(sectionName, out var section) || section.ValueKind != JsonValueKind.Object)
            {
                return false;
            }

            foreach (var key in keys)
            {
                if (!section.TryGetProperty(key, out var value))
                {
                    continue;
                }

                switch (value.ValueKind)
                {
                    case JsonValueKind.Number when value.TryGetInt64(out id):
                        return id > 0;
                    case JsonValueKind.String when long.TryParse(value.GetString(), out id):
                        return id > 0;
                }
            }
        }
        catch
        {
            return false;
        }

        return false;
    }

    internal sealed class RuleSiteUsage
    {
        public long Id { get; init; }
        public string PrimaryDomain { get; init; } = string.Empty;
        public string Username { get; init; } = string.Empty;
    }
}
