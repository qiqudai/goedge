using System.Net;
using Cnn.Domain.Entities;
using SqlSugar;

namespace Cnn.Api.Services.Common;

public sealed class PlanGroup
{
    public long NodeGroupId { get; init; }
    public long BackupNodeGroup { get; init; }
}

public static class DnsHelper
{
    public static async Task<Dictionary<int, PlanGroup>> LoadPlanGroupMapAsync(ISqlSugarClient db, IReadOnlyList<int> planIds)
    {
        var result = new Dictionary<int, PlanGroup>();
        if (planIds == null || planIds.Count == 0)
        {
            return result;
        }

        var rows = await db.Queryable<Package>()
            .Where(p => planIds.Contains(p.Id))
            .Select(p => new { p.Id, p.NodeGroupId, p.BackupNodeGroup })
            .ToListAsync();

        foreach (var row in rows)
        {
            result[row.Id] = new PlanGroup
            {
                NodeGroupId = row.NodeGroupId ?? 0,
                BackupNodeGroup = row.BackupNodeGroup ?? 0
            };
        }

        return result;
    }

    public static (string Root, string Name) SplitRootDomain(string domain)
    {
        var host = DomainHelper.NormalizeDomainInput(domain);
        if (string.IsNullOrWhiteSpace(host) || IPAddress.TryParse(host, out _))
        {
            return (string.Empty, string.Empty);
        }
        if (host.StartsWith("*.", StringComparison.Ordinal))
        {
            host = host[2..];
        }
        var parts = host.Split('.', StringSplitOptions.RemoveEmptyEntries);
        if (parts.Length < 2)
        {
            return (string.Empty, string.Empty);
        }
        var root = $"{parts[^2]}.{parts[^1]}";
        var name = parts.Length > 2 ? string.Join(".", parts[..^2]) : "@";
        return (root, name);
    }
}
