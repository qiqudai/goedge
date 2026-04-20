using System.Text.Json;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;
using Cnn.Api.Services.Stats;
using Microsoft.Extensions.Configuration;

namespace Cnn.Api.Services.Admin;

public interface IBlockLogService
{
    Task<ServiceResult<BlockCurrentListResult>> ListCurrentAsync(BlockLogQuery query, DateTime start, DateTime end, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<BlockStatListResult>> ListStatsAsync(BlockLogQuery query, DateTime start, DateTime end, long? userId, bool isAdmin, CancellationToken cancellationToken);
    Task<ServiceResult<BlockHistoryListResult>> ListHistoryAsync(BlockLogQuery query, DateTime start, DateTime end, long? userId, bool isAdmin, CancellationToken cancellationToken);
}

public sealed class BlockLogService : IBlockLogService
{
    private static readonly int[] BlockedStatusCodes = { 403, 418, 429, 451, 410 };
    private const int DefaultPageSize = 10;
    private const int MaxPageSize = 200;

    private readonly IConfiguration _configuration;
    private readonly ISiteHostIndexService _siteHostIndexService;

    public BlockLogService(
        IConfiguration configuration,
        ISiteHostIndexService siteHostIndexService)
    {
        _configuration = configuration;
        _siteHostIndexService = siteHostIndexService;
    }

    public async Task<ServiceResult<BlockCurrentListResult>> ListCurrentAsync(
        BlockLogQuery query,
        DateTime start,
        DateTime end,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        query ??= new BlockLogQuery();
        var (page, pageSize) = ResolvePaging(query);
        var (index, hostFilter, ipFilter) = await ResolveFiltersAsync(query, userId, isAdmin, cancellationToken);
        if (hostFilter == null)
        {
            return ServiceResult<BlockCurrentListResult>.Ok(new BlockCurrentListResult(Array.Empty<BlockCurrentItem>(), 0));
        }

        var cfg = ClickHouseHttpHelper.ResolveConfig(_configuration);
        if (cfg == null)
        {
            return ServiceResult<BlockCurrentListResult>.Ok(new BlockCurrentListResult(Array.Empty<BlockCurrentItem>(), 0));
        }

        var where = BuildWhere(start, end, hostFilter, ipFilter);
        var offset = (page - 1) * pageSize;

        var total = await QueryTotalAsync(cfg, $"SELECT uniqExact((host, remote_addr)) AS total FROM node_access_logs WHERE {where}", cancellationToken);
        if (total == 0)
        {
            return ServiceResult<BlockCurrentListResult>.Ok(new BlockCurrentListResult(Array.Empty<BlockCurrentItem>(), 0));
        }

        var querySql = $"SELECT host, remote_addr, " +
                       $"argMax({AccessLogGeoExpressions.ClientCountryExpr()}, ts) AS agg_client_country, " +
                       $"argMax({AccessLogGeoExpressions.ClientProvinceExpr()}, ts) AS agg_client_province, " +
                       $"max(ts) AS block_time, any(status) AS status_code " +
                       $"FROM node_access_logs WHERE {where} GROUP BY host, remote_addr ORDER BY block_time DESC " +
                       $"LIMIT {pageSize} OFFSET {offset} FORMAT JSONEachRow";

        var rows = await ClickHouseHttpHelper.QueryRowsAsync(cfg, querySql, cancellationToken);
        if (rows == null || rows.Length == 0)
        {
            return ServiceResult<BlockCurrentListResult>.Ok(new BlockCurrentListResult(Array.Empty<BlockCurrentItem>(), total));
        }

        var list = new List<BlockCurrentItem>();
        var indexOffset = offset;
        foreach (var row in rows)
        {
            if (string.IsNullOrWhiteSpace(row))
            {
                continue;
            }

            try
            {
                using var doc = JsonDocument.Parse(row);
                var root = doc.RootElement;
                var host = ReadString(root, "host");
                var ip = ReadString(root, "remote_addr");
                var country = ReadString(root, "agg_client_country");
                var province = ReadString(root, "agg_client_province");
                var status = ReadInt(root, "status_code");
                var blockTime = NormalizeBlockTime(ReadString(root, "block_time"));
                var (siteId, domain) = ResolveSite(index, host);
                list.Add(new BlockCurrentItem
                {
                    Id = ++indexOffset,
                    SiteId = siteId,
                    Domain = domain,
                    Ip = ip,
                    Location = AccessLogGeoExpressions.FormatLocation(country, province),
                    Filter = BuildStatusLabel(status),
                    BlockTime = blockTime,
                    ReleaseTime = "PERMANENT"
                });
            }
            catch
            {
                // ignore invalid rows
            }
        }

        return ServiceResult<BlockCurrentListResult>.Ok(new BlockCurrentListResult(list, total));
    }

    public async Task<ServiceResult<BlockStatListResult>> ListStatsAsync(
        BlockLogQuery query,
        DateTime start,
        DateTime end,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        query ??= new BlockLogQuery();
        var (page, pageSize) = ResolvePaging(query);
        var (index, hostFilter, _) = await ResolveFiltersAsync(query, userId, isAdmin, cancellationToken, ignoreIpFilter: true);
        if (hostFilter == null)
        {
            return ServiceResult<BlockStatListResult>.Ok(new BlockStatListResult(Array.Empty<BlockStatItem>(), 0));
        }

        var cfg = ClickHouseHttpHelper.ResolveConfig(_configuration);
        if (cfg == null)
        {
            return ServiceResult<BlockStatListResult>.Ok(new BlockStatListResult(Array.Empty<BlockStatItem>(), 0));
        }

        var where = BuildWhere(start, end, hostFilter, string.Empty);
        var offset = (page - 1) * pageSize;

        var total = await QueryTotalAsync(cfg, $"SELECT uniqExact(host) AS total FROM node_access_logs WHERE {where}", cancellationToken);
        if (total == 0)
        {
            return ServiceResult<BlockStatListResult>.Ok(new BlockStatListResult(Array.Empty<BlockStatItem>(), 0));
        }

        var querySql = $"SELECT host, count() AS cnt FROM node_access_logs WHERE {where} " +
                       $"GROUP BY host ORDER BY cnt DESC LIMIT {pageSize} OFFSET {offset} FORMAT JSONEachRow";

        var rows = await ClickHouseHttpHelper.QueryRowsAsync(cfg, querySql, cancellationToken);
        if (rows == null || rows.Length == 0)
        {
            return ServiceResult<BlockStatListResult>.Ok(new BlockStatListResult(Array.Empty<BlockStatItem>(), total));
        }

        var list = new List<BlockStatItem>();
        foreach (var row in rows)
        {
            if (string.IsNullOrWhiteSpace(row))
            {
                continue;
            }

            try
            {
                using var doc = JsonDocument.Parse(row);
                var root = doc.RootElement;
                var host = ReadString(root, "host");
                var count = ReadLong(root, "cnt");
                var (siteId, domain) = ResolveSite(index, host);
                list.Add(new BlockStatItem
                {
                    SiteId = siteId,
                    Domain = domain,
                    Count = count
                });
            }
            catch
            {
                // ignore invalid rows
            }
        }

        return ServiceResult<BlockStatListResult>.Ok(new BlockStatListResult(list, total));
    }

    public async Task<ServiceResult<BlockHistoryListResult>> ListHistoryAsync(
        BlockLogQuery query,
        DateTime start,
        DateTime end,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        query ??= new BlockLogQuery();
        var (page, pageSize) = ResolvePaging(query);
        var (index, hostFilter, ipFilter) = await ResolveFiltersAsync(query, userId, isAdmin, cancellationToken);
        if (hostFilter == null)
        {
            return ServiceResult<BlockHistoryListResult>.Ok(new BlockHistoryListResult(Array.Empty<BlockHistoryItem>(), 0));
        }

        var cfg = ClickHouseHttpHelper.ResolveConfig(_configuration);
        if (cfg == null)
        {
            return ServiceResult<BlockHistoryListResult>.Ok(new BlockHistoryListResult(Array.Empty<BlockHistoryItem>(), 0));
        }

        var where = BuildWhere(start, end, hostFilter, ipFilter);
        var offset = (page - 1) * pageSize;

        var total = await QueryTotalAsync(cfg, $"SELECT count() AS total FROM node_access_logs WHERE {where}", cancellationToken);
        if (total == 0)
        {
            return ServiceResult<BlockHistoryListResult>.Ok(new BlockHistoryListResult(Array.Empty<BlockHistoryItem>(), 0));
        }

        var querySql = $"SELECT ts, host, remote_addr, " +
                       $"{AccessLogGeoExpressions.ClientCountryExpr()} AS client_country, " +
                       $"{AccessLogGeoExpressions.ClientProvinceExpr()} AS client_province, " +
                       $"status FROM node_access_logs WHERE {where} " +
                       $"ORDER BY ts DESC LIMIT {pageSize} OFFSET {offset} FORMAT JSONEachRow";

        var rows = await ClickHouseHttpHelper.QueryRowsAsync(cfg, querySql, cancellationToken);
        if (rows == null || rows.Length == 0)
        {
            return ServiceResult<BlockHistoryListResult>.Ok(new BlockHistoryListResult(Array.Empty<BlockHistoryItem>(), total));
        }

        var list = new List<BlockHistoryItem>();
        var indexOffset = offset;
        foreach (var row in rows)
        {
            if (string.IsNullOrWhiteSpace(row))
            {
                continue;
            }

            try
            {
                using var doc = JsonDocument.Parse(row);
                var root = doc.RootElement;
                var host = ReadString(root, "host");
                var ip = ReadString(root, "remote_addr");
                var country = ReadString(root, "client_country");
                var province = ReadString(root, "client_province");
                var status = ReadInt(root, "status");
                var blockTime = NormalizeBlockTime(ReadString(root, "ts"));
                var (siteId, domain) = ResolveSite(index, host);
                list.Add(new BlockHistoryItem
                {
                    Id = ++indexOffset,
                    SiteId = siteId,
                    Domain = domain,
                    Ip = ip,
                    Location = AccessLogGeoExpressions.FormatLocation(country, province),
                    Filter = BuildStatusLabel(status),
                    BlockTime = blockTime,
                    IsManual = false
                });
            }
            catch
            {
                // ignore invalid rows
            }
        }

        return ServiceResult<BlockHistoryListResult>.Ok(new BlockHistoryListResult(list, total));
    }

    private static (int Page, int PageSize) ResolvePaging(BlockLogQuery query)
    {
        var page = query.Page.GetValueOrDefault() < 1 ? 1 : query.Page!.Value;
        var pageSize = query.PageSize.GetValueOrDefault() < 1 ? DefaultPageSize : query.PageSize!.Value;
        if (pageSize > MaxPageSize)
        {
            pageSize = MaxPageSize;
        }

        return (page, pageSize);
    }

    private async Task<(SiteHostIndex? Index, HostFilter? Filter, string IpFilter)> ResolveFiltersAsync(
        BlockLogQuery query,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken,
        bool ignoreIpFilter = false)
    {
        var filterType = query.Type?.Trim().ToLowerInvariant();
        if (string.IsNullOrWhiteSpace(filterType))
        {
            filterType = "ip";
        }

        var keyword = query.Keyword?.Trim() ?? string.Empty;
        var index = await _siteHostIndexService.LoadAsync(isAdmin ? 0 : (userId ?? 0), cancellationToken);

        if (!isAdmin)
        {
            if (userId.GetValueOrDefault() <= 0)
            {
                return (index, null, string.Empty);
            }

            if (index.Filter.Empty)
            {
                return (index, null, string.Empty);
            }
        }

        HostFilter? hostFilter = isAdmin ? new HostFilter() : index.Filter;
        var ipFilter = string.Empty;

        if (filterType == "site_id")
        {
            if (!long.TryParse(keyword, out var siteId) || siteId <= 0)
            {
                return (index, null, string.Empty);
            }

            if (!index.SiteFilters.TryGetValue(siteId, out var siteFilter) || siteFilter.Empty)
            {
                return (index, null, string.Empty);
            }

            hostFilter = siteFilter;
        }
        else if (!ignoreIpFilter)
        {
            if (filterType == "ip")
            {
                ipFilter = keyword;
            }
            else if (filterType == "time_range")
            {
                ipFilter = string.Empty;
            }
            else
            {
                ipFilter = keyword;
            }
        }

        return (index, hostFilter ?? new HostFilter(), ipFilter);
    }

    private static string BuildWhere(DateTime start, DateTime end, HostFilter? hostFilter, string ipFilter)
    {
        var conditions = new List<string>
        {
            $"ts >= toDateTime('{start:yyyy-MM-dd HH:mm:ss}') AND ts <= toDateTime('{end:yyyy-MM-dd HH:mm:ss}')",
            BuildStatusCondition()
        };

        if (hostFilter != null && !hostFilter.Empty)
        {
            conditions.Add(hostFilter.BuildHttpCondition());
        }

        if (!string.IsNullOrWhiteSpace(ipFilter))
        {
            conditions.Add("remote_addr = " + ClickHouseHttpHelper.QuoteString(ipFilter));
        }

        return string.Join(" AND ", conditions);
    }

    private static string BuildStatusCondition()
    {
        return "status IN (" + string.Join(",", BlockedStatusCodes) + ")";
    }

    private static async Task<long> QueryTotalAsync(ClickHouseHttpConfig cfg, string sql, CancellationToken cancellationToken)
    {
        var query = sql + " FORMAT JSONEachRow";
        var rows = await ClickHouseHttpHelper.QueryRowsAsync(cfg, query, cancellationToken);
        if (rows == null || rows.Length == 0)
        {
            return 0;
        }

        foreach (var row in rows)
        {
            if (string.IsNullOrWhiteSpace(row))
            {
                continue;
            }

            try
            {
                using var doc = JsonDocument.Parse(row);
                var root = doc.RootElement;
                if (root.TryGetProperty("total", out var value) && value.TryGetInt64(out var total))
                {
                    return total;
                }
            }
            catch
            {
                return 0;
            }
        }

        return 0;
    }

    private (long SiteId, string Domain) ResolveSite(SiteHostIndex? index, string? host)
    {
        var domain = host?.Trim() ?? string.Empty;
        if (index == null || string.IsNullOrWhiteSpace(domain))
        {
            return (0, domain);
        }

        var normalized = DomainParser.NormalizeDomain(domain);
        foreach (var entry in index.SiteFilters)
        {
            var filter = entry.Value;
            if (filter.Exact.Any(item => string.Equals(item, normalized, StringComparison.OrdinalIgnoreCase)))
            {
                return (entry.Key, domain);
            }

            foreach (var suffix in filter.Wildcards)
            {
                if (normalized.EndsWith(suffix, StringComparison.OrdinalIgnoreCase))
                {
                    return (entry.Key, domain);
                }
            }
        }

        return (0, domain);
    }

    private static string BuildStatusLabel(int status)
    {
        if (status <= 0)
        {
            return "-";
        }

        return "HTTP_" + status;
    }

    private static string NormalizeBlockTime(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return "-";
        }

        return raw;
    }

    private static string? ReadString(JsonElement root, string name)
    {
        if (!root.TryGetProperty(name, out var value))
        {
            return null;
        }

        return value.ValueKind switch
        {
            JsonValueKind.String => value.GetString(),
            JsonValueKind.Number => value.GetRawText(),
            _ => null
        };
    }

    private static int ReadInt(JsonElement root, string name)
    {
        if (!root.TryGetProperty(name, out var value))
        {
            return 0;
        }

        if (value.ValueKind == JsonValueKind.Number && value.TryGetInt32(out var result))
        {
            return result;
        }

        if (value.ValueKind == JsonValueKind.String && int.TryParse(value.GetString(), out result))
        {
            return result;
        }

        return 0;
    }

    private static long ReadLong(JsonElement root, string name)
    {
        if (!root.TryGetProperty(name, out var value))
        {
            return 0;
        }

        if (value.ValueKind == JsonValueKind.Number && value.TryGetInt64(out var result))
        {
            return result;
        }

        if (value.ValueKind == JsonValueKind.String && long.TryParse(value.GetString(), out result))
        {
            return result;
        }

        return 0;
    }
}
