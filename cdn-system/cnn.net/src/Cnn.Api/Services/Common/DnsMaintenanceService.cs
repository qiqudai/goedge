using System.Data;
using System.Net;
using System.Security.Cryptography;
using Cnn.Api.Services.Common.Dns;
using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Common;

public interface IDnsMaintenanceService
{
    Task<IReadOnlyList<string>> RepairRecordsAsync(CancellationToken cancellationToken);
    Task<IReadOnlyList<string>> CleanupInvalidRecordsAsync(CancellationToken cancellationToken);
    Task<IReadOnlyList<string>> ResyncForProviderAsync(long providerId, CancellationToken cancellationToken);
}

public sealed class DnsMaintenanceService : IDnsMaintenanceService
{
    private readonly ISqlSugarClient _db;
    private readonly IDnsSyncService _dnsSyncService;
    private readonly ISystemConfigService _systemConfigService;

    public DnsMaintenanceService(ISqlSugarClient db, IDnsSyncService dnsSyncService, ISystemConfigService systemConfigService)
    {
        _db = db;
        _dnsSyncService = dnsSyncService;
        _systemConfigService = systemConfigService;
    }

    public async Task<IReadOnlyList<string>> RepairRecordsAsync(CancellationToken cancellationToken)
    {
        var groups = await _db.Queryable<NodeGroup>().ToListAsync();
        if (groups.Count == 0)
        {
            return Array.Empty<string>();
        }

        var errors = new List<string>();
        foreach (var group in groups)
        {
            var resyncErrors = await ResyncGroupRecordsAsync(group.Id, cancellationToken);
            if (resyncErrors.Count > 0)
            {
                errors.AddRange(resyncErrors);
            }
        }

        return errors;
    }

    public async Task<IReadOnlyList<string>> ResyncForProviderAsync(long providerId, CancellationToken cancellationToken)
    {
        if (providerId <= 0)
        {
            return Array.Empty<string>();
        }

        var domains = await _db.Queryable<CnameDomains>()
            .Where(d => d.DnsProviderId == providerId)
            .ToListAsync();

        if (domains.Count == 0)
        {
            return Array.Empty<string>();
        }

        var domainKeys = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        foreach (var domain in domains)
        {
            var key = NormalizeDomainName(domain.Domain);
            if (!string.IsNullOrWhiteSpace(key))
            {
                domainKeys.Add(key);
            }
        }

        if (domainKeys.Count == 0)
        {
            return Array.Empty<string>();
        }

        return await ResyncForCnameDomainsAsync(domainKeys.ToList(), cancellationToken);
    }

    public async Task<IReadOnlyList<string>> CleanupInvalidRecordsAsync(CancellationToken cancellationToken)
    {
        var cfg = await _systemConfigService.LoadSystemConfigAsync(cancellationToken);
        cfg.TryGetValue("dns_rs_protect", out var protectRaw);
        var protectedHosts = ParseProtectedHosts(protectRaw ?? string.Empty);
        return await CleanupInvalidRecordsAsync(protectedHosts, cancellationToken);
    }

    private async Task<IReadOnlyList<string>> ResyncForCnameDomainsAsync(IReadOnlyList<string> domains, CancellationToken cancellationToken)
    {
        if (domains == null || domains.Count == 0)
        {
            return Array.Empty<string>();
        }

        var domainSet = new HashSet<string>(domains, StringComparer.OrdinalIgnoreCase);
        if (domainSet.Count == 0)
        {
            return Array.Empty<string>();
        }

        var groups = await _db.Queryable<NodeGroup>()
            .Where(g => g.CnameDomain == null || g.CnameDomain == string.Empty || domainSet.Contains(g.CnameDomain!))
            .ToListAsync();

        if (groups.Count == 0)
        {
            return Array.Empty<string>();
        }

        var groupIds = new HashSet<long>();
        var errors = new List<string>();
        foreach (var group in groups)
        {
            var resolved = await EnsureGroupDnsConfigAsync(group.Id, cancellationToken);
            if (resolved == null)
            {
                errors.Add($"dns group config failed: {group.Id}");
                continue;
            }

            var domainKey = NormalizeDomainName(resolved.CnameDomain);
            if (!string.IsNullOrWhiteSpace(domainKey) && domainSet.Contains(domainKey))
            {
                groupIds.Add(resolved.Id);
            }
        }

        foreach (var groupId in groupIds)
        {
            var resyncErrors = await ResyncGroupRecordsAsync(groupId, cancellationToken);
            if (resyncErrors.Count > 0)
            {
                errors.AddRange(resyncErrors);
            }
        }

        return errors;
    }

    private async Task<IReadOnlyList<string>> ResyncGroupRecordsAsync(long groupId, CancellationToken cancellationToken)
    {
        if (groupId <= 0)
        {
            return Array.Empty<string>();
        }

        var resolved = await EnsureGroupDnsConfigAsync(groupId, cancellationToken);
        if (resolved == null)
        {
            return new[] { $"dns group config failed: {groupId}" };
        }

        var lines = await _db.Queryable<Line>()
            .Where(l => l.NodeGroupId == resolved.Id)
            .Select(l => new { l.LineId, l.LineName, l.NodeId, l.NodeIpId, l.Enable })
            .ToListAsync();

        if (lines.Count == 0)
        {
            return Array.Empty<string>();
        }

        var lineMap = new Dictionary<string, LineBucket>(StringComparer.OrdinalIgnoreCase);
        foreach (var line in lines)
        {
            if (line.Enable != true)
            {
                continue;
            }

            var lineKey = NormalizeLineId(line.LineId);
            var lineName = NormalizeLineName(line.LineName, lineKey);
            if (!lineMap.TryGetValue(lineKey, out var bucket))
            {
                bucket = new LineBucket(lineName);
                lineMap[lineKey] = bucket;
            }

            var nodeId = line.NodeIpId ?? 0;
            if (nodeId == 0)
            {
                nodeId = line.NodeId ?? 0;
            }
            if (nodeId > 0)
            {
                bucket.NodeIds.Add(nodeId);
            }
        }

        var errors = new List<string>();
        foreach (var pair in lineMap)
        {
            var ids = pair.Value.NodeIds.Where(id => id > 0).Distinct().ToList();
            var ok = await _dnsSyncService.SyncLineRecordsAsync(resolved.Id, pair.Key, pair.Value.Name, "resync", ids);
            if (!ok)
            {
                errors.Add($"dns sync failed group={resolved.Id} line={pair.Key}");
            }

            var ok2 = await _dnsSyncService.SyncPackageCnameForLineChangeAsync(resolved.Id, pair.Key, pair.Value.Name, ids, "resync");
            if (!ok2)
            {
                errors.Add($"dns package cname sync failed group={resolved.Id} line={pair.Key}");
            }
        }

        return errors;
    }

    private async Task<IReadOnlyList<string>> CleanupInvalidRecordsAsync(HashSet<string> protectedHosts, CancellationToken cancellationToken)
    {
        var groups = await _db.Queryable<NodeGroup>().ToListAsync();
        if (groups.Count == 0)
        {
            return Array.Empty<string>();
        }

        var errors = new List<string>();
        var allowedAValues = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        var allowedCnameValues = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        var domainSet = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        var groupIds = new HashSet<long>();

        foreach (var group in groups)
        {
            var resolved = await EnsureGroupDnsConfigAsync(group.Id, cancellationToken);
            if (resolved == null)
            {
                errors.Add($"dns group config failed: {group.Id}");
                continue;
            }

            groupIds.Add(resolved.Id);
            var domainKey = NormalizeDomainName(resolved.CnameDomain);
            if (!string.IsNullOrWhiteSpace(domainKey))
            {
                domainSet.Add(domainKey);
            }

            var lineValue = BuildLineCnameValue(domainKey, resolved.CnameHostname ?? string.Empty);
            AddAllowedCnameValue(allowedCnameValues, lineValue);
        }

        if (groupIds.Count > 0)
        {
            var ids = groupIds.ToList();
            var lines = await _db.Queryable<Line>()
                .Where(l => ids.Contains(l.NodeGroupId ?? 0))
                .Select(l => new { l.NodeId, l.NodeIpId, l.Enable })
                .ToListAsync();

            var nodeIds = new HashSet<long>();
            foreach (var line in lines)
            {
                if (line.Enable != true)
                {
                    continue;
                }

                var nodeId = line.NodeIpId ?? 0;
                if (nodeId == 0)
                {
                    nodeId = line.NodeId ?? 0;
                }
                if (nodeId > 0)
                {
                    nodeIds.Add(nodeId);
                }
            }

            if (nodeIds.Count > 0)
            {
                var nodes = await _db.Queryable<Node>()
                    .Where(n => nodeIds.Contains(n.Id))
                    .Select(n => new { n.Id, n.Ip })
                    .ToListAsync();

                foreach (var node in nodes)
                {
                    AddAllowedAValue(allowedAValues, node.Ip);
                }
            }

            var siteDomainKeys = await LoadSiteCnameDomainKeysAsync(ids, cancellationToken);
            foreach (var key in siteDomainKeys)
            {
                if (!string.IsNullOrWhiteSpace(key))
                {
                    domainSet.Add(key);
                }
            }
        }

        var siteColumns = new List<string> { "cname_hostname" };
        if (_db.DbMaintenance.IsAnyColumn("site", "cname_hostname2"))
        {
            siteColumns.Add("cname_hostname2");
        }

        if (siteColumns.Count > 0)
        {
            var table = _db.Ado.GetDataTable($"SELECT {string.Join(",", siteColumns)} FROM site");
            foreach (DataRow row in table.Rows)
            {
                foreach (var col in siteColumns)
                {
                    var value = row[col]?.ToString();
                    AddAllowedCnameValue(allowedCnameValues, value);
                }
            }
        }

        if (domainSet.Count == 0)
        {
            return errors;
        }
        if (allowedAValues.Count == 0 && allowedCnameValues.Count == 0)
        {
            return errors;
        }

        var domainList = domainSet.ToList();
        var domainRows = await _db.Queryable<CnameDomains>()
            .Where(d => domainList.Contains(d.Domain))
            .ToListAsync();

        if (domainRows.Count == 0)
        {
            return errors;
        }

        var apiMap = new Dictionary<long, Dnsapi>();
        foreach (var domain in domainRows)
        {
            if (domain.DnsProviderId <= 0)
            {
                continue;
            }
            if (!apiMap.ContainsKey(domain.DnsProviderId))
            {
                var api = await _db.Queryable<Dnsapi>().Where(p => p.Id == domain.DnsProviderId).FirstAsync();
                if (api != null)
                {
                    apiMap[domain.DnsProviderId] = api;
                }
            }
        }

        foreach (var domain in domainRows)
        {
            if (domain.DnsProviderId <= 0)
            {
                continue;
            }
            if (!apiMap.TryGetValue(domain.DnsProviderId, out var api))
            {
                continue;
            }

            var provider = DnsProviderFactory.TryCreate(api.Type, api.Auth);
            if (provider == null)
            {
                errors.Add("dns provider not available");
                continue;
            }

            IReadOnlyList<DnsRecord> records;
            try
            {
                records = await provider.GetRecordsAsync(domain.Domain);
            }
            catch (Exception ex)
            {
                errors.Add(ex.Message);
                continue;
            }

            foreach (var record in records)
            {
                if (string.Equals(record.Type, "NS", StringComparison.OrdinalIgnoreCase))
                {
                    continue;
                }
                if (!string.Equals(record.Type, "A", StringComparison.OrdinalIgnoreCase) &&
                    !string.Equals(record.Type, "CNAME", StringComparison.OrdinalIgnoreCase))
                {
                    continue;
                }
                if (IsProtectedRecord(record.Name, domain.Domain, protectedHosts))
                {
                    continue;
                }

                if (string.Equals(record.Type, "A", StringComparison.OrdinalIgnoreCase))
                {
                    var value = (record.Value ?? string.Empty).Trim();
                    if (!string.IsNullOrWhiteSpace(value) && allowedAValues.Contains(value))
                    {
                        continue;
                    }
                }
                else if (string.Equals(record.Type, "CNAME", StringComparison.OrdinalIgnoreCase))
                {
                    var value = NormalizeDomainName(record.Value);
                    if (!string.IsNullOrWhiteSpace(value) && allowedCnameValues.Contains(value))
                    {
                        continue;
                    }
                }

                try
                {
                    await provider.DeleteRecordAsync(domain.Domain, record);
                }
                catch (Exception ex)
                {
                    errors.Add(ex.Message);
                }
            }
        }

        return errors;
    }

    private async Task<NodeGroup?> EnsureGroupDnsConfigAsync(long groupId, CancellationToken cancellationToken)
    {
        var group = await _db.Queryable<NodeGroup>().Where(g => g.Id == groupId).FirstAsync();
        if (group == null)
        {
            return null;
        }

        var updates = new Dictionary<string, object?>();
        var domain = NormalizeDomainName(group.CnameDomain);
        if (string.IsNullOrWhiteSpace(domain))
        {
            var fallback = await LoadFirstCnameDomainAsync(cancellationToken);
            if (string.IsNullOrWhiteSpace(fallback))
            {
                return null;
            }
            domain = fallback;
            updates["cname_domain"] = domain;
        }
        else if (!string.Equals(group.CnameDomain?.Trim(), domain, StringComparison.OrdinalIgnoreCase))
        {
            updates["cname_domain"] = domain;
        }
        group.CnameDomain = domain;

        var host = group.CnameHostname?.Trim();
        if (string.IsNullOrWhiteSpace(host))
        {
            var generated = await GenerateUniqueGroupHostnameAsync(cancellationToken);
            if (string.IsNullOrWhiteSpace(generated))
            {
                return null;
            }
            host = generated;
            updates["cname_hostname"] = host;
        }
        group.CnameHostname = host;

        if (updates.Count > 0)
        {
            var now = DateTime.Now;
            await _db.Updateable<NodeGroup>()
                .SetColumns(g => new NodeGroup
                {
                    CnameDomain = domain,
                    CnameHostname = host,
                    UpdateAt = now
                })
                .Where(g => g.Id == groupId)
                .ExecuteCommandAsync();
        }

        return group;
    }

    private async Task<string> LoadFirstCnameDomainAsync(CancellationToken cancellationToken)
    {
        var row = await _db.Queryable<CnameDomains>().OrderBy(d => d.Id).FirstAsync();
        if (row == null)
        {
            return string.Empty;
        }
        return NormalizeDomainName(row.Domain);
    }

    private async Task<string> GenerateUniqueGroupHostnameAsync(CancellationToken cancellationToken)
    {
        for (var i = 0; i < 5; i++)
        {
            var token = GenerateToken(8);
            var exists = await _db.Queryable<NodeGroup>().Where(g => g.CnameHostname == token).AnyAsync();
            if (!exists)
            {
                return token;
            }
        }

        return string.Empty;
    }

    private static string GenerateToken(int length)
    {
        const string chars = "abcdefghijklmnopqrstuvwxyz0123456789";
        var buffer = new byte[length];
        RandomNumberGenerator.Fill(buffer);
        var output = new char[length];
        for (var i = 0; i < length; i++)
        {
            output[i] = chars[buffer[i] % chars.Length];
        }
        return new string(output);
    }

    private async Task<HashSet<string>> LoadSiteCnameDomainKeysAsync(IReadOnlyList<long> groupIds, CancellationToken cancellationToken)
    {
        var result = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        if (groupIds == null || groupIds.Count == 0)
        {
            return result;
        }

        var groups = groupIds.Where(id => id > 0).Distinct().ToList();
        if (groups.Count == 0)
        {
            return result;
        }

        var packs = await _db.Queryable<UserPackage>()
            .Where(p => groups.Contains(p.NodeGroupId ?? 0) || groups.Contains(p.BackupNodeGroup ?? 0))
            .ToListAsync();

        var packMap = packs.ToDictionary(p => (long)p.Id, p => p);
        var packIds = packMap.Keys.ToList();

        var cond = Expressionable.Create<Site>();
        cond.Or(s => groups.Contains(s.NodeGroupId ?? 0) || groups.Contains(s.BackupNodeGroup ?? 0));
        if (packIds.Count > 0)
        {
            cond.Or(s => packIds.Contains(s.UserPackage ?? 0));
        }

        var sites = await _db.Queryable<Site>().Where(cond.ToExpression()).ToListAsync();
        if (sites.Count == 0)
        {
            return result;
        }

        var missingPackIds = sites.Select(s => s.UserPackage ?? 0)
            .Where(id => id > 0 && !packMap.ContainsKey(id))
            .Distinct()
            .ToList();

        if (missingPackIds.Count > 0)
        {
            var extra = await _db.Queryable<UserPackage>().Where(p => missingPackIds.Contains(p.Id)).ToListAsync();
            foreach (var pack in extra)
            {
                packMap[pack.Id] = pack;
            }
        }

        var planIds = packMap.Values.Select(p => p.Package ?? 0).Where(id => id > 0).Distinct().ToList();
        var planGroupMap = await DnsHelper.LoadPlanGroupMapAsync(_db, planIds);

        foreach (var site in sites)
        {
            if (site.UserPackage is null or <= 0 || !packMap.TryGetValue(site.UserPackage.Value, out var pkg))
            {
                continue;
            }

            var (domainKey, _) = ResolveSiteCnameTarget(site, pkg);
            if (string.IsNullOrWhiteSpace(domainKey))
            {
                continue;
            }

            var planGroup = planGroupMap.TryGetValue(pkg.Package ?? 0, out var p) ? p : new PlanGroup();
            _ = ResolveSiteGroups(site, pkg, planGroup);

            result.Add(domainKey);
        }

        return result;
    }

    private static (long Primary, long Backup, bool EnableBackup) ResolveSiteGroups(
        Site site,
        UserPackage pkg,
        PlanGroup planGroup)
    {
        long primary = site.NodeGroupId ?? 0;
        if (primary == 0)
        {
            primary = pkg.NodeGroupId ?? 0;
        }
        if (primary == 0)
        {
            primary = planGroup.NodeGroupId;
        }

        var enableBackup = site.EnableBackupGroup ?? false;
        if (!enableBackup)
        {
            enableBackup = pkg.EnableBackupGroup ?? false;
        }

        var backup = 0L;
        if (enableBackup)
        {
            backup = site.BackupNodeGroup ?? 0;
            if (backup == 0)
            {
                backup = pkg.BackupNodeGroup ?? 0;
            }
            if (backup == 0)
            {
                backup = planGroup.BackupNodeGroup;
            }
        }

        return (primary, backup, enableBackup);
    }

    private static (string DomainKey, string Host) ResolveSiteCnameTarget(Site site, UserPackage pkg)
    {
        var siteMode = (site.CnameMode ?? string.Empty).Trim();
        var pkgMode = (pkg.CnameMode ?? string.Empty).Trim();

        if (string.Equals(siteMode, "package", StringComparison.OrdinalIgnoreCase) ||
            (string.IsNullOrWhiteSpace(siteMode) && string.Equals(pkgMode, "package", StringComparison.OrdinalIgnoreCase)))
        {
            var domainKey = NormalizeDomainName(pkg.CnameDomain);
            var host = NormalizeDomainName(pkg.CnameHostname);
            if (string.IsNullOrWhiteSpace(host))
            {
                host = NormalizeDomainName(pkg.RecordId);
            }
            if (string.IsNullOrWhiteSpace(host))
            {
                return (domainKey, string.Empty);
            }
            if (string.IsNullOrWhiteSpace(domainKey))
            {
                var (root, name) = DnsHelper.SplitRootDomain(host);
                return (NormalizeDomainName(root), name);
            }

            var suffix = "." + domainKey;
            if (string.Equals(host, domainKey, StringComparison.OrdinalIgnoreCase))
            {
                host = "@";
            }
            else if (host.EndsWith(suffix, StringComparison.OrdinalIgnoreCase))
            {
                host = host[..^suffix.Length];
            }

            return (domainKey, host.TrimEnd('.'));
        }

        var domainKey2 = NormalizeDomainName(site.CnameDomain);
        if (string.IsNullOrWhiteSpace(domainKey2))
        {
            domainKey2 = NormalizeDomainName(pkg.CnameDomain);
        }

        var full = NormalizeDomainName(site.CnameHostname);
        if (string.IsNullOrWhiteSpace(full))
        {
            var domains = DomainParser.ParseDomains(site.Domain);
            if (domains.Count > 0 && !string.IsNullOrWhiteSpace(domainKey2))
            {
                full = NormalizeDomainName(domains[0] + "." + domainKey2);
            }
        }

        if (string.IsNullOrWhiteSpace(full))
        {
            return (domainKey2, string.Empty);
        }

        if (string.IsNullOrWhiteSpace(domainKey2))
        {
            var (root, name) = DnsHelper.SplitRootDomain(full);
            return (NormalizeDomainName(root), name);
        }

        var host2 = full;
        var suffix2 = "." + domainKey2;
        if (string.Equals(full, domainKey2, StringComparison.OrdinalIgnoreCase))
        {
            host2 = "@";
        }
        else if (full.EndsWith(suffix2, StringComparison.OrdinalIgnoreCase))
        {
            host2 = full[..^suffix2.Length];
        }

        return (domainKey2, host2.TrimEnd('.'));
    }





    private static string NormalizeLineId(string? lineId)
    {
        var id = (lineId ?? string.Empty).Trim();
        return string.IsNullOrWhiteSpace(id) ? "default" : id;
    }

    private static string NormalizeLineName(string? lineName, string? fallback)
    {
        var name = (lineName ?? string.Empty).Trim();
        if (string.IsNullOrWhiteSpace(name))
        {
            name = (fallback ?? string.Empty).Trim();
        }
        return string.IsNullOrWhiteSpace(name) ? "default" : name;
    }

    private static string NormalizeDomainName(string? input)
    {
        return DomainHelper.NormalizeDomainInput(input);
    }

    private static string BuildLineCnameValue(string domainKey, string host)
    {
        domainKey = NormalizeDomainName(domainKey);
        if (string.IsNullOrWhiteSpace(domainKey))
        {
            return string.Empty;
        }
        host = NormalizeDomainName(host);
        if (string.IsNullOrWhiteSpace(host))
        {
            return string.Empty;
        }

        var recordHost = NormalizeRecordHost(host, domainKey);
        if (string.IsNullOrWhiteSpace(recordHost))
        {
            return string.Empty;
        }
        if (recordHost == "@")
        {
            return domainKey;
        }
        return recordHost + "." + domainKey;
    }

    private static void AddAllowedAValue(HashSet<string> values, string? value)
    {
        value = (value ?? string.Empty).Trim();
        if (string.IsNullOrWhiteSpace(value))
        {
            return;
        }
        values.Add(value);
    }

    private static void AddAllowedCnameValue(HashSet<string> values, string? value)
    {
        value = NormalizeDomainName(value);
        if (string.IsNullOrWhiteSpace(value))
        {
            return;
        }
        values.Add(value);
    }

    private static HashSet<string> ParseProtectedHosts(string raw)
    {
        var result = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        if (string.IsNullOrWhiteSpace(raw))
        {
            return result;
        }

        var parts = raw.Split(new[] { ',', ';', ' ', '\n', '\r', '\t' }, StringSplitOptions.RemoveEmptyEntries);
        foreach (var part in parts)
        {
            var host = part.Trim().ToLowerInvariant().TrimEnd('.');
            if (!string.IsNullOrWhiteSpace(host))
            {
                result.Add(host);
            }
        }

        return result;
    }

    private static bool IsProtectedRecord(string? recordName, string domain, HashSet<string> protectedHosts)
    {
        if (protectedHosts.Count == 0)
        {
            return false;
        }

        var name = NormalizeRecordName(recordName);
        if (!string.IsNullOrWhiteSpace(name) && protectedHosts.Contains(name))
        {
            return true;
        }

        var host = NormalizeRecordHost(recordName, domain);
        if (!string.IsNullOrWhiteSpace(host) && protectedHosts.Contains(host))
        {
            return true;
        }

        var domainKey = NormalizeDomainName(domain);
        if (string.IsNullOrWhiteSpace(domainKey))
        {
            return false;
        }

        if (host == "@" && protectedHosts.Contains(domainKey))
        {
            return true;
        }

        if (!string.IsNullOrWhiteSpace(host) && host != "@")
        {
            var fqdn = host + "." + domainKey;
            if (protectedHosts.Contains(fqdn))
            {
                return true;
            }
        }

        return false;
    }

    private static string NormalizeRecordHost(string? recordName, string domain)
    {
        var host = NormalizeRecordName(recordName);
        if (string.IsNullOrWhiteSpace(host))
        {
            return string.Empty;
        }

        var domainKey = NormalizeDomainName(domain);
        if (string.IsNullOrWhiteSpace(domainKey))
        {
            return host;
        }

        if (host == domainKey)
        {
            return "@";
        }

        var suffix = "." + domainKey;
        if (host.EndsWith(suffix, StringComparison.OrdinalIgnoreCase))
        {
            host = host[..^suffix.Length];
        }

        return host.TrimEnd('.');
    }

    private static string NormalizeRecordName(string? input)
    {
        var name = (input ?? string.Empty).Trim().ToLowerInvariant();
        return name.TrimEnd('.');
    }



    private sealed class LineBucket
    {
        public LineBucket(string name)
        {
            Name = name;
        }

        public string Name { get; }
        public List<long> NodeIds { get; } = new();
    }
}
