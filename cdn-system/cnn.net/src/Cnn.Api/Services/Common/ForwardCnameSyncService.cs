using System.Net;
using Cnn.Domain.Entities;
using SqlSugar;
using Stream = Cnn.Domain.Entities.Stream;

namespace Cnn.Api.Services.Common;

public interface IForwardCnameSyncService
{
    Task<bool> SyncAsync(Stream forward, CancellationToken cancellationToken);
}

public sealed class ForwardCnameSyncService : IForwardCnameSyncService
{
    private readonly ISqlSugarClient _db;
    private readonly IDnsSyncService _dnsSyncService;

    public ForwardCnameSyncService(ISqlSugarClient db, IDnsSyncService dnsSyncService)
    {
        _db = db;
        _dnsSyncService = dnsSyncService;
    }

    public async Task<bool> SyncAsync(Stream forward, CancellationToken cancellationToken)
    {
        if (forward == null)
        {
            return true;
        }

        var (domainKey, host) = ResolveForwardCnameTarget(forward);
        if (string.IsNullOrWhiteSpace(domainKey) || string.IsNullOrWhiteSpace(host))
        {
            return true;
        }

        var domain = await _db.Queryable<CnameDomains>()
            .Where(d => d.Domain == domainKey)
            .FirstAsync();
        if (domain == null)
        {
            return false;
        }

        var groupId = forward.NodeGroupId ?? 0;
        if (groupId == 0 && forward.UserPackage is > 0)
        {
            var pkgGroup = await _db.Queryable<UserPackage>()
                .Where(p => p.Id == forward.UserPackage.Value)
                .Select(p => p.NodeGroupId)
                .FirstAsync();
            groupId = pkgGroup ?? 0;
        }

        if (groupId == 0)
        {
            return true;
        }

        var lines = await _db.Queryable<Line>()
            .Where(l => l.NodeGroupId == groupId)
            .Select(l => new { l.LineId, l.LineName })
            .ToListAsync();

        var lineMap = new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase);
        foreach (var line in lines)
        {
            var lineId = line.LineId?.Trim();
            if (string.IsNullOrWhiteSpace(lineId))
            {
                lineId = "default";
            }

            var lineName = line.LineName?.Trim();
            if (string.IsNullOrWhiteSpace(lineName))
            {
                lineName = lineId;
            }

            if (!lineMap.ContainsKey(lineId))
            {
                lineMap[lineId] = lineName;
            }
        }

        if (lineMap.Count == 0)
        {
            return true;
        }

        foreach (var (lineId, lineName) in lineMap)
        {
            var nodeIds = await LoadLineNodeIdsAsync(groupId, lineId);
            var ok = await _dnsSyncService.SyncPackageLineRecordsAsync(domain, host, groupId, lineId, lineName, "resync", nodeIds);
            if (!ok)
            {
                return false;
            }
        }

        return true;
    }

    private async Task<IReadOnlyList<long>> LoadLineNodeIdsAsync(long groupId, string lineId)
    {
        var lines = await _db.Queryable<Line>()
            .Where(l => l.NodeGroupId == groupId && l.LineId == lineId && l.Enable == true)
            .Select(l => new { l.NodeId, l.NodeIpId })
            .ToListAsync();

        var ids = new List<long>();
        var seen = new HashSet<long>();
        foreach (var line in lines)
        {
            var nodeId = line.NodeIpId ?? 0;
            if (nodeId == 0)
            {
                nodeId = line.NodeId ?? 0;
            }

            if (nodeId <= 0 || !seen.Add(nodeId))
            {
                continue;
            }

            ids.Add(nodeId);
        }

        return ids;
    }

    private static (string DomainKey, string Host) ResolveForwardCnameTarget(Stream forward)
    {
        var domainKey = NormalizeDomain(forward.CnameDomain);
        var full = NormalizeDomain(forward.CnameHostname);
        if (string.IsNullOrWhiteSpace(full))
        {
            return (domainKey, string.Empty);
        }

        if (string.IsNullOrWhiteSpace(domainKey))
        {
            var (root, name) = SplitRootDomain(full);
            if (string.IsNullOrWhiteSpace(root) || string.IsNullOrWhiteSpace(name))
            {
                return (string.Empty, string.Empty);
            }

            return (NormalizeDomain(root), name);
        }

        var host = full;
        var suffix = "." + domainKey;
        if (string.Equals(full, domainKey, StringComparison.OrdinalIgnoreCase))
        {
            host = "@";
        }
        else if (full.EndsWith(suffix, StringComparison.OrdinalIgnoreCase))
        {
            host = full[..^suffix.Length];
        }
        else
        {
            var (root, name) = SplitRootDomain(full);
            if (!string.IsNullOrWhiteSpace(root) && !string.IsNullOrWhiteSpace(name))
            {
                return (NormalizeDomain(root), name);
            }
        }

        host = host.TrimEnd('.');
        return (domainKey, host);
    }

    private static (string Root, string Name) SplitRootDomain(string domain)
    {
        var host = NormalizeDomainHost(domain);
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

    private static string NormalizeDomainHost(string? input)
    {
        var host = DomainHelper.NormalizeDomainInput(input);
        if (string.IsNullOrWhiteSpace(host))
        {
            return string.Empty;
        }

        host = host.Trim().ToLowerInvariant();
        return host.TrimEnd('.');
    }

    private static string NormalizeDomain(string? input)
    {
        return NormalizeDomainHost(input);
    }
}
