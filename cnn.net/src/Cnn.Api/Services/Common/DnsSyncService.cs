
using System.Net;
using System.Security.Cryptography;
using System.Text.Json;
using Cnn.Api.Services.Common.Dns;
using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Common;

public sealed class DnsSyncService : IDnsSyncService
{
    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true
    };

    private static readonly IReadOnlyDictionary<string, IReadOnlyDictionary<string, string>> LineMaps =
        new Dictionary<string, IReadOnlyDictionary<string, string>>(StringComparer.OrdinalIgnoreCase)
        {
            {
                "aliyun",
                new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase)
                {
                    ["default"] = "default",
                    ["telecom"] = "telecom",
                    ["unicom"] = "unicom",
                    ["mobile"] = "mobile",
                    ["ctt"] = "tieTong",
                    ["broadnet"] = "broadcast",
                    ["edu"] = "edu",
                    ["cn"] = "mainland",
                    ["global"] = "oversea",
                    ["search"] = "search"
                }
            },
            { "dnspod", BuildChinaLineMap() },
            { "dnspod_intl", BuildDnsPodIntlLineMap() },
            { "dnsla", BuildCommonLineMap() },
            { "huawei", BuildCommonLineMap() }
        };

    private readonly ISqlSugarClient _db;
    private readonly IDnsProviderResolver _providerResolver;

    public DnsSyncService(ISqlSugarClient db)
        : this(db, new DnsProviderResolver())
    {
    }

    public DnsSyncService(ISqlSugarClient db, IDnsProviderResolver providerResolver)
    {
        _db = db;
        _providerResolver = providerResolver;
    }

    public async Task<bool> SyncUserDnsRecordsAsync(Site? oldSite, Site? newSite)
    {
        if (oldSite == null && newSite == null)
        {
            return true;
        }

        if (oldSite != null && (newSite == null || (newSite.DnsProviderId ?? 0) != (oldSite.DnsProviderId ?? 0)))
        {
            var oldApi = await ResolveDnsApiForSiteAsync(oldSite);
            if (oldApi != null)
            {
                await DeleteSiteDomainsAsync(oldApi, DecodeDomains(oldSite.Domain), oldSite.CnameHostname);
            }
        }

        if (newSite == null)
        {
            return true;
        }

        var newDomains = DecodeDomains(newSite.Domain);
        if (newDomains.Count == 0 || string.IsNullOrWhiteSpace(newSite.CnameHostname))
        {
            return true;
        }

        var api = await ResolveDnsApiForSiteAsync(newSite);
        if (api == null)
        {
            return false;
        }

        if (oldSite != null && (oldSite.DnsProviderId ?? 0) == (newSite.DnsProviderId ?? 0))
        {
            var removed = DiffDomains(DecodeDomains(oldSite.Domain), newDomains);
            if (removed.Count > 0)
            {
                await DeleteSiteDomainsAsync(api, removed, oldSite.CnameHostname);
            }
        }

        await UpsertSiteDomainsAsync(api, newDomains, newSite.CnameHostname);
        return true;
    }

    public async Task<bool> SyncLineRecordsAsync(long groupId, string lineId, string lineName, string action, IReadOnlyList<long> nodeIds)
    {
        if (groupId <= 0)
        {
            return true;
        }

        var group = await EnsureGroupDnsConfigAsync(groupId);
        if (group == null)
        {
            return false;
        }

        var domainName = NormalizeDomainName(group.CnameDomain);
        if (string.IsNullOrWhiteSpace(domainName))
        {
            return false;
        }

        var recordName = NormalizeRecordHostname(group.CnameHostname, domainName);
        if (string.IsNullOrWhiteSpace(recordName))
        {
            return false;
        }

        var cname = await _db.Queryable<CnameDomains>().Where(d => d.Domain == domainName).FirstAsync();
        if (cname == null || cname.DnsProviderId <= 0)
        {
            return false;
        }

        var api = await _db.Queryable<Dnsapi>().Where(p => p.Id == cname.DnsProviderId).FirstAsync();
        if (api == null)
        {
            return false;
        }

        if (!ValidateDnsPodIntl(api.Type, api.Auth))
        {
            return false;
        }

        var provider = await TryCreateProviderAsync(api);
        if (provider == null)
        {
            return false;
        }

        var ttl = ResolveTtl(api.Auth);
        var lineValue = ResolveLineValue(api.Type, lineId, lineName);

        action = (action ?? string.Empty).Trim().ToLowerInvariant();
        if (action == "enable")
        {
            action = "add";
        }
        if (action == "disable")
        {
            action = "delete";
        }

        var resync = action == "resync";
        if ((nodeIds == null || nodeIds.Count == 0) && !resync)
        {
            return true;
        }

        var record = new DnsRecord
        {
            Type = "A",
            Name = recordName,
            Line = lineValue,
            TTL = ttl
        };

        if (resync)
        {
            var desiredNodeIds = await LoadLineNodeIdsAsync(groupId, lineId);
            if (desiredNodeIds.Count == 0)
            {
                return true;
            }

            var nodes = await _db.Queryable<Node>()
                .Where(n => desiredNodeIds.Contains(n.Id))
                .Select(n => new { n.Id, n.Ip })
                .ToListAsync();

            var weightMap = await LoadLineWeightMapAsync(groupId, lineId, desiredNodeIds);
            var ipWeights = new Dictionary<string, int>(StringComparer.OrdinalIgnoreCase);
            foreach (var node in nodes)
            {
                var ip = node.Ip?.Trim();
                if (string.IsNullOrWhiteSpace(ip))
                {
                    continue;
                }
                ipWeights[ip] = weightMap.TryGetValue(node.Id, out var weight) ? weight : 0;
            }

            if (ipWeights.Count == 0)
            {
                return true;
            }

            var desiredIps = ipWeights.Keys.ToList();
            if (provider is IDnsRecordSetUpdater updater)
            {
                await updater.UpsertRecordSetAsync(domainName, record, desiredIps);
            }
            else
            {
                await SyncLineRecordSetLegacyAsync(provider, domainName, record, ipWeights);
            }
            return true;
        }

        if (action != "add" && action != "delete")
        {
            return false;
        }

        var ids = nodeIds?.Where(id => id > 0).Distinct().ToList() ?? new List<long>();
        if (ids.Count == 0)
        {
            return true;
        }

        if (action == "delete")
        {
            var remaining = await _db.Queryable<Line>()
                .Where(l => l.NodeGroupId == groupId && l.LineId == lineId && l.Enable == true)
                .CountAsync();
            if (remaining == 0)
            {
                return true;
            }
        }

        var nodeRows = await _db.Queryable<Node>()
            .Where(n => ids.Contains(n.Id))
            .Select(n => new { n.Id, n.Ip })
            .ToListAsync();

        if (nodeRows.Count == 0)
        {
            return false;
        }

        var weights = action == "add"
            ? await LoadLineWeightMapAsync(groupId, lineId, ids)
            : new Dictionary<long, int>();

        foreach (var node in nodeRows)
        {
            var ip = node.Ip?.Trim();
            if (string.IsNullOrWhiteSpace(ip))
            {
                continue;
            }

            record.Value = ip;
            record.Weight = weights.TryGetValue(node.Id, out var weight) ? weight : 0;
            if (action == "add")
            {
                await provider.AddRecordAsync(domainName, record);
            }
            else
            {
                await provider.DeleteRecordAsync(domainName, record);
            }
        }

        return true;
    }

    public async Task<bool> SyncPackageCnameForLineChangeAsync(
        long groupId,
        string lineId,
        string lineName,
        IReadOnlyList<long> nodeIds,
        string action)
    {
        if (groupId <= 0)
        {
            return true;
        }

        action = (action ?? string.Empty).Trim().ToLowerInvariant();
        if (action == "enable")
        {
            action = "add";
        }
        if (action == "disable")
        {
            action = "delete";
        }
        if (string.IsNullOrWhiteSpace(action))
        {
            return true;
        }

        IReadOnlyList<long> resolvedNodeIds = nodeIds ?? Array.Empty<long>();
        if (action == "resync")
        {
            resolvedNodeIds = await LoadLineNodeIdsAsync(groupId, lineId);
        }
        else
        {
            resolvedNodeIds = resolvedNodeIds.Where(id => id > 0).Distinct().ToList();
        }

        var (infos, domainMap) = await LoadSiteCnameInfosAsync(new[] { groupId });
        if (infos.Count == 0 || domainMap.Count == 0)
        {
            return true;
        }

        foreach (var info in infos)
        {
            if (string.IsNullOrWhiteSpace(info.Hostname) || string.IsNullOrWhiteSpace(info.DomainKey))
            {
                continue;
            }
            if (info.PrimaryGroup != groupId && (!info.EnableBackup || info.BackupGroup != groupId))
            {
                if (info.PrimaryGroup != 0 || info.BackupGroup != 0)
                {
                    continue;
                }
            }

            if (!domainMap.TryGetValue(info.DomainKey, out var domain))
            {
                continue;
            }

            var ok = await SyncPackageLineRecordsAsync(domain, info.Hostname, groupId, lineId, lineName, action, resolvedNodeIds);
            if (!ok)
            {
                return false;
            }
        }

        return true;
    }

    public async Task<bool> SyncPackageCnameForNodesAsync(IReadOnlyList<long> nodeIds, string action)
    {
        if (nodeIds == null || nodeIds.Count == 0)
        {
            return true;
        }

        action = (action ?? string.Empty).Trim().ToLowerInvariant();
        if (string.IsNullOrWhiteSpace(action))
        {
            return true;
        }

        var ids = nodeIds.Where(id => id > 0).Distinct().ToList();
        if (ids.Count == 0)
        {
            return true;
        }

        var subIds = await _db.Queryable<Node>()
            .Where(n => ids.Contains(n.Pid))
            .Select(n => (long)n.Id)
            .ToListAsync();

        ids.AddRange(subIds);
        ids = ids.Where(id => id > 0).Distinct().ToList();
        if (ids.Count == 0)
        {
            return true;
        }

        var lines = await _db.Queryable<Line>()
            .Where(l => ids.Contains(l.NodeId ?? 0) || ids.Contains(l.NodeIpId ?? 0))
            .Select(l => new { l.NodeGroupId, l.LineId, l.LineName, l.NodeId, l.NodeIpId, l.Enable })
            .ToListAsync();

        if (lines.Count == 0)
        {
            return true;
        }

        var groupLineNodes = new Dictionary<long, Dictionary<LineKey, List<long>>>();
        foreach (var line in lines)
        {
            if (action != "delete" && line.Enable != true)
            {
                continue;
            }

            var key = new LineKey(NormalizeLineId(line.LineId), NormalizeLineName(line.LineName, line.LineId));
            if (!groupLineNodes.TryGetValue(line.NodeGroupId ?? 0, out var map))
            {
                map = new Dictionary<LineKey, List<long>>();
                groupLineNodes[line.NodeGroupId ?? 0] = map;
            }

            if (!map.TryGetValue(key, out var list))
            {
                list = new List<long>();
                map[key] = list;
            }

            var nodeId = line.NodeIpId ?? 0;
            if (nodeId == 0)
            {
                nodeId = line.NodeId ?? 0;
            }
            if (nodeId > 0)
            {
                list.Add(nodeId);
            }
        }

        var groupIds = groupLineNodes.Keys.Where(id => id > 0).Distinct().ToList();
        if (groupIds.Count == 0)
        {
            return true;
        }

        var (infos, domainMap) = await LoadSiteCnameInfosAsync(groupIds);
        if (infos.Count == 0 || domainMap.Count == 0)
        {
            return true;
        }

        foreach (var info in infos)
        {
            if (string.IsNullOrWhiteSpace(info.Hostname) || string.IsNullOrWhiteSpace(info.DomainKey))
            {
                continue;
            }

            if (!domainMap.TryGetValue(info.DomainKey, out var domain))
            {
                continue;
            }

            var targetGroupSet = new HashSet<long>();
            if (info.PrimaryGroup != 0 && groupLineNodes.ContainsKey(info.PrimaryGroup))
            {
                targetGroupSet.Add(info.PrimaryGroup);
            }
            if (info.EnableBackup && info.BackupGroup != 0 && groupLineNodes.ContainsKey(info.BackupGroup))
            {
                targetGroupSet.Add(info.BackupGroup);
            }
            if (targetGroupSet.Count == 0 && info.PrimaryGroup == 0 && info.BackupGroup == 0)
            {
                foreach (var gid in groupIds)
                {
                    targetGroupSet.Add(gid);
                }
            }

            foreach (var gid in targetGroupSet)
            {
                if (!groupLineNodes.TryGetValue(gid, out var lineMap))
                {
                    continue;
                }

                foreach (var pair in lineMap)
                {
                    var uniqueNodes = pair.Value.Where(id => id > 0).Distinct().ToList();
                    if (uniqueNodes.Count == 0)
                    {
                        continue;
                    }

                    var ok = await SyncPackageLineRecordsAsync(domain, info.Hostname, gid, pair.Key.Id, pair.Key.Name, action, uniqueNodes);
                    if (!ok)
                    {
                        return false;
                    }
                }
            }
        }

        return true;
    }

    public async Task<bool> SyncPackageLineRecordsAsync(
        CnameDomains domain,
        string host,
        long groupId,
        string lineId,
        string lineName,
        string action,
        IReadOnlyList<long> nodeIds)
    {
        if (domain == null)
        {
            return false;
        }

        var root = NormalizeDomainName(domain.Domain);
        if (string.IsNullOrWhiteSpace(root) || string.IsNullOrWhiteSpace(host))
        {
            return false;
        }

        if (domain.DnsProviderId <= 0 || groupId <= 0)
        {
            return true;
        }

        var group = await EnsureGroupDnsConfigAsync(groupId);
        if (group == null)
        {
            return false;
        }

        var lineDomain = NormalizeDomainName(group.CnameDomain);
        var lineHost = NormalizeRecordHostname(group.CnameHostname, lineDomain);
        if (string.IsNullOrWhiteSpace(lineDomain) || string.IsNullOrWhiteSpace(lineHost))
        {
            return false;
        }

        var resolvedLineHost = lineHost == "@" ? lineDomain : $"{lineHost}.{lineDomain}";

        var api = await _db.Queryable<Dnsapi>().Where(p => p.Id == domain.DnsProviderId).FirstAsync();
        if (api == null)
        {
            return false;
        }

        if (!ValidateDnsPodIntl(api.Type, api.Auth))
        {
            return false;
        }

        var provider = await TryCreateProviderAsync(api);
        if (provider == null)
        {
            return false;
        }

        var ttl = ResolveTtl(api.Auth);
        var lineValue = ResolveLineValue(api.Type, lineId, lineName);

        action = (action ?? string.Empty).Trim().ToLowerInvariant();
        if (action != "add" && action != "delete" && action != "enable" && action != "disable" && action != "resync")
        {
            return false;
        }

        var record = new DnsRecord
        {
            Type = "CNAME",
            Name = host.Trim(),
            Value = resolvedLineHost,
            TTL = ttl,
            Line = lineValue
        };

        var recordName = record.Name;
        var desiredValue = NormalizeDomainName(resolvedLineHost);
        var records = await provider.GetRecordsAsync(root);
        var existing = new List<DnsRecord>();
        var hasDesired = false;

        foreach (var item in records)
        {
            if (!string.Equals(item.Type, "CNAME", StringComparison.OrdinalIgnoreCase))
            {
                continue;
            }
            if (!string.Equals(item.Name, recordName, StringComparison.OrdinalIgnoreCase))
            {
                continue;
            }
            if (!string.IsNullOrWhiteSpace(lineValue) && !string.Equals(item.Line, lineValue, StringComparison.OrdinalIgnoreCase))
            {
                continue;
            }

            existing.Add(item);
            if (!string.IsNullOrWhiteSpace(desiredValue) && string.Equals(NormalizeDomainName(item.Value), desiredValue, StringComparison.OrdinalIgnoreCase))
            {
                hasDesired = true;
            }
        }

        if (!hasDesired)
        {
            if (existing.Count > 0)
            {
                if (provider is IDnsRecordValueReplacer replacer)
                {
                    await replacer.ReplaceRecordValueAsync(root, new DnsRecord
                    {
                        Type = "CNAME",
                        Name = recordName,
                        Line = lineValue,
                        TTL = ttl
                    }, resolvedLineHost);
                    hasDesired = true;
                }
                else
                {
                    await DeleteAllByLineAsync(provider, root, new DnsRecord
                    {
                        Type = "CNAME",
                        Name = recordName,
                        Line = lineValue
                    });
                }
            }

            if (!hasDesired)
            {
                await provider.AddRecordAsync(root, record);
            }
        }

        foreach (var item in existing)
        {
            if (string.Equals(NormalizeDomainName(item.Value), desiredValue, StringComparison.OrdinalIgnoreCase))
            {
                continue;
            }
            await provider.DeleteRecordAsync(root, item);
        }
        return true;
    }

    private static async Task SyncLineRecordSetLegacyAsync(
        IDnsRecordProvider provider,
        string domain,
        DnsRecord record,
        IReadOnlyDictionary<string, int> desiredWeights)
    {
        if (desiredWeights == null || desiredWeights.Count == 0)
        {
            return;
        }

        var records = await provider.GetRecordsAsync(domain);
        var existing = new List<DnsRecord>();
        foreach (var item in records)
        {
            if (!string.Equals(item.Type, record.Type, StringComparison.OrdinalIgnoreCase))
            {
                continue;
            }
            if (!string.Equals(item.Name, record.Name, StringComparison.OrdinalIgnoreCase))
            {
                continue;
            }
            if (!string.IsNullOrWhiteSpace(record.Line) && !string.Equals(item.Line, record.Line, StringComparison.OrdinalIgnoreCase))
            {
                continue;
            }

            existing.Add(item);
        }

        var desiredSet = new HashSet<string>(desiredWeights.Keys, StringComparer.OrdinalIgnoreCase);
        foreach (var item in existing)
        {
            if (desiredSet.Contains(item.Value))
            {
                continue;
            }
            await provider.DeleteRecordAsync(domain, item);
        }

        foreach (var pair in desiredWeights)
        {
            var value = pair.Key;
            if (string.IsNullOrWhiteSpace(value))
            {
                continue;
            }

            var weight = pair.Value;
            var matches = existing.Where(r => string.Equals(r.Value, value, StringComparison.OrdinalIgnoreCase)).ToList();
            if (matches.Count == 0)
            {
                await provider.AddRecordAsync(domain, new DnsRecord
                {
                    Type = record.Type,
                    Name = record.Name,
                    Line = record.Line,
                    TTL = record.TTL,
                    Value = value,
                    Weight = weight
                });
                continue;
            }

            var needsUpdate = matches.Any(r =>
                (record.TTL > 0 && r.TTL != record.TTL) ||
                (weight > 0 && r.Weight != weight));
            if (!needsUpdate)
            {
                continue;
            }

            foreach (var item in matches)
            {
                await provider.DeleteRecordAsync(domain, item);
            }

            await provider.AddRecordAsync(domain, new DnsRecord
            {
                Type = record.Type,
                Name = record.Name,
                Line = record.Line,
                TTL = record.TTL,
                Value = value,
                Weight = weight
            });
        }
    }

    private static async Task DeleteAllByLineAsync(IDnsRecordProvider provider, string domain, DnsRecord record)
    {
        if (provider is IDnsLineRecordDeleter deleter)
        {
            await deleter.DeleteRecordsByLineAsync(domain, record);
            return;
        }

        var records = await provider.GetRecordsAsync(domain);
        foreach (var item in records)
        {
            if (!string.Equals(item.Type, record.Type, StringComparison.OrdinalIgnoreCase))
            {
                continue;
            }
            if (!string.Equals(item.Name, record.Name, StringComparison.OrdinalIgnoreCase))
            {
                continue;
            }
            if (!string.IsNullOrWhiteSpace(record.Line) && !string.Equals(item.Line, record.Line, StringComparison.OrdinalIgnoreCase))
            {
                continue;
            }

            await provider.DeleteRecordAsync(domain, item);
        }
    }

    private async Task<NodeGroup?> EnsureGroupDnsConfigAsync(long groupId)
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
            var fallback = await LoadFirstCnameDomainAsync();
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
            var generated = await GenerateUniqueGroupHostnameAsync();
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

    private async Task<string> LoadFirstCnameDomainAsync()
    {
        var row = await _db.Queryable<CnameDomains>().OrderBy(d => d.Id).FirstAsync();
        if (row == null)
        {
            return string.Empty;
        }
        return NormalizeDomainName(row.Domain);
    }

    private async Task<string> GenerateUniqueGroupHostnameAsync()
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

    private async Task<Dnsapi?> ResolveDnsApiForSiteAsync(Site site)
    {
        if (site == null)
        {
            return null;
        }

        if (site.DnsProviderId is > 0)
        {
            var query = _db.Queryable<Dnsapi>().Where(d => d.Id == site.DnsProviderId.Value);
            if (site.Uid is > 0)
            {
                query = query.Where(d => d.Uid == site.Uid.Value);
            }
            return await query.FirstAsync();
        }

        var domainKey = NormalizeDomainName(site.CnameDomain);
        if (string.IsNullOrWhiteSpace(domainKey) && !string.IsNullOrWhiteSpace(site.CnameHostname))
        {
            var (root, _) = DnsHelper.SplitRootDomain(site.CnameHostname);
            domainKey = NormalizeDomainName(root);
        }

        if (string.IsNullOrWhiteSpace(domainKey))
        {
            return null;
        }

        var cname = await _db.Queryable<CnameDomains>().Where(d => d.Domain == domainKey).FirstAsync();
        if (cname == null || cname.DnsProviderId <= 0)
        {
            return null;
        }

        return await _db.Queryable<Dnsapi>().Where(d => d.Id == cname.DnsProviderId).FirstAsync();
    }

    private async Task UpsertSiteDomainsAsync(Dnsapi api, IReadOnlyList<string> domains, string? cname)
    {
        if (api == null || domains.Count == 0 || string.IsNullOrWhiteSpace(cname))
        {
            return;
        }

        var provider = await TryCreateProviderAsync(api);
        if (provider == null)
        {
            return;
        }

        foreach (var domain in domains)
        {
            var (root, name) = DnsHelper.SplitRootDomain(domain);
            if (string.IsNullOrWhiteSpace(root) || string.IsNullOrWhiteSpace(name))
            {
                continue;
            }

            var record = new DnsRecord
            {
                Type = "CNAME",
                Name = name,
                Value = cname.Trim(),
                TTL = 600
            };

            var existing = await provider.GetRecordsAsync(root);
            var matched = existing.FirstOrDefault(r =>
                string.Equals(r.Type, "CNAME", StringComparison.OrdinalIgnoreCase) &&
                string.Equals(r.Name, name, StringComparison.OrdinalIgnoreCase));

            if (matched != null)
            {
                if (string.Equals(matched.Value, record.Value, StringComparison.OrdinalIgnoreCase))
                {
                    continue;
                }

                await provider.DeleteRecordAsync(root, matched);
            }

            await provider.AddRecordAsync(root, record);
        }
    }

    private async Task DeleteSiteDomainsAsync(Dnsapi api, IReadOnlyList<string> domains, string? cname)
    {
        if (api == null || domains.Count == 0 || string.IsNullOrWhiteSpace(cname))
        {
            return;
        }

        var provider = await TryCreateProviderAsync(api);
        if (provider == null)
        {
            return;
        }

        foreach (var domain in domains)
        {
            var (root, name) = DnsHelper.SplitRootDomain(domain);
            if (string.IsNullOrWhiteSpace(root) || string.IsNullOrWhiteSpace(name))
            {
                continue;
            }

            var record = new DnsRecord
            {
                Type = "CNAME",
                Name = name,
                Value = cname.Trim(),
                TTL = 600
            };

            await provider.DeleteRecordAsync(root, record);
        }
    }

    private static List<string> DecodeDomains(string? raw)
    {
        return DomainParser.ParseDomains(raw).Select(d => d?.Trim() ?? string.Empty)
            .Where(d => !string.IsNullOrWhiteSpace(d))
            .ToList();
    }

    private static List<string> DiffDomains(IReadOnlyList<string> oldDomains, IReadOnlyList<string> newDomains)
    {
        var newSet = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
        foreach (var domain in newDomains)
        {
            var normalized = NormalizeDomainName(domain);
            if (!string.IsNullOrWhiteSpace(normalized))
            {
                newSet.Add(normalized);
            }
        }

        var removed = new List<string>();
        foreach (var domain in oldDomains)
        {
            var normalized = NormalizeDomainName(domain);
            if (string.IsNullOrWhiteSpace(normalized))
            {
                continue;
            }
            if (!newSet.Contains(normalized))
            {
                removed.Add(domain);
            }
        }

        return removed;
    }

    private static string NormalizeDomainName(string? input)
    {
        return DomainHelper.NormalizeDomainInput(input);
    }

    private static string NormalizeRecordHostname(string? input, string domain)
    {
        var host = NormalizeDomainName(input);
        if (string.IsNullOrWhiteSpace(host))
        {
            return string.Empty;
        }

        domain = NormalizeDomainName(domain);
        if (string.IsNullOrWhiteSpace(domain))
        {
            return host;
        }

        if (string.Equals(host, domain, StringComparison.OrdinalIgnoreCase))
        {
            return "@";
        }

        var suffix = "." + domain;
        if (host.EndsWith(suffix, StringComparison.OrdinalIgnoreCase))
        {
            return host[..^suffix.Length];
        }

        return host;
    }

    private async Task<IReadOnlyList<long>> LoadLineNodeIdsAsync(long groupId, string lineId)
    {
        if (groupId <= 0)
        {
            return Array.Empty<long>();
        }

        var lines = await _db.Queryable<Line>()
            .Where(l => l.NodeGroupId == groupId && l.LineId == lineId && l.Enable == true)
            .Select(l => new { l.NodeId, l.NodeIpId })
            .ToListAsync();

        var ids = new HashSet<long>();
        foreach (var line in lines)
        {
            var nodeId = line.NodeIpId ?? 0;
            if (nodeId == 0)
            {
                nodeId = line.NodeId ?? 0;
            }
            if (nodeId > 0)
            {
                ids.Add(nodeId);
            }
        }

        return ids.ToList();
    }

    private async Task<Dictionary<long, int>> LoadLineWeightMapAsync(long groupId, string lineId, IReadOnlyList<long> nodeIds)
    {
        var result = new Dictionary<long, int>();
        if (groupId <= 0 || nodeIds == null || nodeIds.Count == 0)
        {
            return result;
        }

        var lines = await _db.Queryable<Line>()
            .Where(l => l.NodeGroupId == groupId && l.LineId == lineId &&
                        (nodeIds.Contains(l.NodeId ?? 0) || nodeIds.Contains(l.NodeIpId ?? 0)))
            .Select(l => new { l.NodeId, l.NodeIpId, l.Weight })
            .ToListAsync();

        foreach (var line in lines)
        {
            var nodeId = line.NodeIpId ?? 0;
            if (nodeId == 0)
            {
                nodeId = line.NodeId ?? 0;
            }
            if (nodeId == 0)
            {
                continue;
            }
            result[nodeId] = ParseWeight(line.Weight);
        }

        return result;
    }

    private static int ParseWeight(string? value)
    {
        if (string.IsNullOrWhiteSpace(value))
        {
            return 0;
        }
        return int.TryParse(value.Trim(), out var parsed) && parsed >= 0 ? parsed : 0;
    }

    private static string ResolveLineValue(string? providerType, string lineId, string lineName)
    {
        var lineKey = (lineId ?? string.Empty).Trim().ToLowerInvariant();
        if (lineKey == "custom")
        {
            return (lineName ?? string.Empty).Trim();
        }

        if (!string.IsNullOrWhiteSpace(providerType) && LineMaps.TryGetValue(providerType, out var map))
        {
            if (map.TryGetValue(lineKey, out var value))
            {
                return value;
            }
        }

        if (!string.IsNullOrWhiteSpace(lineName))
        {
            return lineName.Trim();
        }

        return string.Empty;
    }

    private static bool ValidateDnsPodIntl(string? providerType, string? auth)
    {
        if (!string.Equals(providerType, "dnspod_intl", StringComparison.OrdinalIgnoreCase))
        {
            return true;
        }
        if (string.IsNullOrWhiteSpace(auth))
        {
            return false;
        }
        try
        {
            using var doc = JsonDocument.Parse(auth);
            if (!doc.RootElement.TryGetProperty("secret_id", out var secretId))
            {
                return false;
            }
            if (!doc.RootElement.TryGetProperty("secret_key", out var secretKey))
            {
                return false;
            }
            return !string.IsNullOrWhiteSpace(secretId.GetString()) && !string.IsNullOrWhiteSpace(secretKey.GetString());
        }
        catch
        {
            return false;
        }
    }

    private static int ResolveTtl(string? auth)
    {
        if (string.IsNullOrWhiteSpace(auth))
        {
            return 600;
        }
        try
        {
            using var doc = JsonDocument.Parse(auth);
            if (doc.RootElement.TryGetProperty("ttl", out var ttlElement))
            {
                if (ttlElement.ValueKind == JsonValueKind.Number && ttlElement.TryGetInt32(out var ttl) && ttl > 0)
                {
                    return ttl;
                }
                if (ttlElement.ValueKind == JsonValueKind.String &&
                    int.TryParse(ttlElement.GetString(), out var ttlParsed) && ttlParsed > 0)
                {
                    return ttlParsed;
                }
            }
        }
        catch
        {
        }
        return 600;
    }

    private async Task<(List<SiteCnameInfo> Infos, Dictionary<string, CnameDomains> Domains)> LoadSiteCnameInfosAsync(
        IReadOnlyList<long> groupIds)
    {
        if (groupIds == null || groupIds.Count == 0)
        {
            return (new List<SiteCnameInfo>(), new Dictionary<string, CnameDomains>());
        }

        var groups = groupIds.Where(id => id > 0).Distinct().ToList();
        if (groups.Count == 0)
        {
            return (new List<SiteCnameInfo>(), new Dictionary<string, CnameDomains>());
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
            return (new List<SiteCnameInfo>(), new Dictionary<string, CnameDomains>());
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

        var infos = new List<SiteCnameInfo>(sites.Count);
        var domainSet = new HashSet<string>(StringComparer.OrdinalIgnoreCase);

        foreach (var site in sites)
        {
            if (site.UserPackage is null or <= 0 || !packMap.TryGetValue(site.UserPackage.Value, out var pkg))
            {
                continue;
            }

            var (domainKey, host) = ResolveSiteCnameTarget(site, pkg);
            if (string.IsNullOrWhiteSpace(domainKey) || string.IsNullOrWhiteSpace(host))
            {
                continue;
            }

            var planGroup = planGroupMap.TryGetValue(pkg.Package ?? 0, out var p) ? p : new PlanGroup();
            var (primary, backup, enableBackup) = ResolveSiteGroups(site, pkg, planGroup);

            infos.Add(new SiteCnameInfo
            {
                SiteId = site.Id,
                Hostname = host,
                DomainKey = domainKey,
                PrimaryGroup = primary,
                BackupGroup = backup,
                EnableBackup = enableBackup
            });

            domainSet.Add(domainKey);
        }

        var domains = new Dictionary<string, CnameDomains>(StringComparer.OrdinalIgnoreCase);
        if (domainSet.Count > 0)
        {
            var domainRows = await _db.Queryable<CnameDomains>()
                .Where(d => domainSet.Contains(d.Domain))
                .ToListAsync();
            foreach (var row in domainRows)
            {
                var key = NormalizeDomainName(row.Domain);
                if (!string.IsNullOrWhiteSpace(key))
                {
                    domains[key] = row;
                }
            }
        }

        return (infos, domains);
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

    private static IReadOnlyDictionary<string, string> BuildCommonLineMap()
    {
        return new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase)
        {
            ["default"] = "Default",
            ["telecom"] = "Telecom",
            ["unicom"] = "Unicom",
            ["mobile"] = "Mobile",
            ["ctt"] = "TieTong",
            ["broadnet"] = "Broadcast",
            ["edu"] = "Edu",
            ["cn"] = "China",
            ["global"] = "Oversea",
            ["search"] = "Search"
        };
    }

    private static IReadOnlyDictionary<string, string> BuildChinaLineMap()
    {
        return new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase)
        {
            ["default"] = "Default",
            ["telecom"] = "Telecom",
            ["unicom"] = "Unicom",
            ["mobile"] = "Mobile",
            ["china"] = "China",
            ["cn"] = "China",
            ["global"] = "Oversea",
            ["search"] = "Search",
            ["anhui"] = "Anhui",
            ["beijing"] = "Beijing",
            ["chongqing"] = "Chongqing",
            ["fujian"] = "Fujian",
            ["gansu"] = "Gansu",
            ["guangdong"] = "Guangdong",
            ["guangxi"] = "Guangxi",
            ["guizhou"] = "Guizhou",
            ["hainan"] = "Hainan",
            ["hebei"] = "Hebei",
            ["heilongjiang"] = "Heilongjiang",
            ["henan"] = "Henan",
            ["hubei"] = "Hubei",
            ["hunan"] = "Hunan",
            ["jiangsu"] = "Jiangsu",
            ["jiangxi"] = "Jiangxi",
            ["jilin"] = "Jilin",
            ["liaoning"] = "Liaoning",
            ["neimenggu"] = "Neimenggu",
            ["ningxia"] = "Ningxia",
            ["qinghai"] = "Qinghai",
            ["shaanxi"] = "Shaanxi",
            ["shandong"] = "Shandong",
            ["shanghai"] = "Shanghai",
            ["shanxi"] = "Shanxi",
            ["sichuan"] = "Sichuan",
            ["tianjin"] = "Tianjin",
            ["xizang"] = "Xizang",
            ["xinjiang"] = "Xinjiang",
            ["yunnan"] = "Yunnan",
            ["zhejiang"] = "Zhejiang",
            ["tie-tong"] = "TieTong",
            ["ctt"] = "TieTong",
            ["broadcast"] = "Broadcast",
            ["broadnet"] = "Broadcast",
            ["edu"] = "Edu"
        };
    }

    private static IReadOnlyDictionary<string, string> BuildDnsPodIntlLineMap()
    {
        return new Dictionary<string, string>(StringComparer.OrdinalIgnoreCase)
        {
            ["default"] = "Default",
            ["telecom"] = "Telecom",
            ["unicom"] = "Unicom",
            ["mobile"] = "Mobile",
            ["china"] = "China",
            ["cn"] = "China",
            ["global"] = "Oversea",
            ["search"] = "Search",
            ["anhui"] = "Anhui",
            ["beijing"] = "Beijing",
            ["chongqing"] = "Chongqing",
            ["fujian"] = "Fujian",
            ["gansu"] = "Gansu",
            ["guangdong"] = "Guangdong",
            ["guangxi"] = "Guangxi",
            ["guizhou"] = "Guizhou",
            ["hainan"] = "Hainan",
            ["hebei"] = "Hebei",
            ["heilongjiang"] = "Heilongjiang",
            ["henan"] = "Henan",
            ["hubei"] = "Hubei",
            ["hunan"] = "Hunan",
            ["jiangsu"] = "Jiangsu",
            ["jiangxi"] = "Jiangxi",
            ["jilin"] = "Jilin",
            ["liaoning"] = "Liaoning",
            ["neimenggu"] = "Neimenggu",
            ["ningxia"] = "Ningxia",
            ["qinghai"] = "Qinghai",
            ["shaanxi"] = "Shaanxi",
            ["shandong"] = "Shandong",
            ["shanghai"] = "Shanghai",
            ["shanxi"] = "Shanxi",
            ["sichuan"] = "Sichuan",
            ["tianjin"] = "Tianjin",
            ["xizang"] = "Xizang",
            ["xinjiang"] = "Xinjiang",
            ["yunnan"] = "Yunnan",
            ["zhejiang"] = "Zhejiang",
            ["tie-tong"] = "TieTong",
            ["ctt"] = "TieTong",
            ["broadcast"] = "Broadcast",
            ["broadnet"] = "Broadcast",
            ["edu"] = "Edu"
        };
    }

    private Task<IDnsRecordProvider?> TryCreateProviderAsync(Dnsapi api)
    {
        if (api == null || string.IsNullOrWhiteSpace(api.Type) || string.IsNullOrWhiteSpace(api.Auth))
        {
            return Task.FromResult<IDnsRecordProvider?>(null);
        }

        return Task.FromResult(_providerResolver.Resolve(api.Type, api.Auth));
    }

    private sealed record LineKey(string Id, string Name);

    private sealed class SiteCnameInfo
    {
        public long SiteId { get; init; }
        public string Hostname { get; init; } = string.Empty;
        public string DomainKey { get; init; } = string.Empty;
        public long PrimaryGroup { get; init; }
        public long BackupGroup { get; init; }
        public bool EnableBackup { get; init; }
    }


}
