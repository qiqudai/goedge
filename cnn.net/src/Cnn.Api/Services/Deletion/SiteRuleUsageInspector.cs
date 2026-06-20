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

    public static Task<IReadOnlyList<CcRuleReferenceUsage>> FindCcMatcherUsagesAsync(
        ISqlSugarClient db,
        long matcherId,
        CancellationToken cancellationToken)
    {
        return FindCcRuleReferencesAsync(
            db,
            matcherId,
            entry => EntryReferencesId(entry, matcherId, "matcher_id", "matcher"));
    }

    public static Task<IReadOnlyList<CcRuleReferenceUsage>> FindCcFilterUsagesAsync(
        ISqlSugarClient db,
        long filterId,
        CancellationToken cancellationToken)
    {
        return FindCcRuleReferencesAsync(
            db,
            filterId,
            entry => EntryReferencesId(entry, filterId, "filter1_id", "filter1", "filter2_id", "filter2", "filter_id"));
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

    private static async Task<IReadOnlyList<CcRuleReferenceUsage>> FindCcRuleReferencesAsync(
        ISqlSugarClient db,
        long targetId,
        Func<JsonElement, bool> entryMatcher)
    {
        if (targetId <= 0)
        {
            return Array.Empty<CcRuleReferenceUsage>();
        }

        var groups = await db.Queryable<CcRule>()
            .ToListAsync();
        if (groups.Count == 0)
        {
            return Array.Empty<CcRuleReferenceUsage>();
        }

        var userIds = groups
            .Where(group => group.Uid is > 0)
            .Select(group => group.Uid!.Value)
            .Distinct()
            .ToList();

        var userNames = new Dictionary<int, string>();
        if (userIds.Count > 0)
        {
            var users = await db.Queryable<User>()
                .Where(user => userIds.Contains(user.Id))
                .Select(user => new { user.Id, user.Name })
                .ToListAsync();

            foreach (var user in users)
            {
                userNames[user.Id] = user.Name ?? string.Empty;
            }
        }

        var usages = new List<CcRuleReferenceUsage>();
        foreach (var group in groups)
        {
            if (!ContainsCcRuleReference(group.Data, entryMatcher))
            {
                continue;
            }

            var ownerName = group.Uid is > 0 && userNames.TryGetValue(group.Uid.Value, out var username)
                ? username
                : string.Empty;
            usages.Add(new CcRuleReferenceUsage
            {
                Id = group.Id,
                Name = group.Name ?? string.Empty,
                Username = ownerName
            });
        }

        return usages.OrderBy(x => x.Id).ToList();
    }

    private static bool ContainsCcRuleReference(string? raw, Func<JsonElement, bool> entryMatcher)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return false;
        }

        try
        {
            using var doc = JsonDocument.Parse(raw);
            return EnumerateCcRuleEntries(doc.RootElement).Any(entryMatcher);
        }
        catch
        {
            return false;
        }
    }

    private static IEnumerable<JsonElement> EnumerateCcRuleEntries(JsonElement root)
    {
        if (root.ValueKind == JsonValueKind.Array)
        {
            foreach (var item in root.EnumerateArray())
            {
                if (item.ValueKind == JsonValueKind.Object)
                {
                    yield return item;
                }
            }

            yield break;
        }

        if (root.ValueKind == JsonValueKind.Object
            && root.TryGetProperty("rules", out var rules)
            && rules.ValueKind == JsonValueKind.Array)
        {
            foreach (var item in rules.EnumerateArray())
            {
                if (item.ValueKind == JsonValueKind.Object)
                {
                    yield return item;
                }
            }
        }
    }

    private static bool EntryReferencesId(JsonElement entry, long targetId, params string[] keys)
    {
        foreach (var key in keys)
        {
            if (!entry.TryGetProperty(key, out var value))
            {
                continue;
            }

            switch (value.ValueKind)
            {
                case JsonValueKind.Number when value.TryGetInt64(out var number) && number == targetId:
                    return true;
                case JsonValueKind.String when long.TryParse(value.GetString(), out var parsed) && parsed == targetId:
                    return true;
            }
        }

        return false;
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

    internal sealed class CcRuleReferenceUsage
    {
        public long Id { get; init; }
        public string Name { get; init; } = string.Empty;
        public string Username { get; init; } = string.Empty;
    }
}
