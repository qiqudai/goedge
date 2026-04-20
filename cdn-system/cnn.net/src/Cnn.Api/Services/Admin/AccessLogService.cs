using System.Text.Json;
using Cnn.Common.Contracts;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;
using Cnn.Api.Services.Stats;
using Microsoft.Extensions.Configuration;

namespace Cnn.Api.Services.Admin;

public interface IAccessLogService
{
    Task<ServiceResult<AccessLogListResult>> ListAsync(
        AccessLogQuery query,
        DateTime? startTime,
        DateTime? endTime,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken);
}

public sealed class AccessLogService : IAccessLogService
{
    private const int DefaultPageSize = 20;
    private const int MaxPageSize = 200;

    private readonly IConfiguration _configuration;
    private readonly ISiteHostIndexService _siteHostIndexService;
    private readonly ISpiderIpAllowlistService _spiderIpAllowlist;

    public AccessLogService(
        IConfiguration configuration,
        ISiteHostIndexService siteHostIndexService,
        ISpiderIpAllowlistService spiderIpAllowlist)
    {
        _configuration = configuration;
        _siteHostIndexService = siteHostIndexService;
        _spiderIpAllowlist = spiderIpAllowlist;
    }

    public async Task<ServiceResult<AccessLogListResult>> ListAsync(
        AccessLogQuery query,
        DateTime? startTime,
        DateTime? endTime,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        query ??= new AccessLogQuery();
        var page = query.Page.GetValueOrDefault() < 1 ? 1 : query.Page!.Value;
        var pageSize = query.PageSize.GetValueOrDefault() < 1 ? DefaultPageSize : query.PageSize!.Value;
        if (pageSize > MaxPageSize)
        {
            pageSize = MaxPageSize;
        }

        HostFilter? hostFilter = null;
        if (!isAdmin)
        {
            if (!userId.HasValue || userId.Value <= 0)
            {
                return ServiceResult<AccessLogListResult>.Ok(new AccessLogListResult(Array.Empty<AccessLogItem>(), 0));
            }

            var index = await _siteHostIndexService.LoadAsync(userId.Value, cancellationToken);
            if (index.Filter.Empty)
            {
                return ServiceResult<AccessLogListResult>.Ok(new AccessLogListResult(Array.Empty<AccessLogItem>(), 0));
            }

            hostFilter = index.Filter;
        }

        var cfg = ClickHouseHttpHelper.ResolveConfig(_configuration);
        if (cfg == null)
        {
            return ServiceResult<AccessLogListResult>.Ok(new AccessLogListResult(Array.Empty<AccessLogItem>(), 0));
        }

        var where = BuildWhere(query, startTime, endTime, hostFilter);
        var offset = (page - 1) * pageSize;

        var total = await QueryCountAsync(cfg, where, cancellationToken);
        if (total == 0)
        {
            return ServiceResult<AccessLogListResult>.Ok(new AccessLogListResult(Array.Empty<AccessLogItem>(), 0));
        }

        var rows = await QueryRowsAsync(cfg, where, pageSize, offset, cancellationToken);
        if (rows == null || rows.Count == 0)
        {
            return ServiceResult<AccessLogListResult>.Ok(new AccessLogListResult(Array.Empty<AccessLogItem>(), total));
        }

        foreach (var item in rows)
        {
            if (!isAdmin)
            {
                item.UpstreamAddr = string.Empty;
                item.NodeIp = string.Empty;
                continue;
            }

            if (!_spiderIpAllowlist.IsSpiderIp(item.RemoteAddr))
            {
                item.UpstreamAddr = string.Empty;
            }
        }

        return ServiceResult<AccessLogListResult>.Ok(new AccessLogListResult(rows, total));
    }

    private static string BuildWhere(AccessLogQuery query, DateTime? startTime, DateTime? endTime, HostFilter? hostFilter)
    {
        var conditions = new List<string> { "1=1" };

        if (startTime.HasValue && endTime.HasValue)
        {
            var start = startTime.Value.ToString("yyyy-MM-dd HH:mm:ss");
            var end = endTime.Value.ToString("yyyy-MM-dd HH:mm:ss");
            conditions.Add($"ts >= toDateTime('{start}') AND ts <= toDateTime('{end}')");
        }

        if (hostFilter != null && !hostFilter.Empty)
        {
            conditions.Add(hostFilter.BuildHttpCondition());
        }

        var domain = query.Domain?.Trim();
        if (!string.IsNullOrWhiteSpace(domain))
        {
            if (string.Equals(query.DomainMode, "fuzzy", StringComparison.OrdinalIgnoreCase))
            {
                conditions.Add("host LIKE " + ClickHouseHttpHelper.QuoteString("%" + domain + "%"));
            }
            else
            {
                conditions.Add("host = " + ClickHouseHttpHelper.QuoteString(domain));
            }
        }

        var clientIp = query.ClientIp?.Trim();
        if (!string.IsNullOrWhiteSpace(clientIp))
        {
            conditions.Add("remote_addr = " + ClickHouseHttpHelper.QuoteString(clientIp));
        }

        var uri = query.Uri?.Trim();
        if (!string.IsNullOrWhiteSpace(uri))
        {
            if (string.Equals(query.UriMode, "exact", StringComparison.OrdinalIgnoreCase))
            {
                conditions.Add("uri = " + ClickHouseHttpHelper.QuoteString(uri));
            }
            else
            {
                conditions.Add("uri LIKE " + ClickHouseHttpHelper.QuoteString("%" + uri + "%"));
            }
        }

        var method = query.Method?.Trim();
        if (!string.IsNullOrWhiteSpace(method))
        {
            conditions.Add("method = " + ClickHouseHttpHelper.QuoteString(method.ToUpperInvariant()));
        }

        if (int.TryParse(query.Status, out var status))
        {
            conditions.Add("status = " + status);
        }

        if (int.TryParse(query.StatusMin, out var statusMin))
        {
            conditions.Add("status >= " + statusMin);
        }

        if (int.TryParse(query.StatusMax, out var statusMax))
        {
            conditions.Add("status <= " + statusMax);
        }

        var nodeId = query.NodeId?.Trim();
        if (!string.IsNullOrWhiteSpace(nodeId))
        {
            conditions.Add("node_id = " + ClickHouseHttpHelper.QuoteString(nodeId));
        }

        var nodeIp = query.NodeIp?.Trim();
        if (!string.IsNullOrWhiteSpace(nodeIp))
        {
            conditions.Add("node_ip = " + ClickHouseHttpHelper.QuoteString(nodeIp));
        }

        var port = query.Port?.Trim();
        if (!string.IsNullOrWhiteSpace(port) && int.TryParse(port, out _))
        {
            conditions.Add("host LIKE " + ClickHouseHttpHelper.QuoteString("%:" + port));
        }

        var scheme = query.Scheme?.Trim();
        if (!string.IsNullOrWhiteSpace(scheme))
        {
            conditions.Add("scheme = " + ClickHouseHttpHelper.QuoteString(scheme));
        }

        var cacheStatus = query.CacheStatus?.Trim();
        if (!string.IsNullOrWhiteSpace(cacheStatus))
        {
            conditions.Add("upstream_cache_status = " + ClickHouseHttpHelper.QuoteString(cacheStatus));
        }

        var referer = query.Referer?.Trim();
        if (!string.IsNullOrWhiteSpace(referer))
        {
            conditions.Add("http_referer LIKE " + ClickHouseHttpHelper.QuoteString("%" + referer + "%"));
        }

        var userAgent = query.UserAgent?.Trim();
        if (!string.IsNullOrWhiteSpace(userAgent))
        {
            conditions.Add("http_user_agent LIKE " + ClickHouseHttpHelper.QuoteString("%" + userAgent + "%"));
        }

        var sslProtocol = query.SslProtocol?.Trim();
        if (!string.IsNullOrWhiteSpace(sslProtocol))
        {
            conditions.Add("ssl_protocol = " + ClickHouseHttpHelper.QuoteString(sslProtocol));
        }

        var sslCipher = query.SslCipher?.Trim();
        if (!string.IsNullOrWhiteSpace(sslCipher))
        {
            conditions.Add("ssl_cipher = " + ClickHouseHttpHelper.QuoteString(sslCipher));
        }

        var keyword = query.Keyword?.Trim();
        if (!string.IsNullOrWhiteSpace(keyword))
        {
            var like = ClickHouseHttpHelper.QuoteString("%" + keyword + "%");
            conditions.Add($"(host LIKE {like} OR uri LIKE {like} OR remote_addr LIKE {like})");
        }

        return string.Join(" AND ", conditions);
    }

    private static async Task<long> QueryCountAsync(ClickHouseHttpConfig cfg, string where, CancellationToken cancellationToken)
    {
        var query = $"SELECT count() AS total FROM node_access_logs WHERE {where} FORMAT JSONEachRow";
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

    private static async Task<List<AccessLogItem>> QueryRowsAsync(
        ClickHouseHttpConfig cfg,
        string where,
        int pageSize,
        int offset,
        CancellationToken cancellationToken)
    {
        var query = $"SELECT ts, node_id, node_ip, remote_addr, host, method, uri, status, bytes," +
                    " request_time, upstream_addr, upstream_response_time, upstream_cache_status, http_referer, http_user_agent," +
                    " scheme, ssl_protocol, ssl_cipher" +
                    $" FROM node_access_logs WHERE {where} ORDER BY ts DESC LIMIT {pageSize} OFFSET {offset} FORMAT JSONEachRow";

        var rows = await ClickHouseHttpHelper.QueryRowsAsync(cfg, query, cancellationToken);
        if (rows == null || rows.Length == 0)
        {
            return new List<AccessLogItem>();
        }

        var list = new List<AccessLogItem>();
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
                list.Add(new AccessLogItem
                {
                    Timestamp = ReadString(root, "ts"),
                    NodeId = ReadString(root, "node_id"),
                    NodeIp = ReadString(root, "node_ip"),
                    RemoteAddr = ReadString(root, "remote_addr"),
                    Host = ReadString(root, "host"),
                    Method = ReadString(root, "method"),
                    Uri = ReadString(root, "uri"),
                    Status = ReadInt(root, "status"),
                    Bytes = ReadLong(root, "bytes"),
                    RequestTime = ReadDouble(root, "request_time"),
                    UpstreamAddr = ReadString(root, "upstream_addr"),
                    UpstreamResponseTime = ReadDouble(root, "upstream_response_time"),
                    UpstreamCacheStatus = ReadString(root, "upstream_cache_status"),
                    HttpReferer = ReadString(root, "http_referer"),
                    HttpUserAgent = ReadString(root, "http_user_agent"),
                    Scheme = ReadString(root, "scheme"),
                    SslProtocol = ReadString(root, "ssl_protocol"),
                    SslCipher = ReadString(root, "ssl_cipher")
                });
            }
            catch
            {
                // ignore invalid rows
            }
        }

        return list;
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

    private static double ReadDouble(JsonElement root, string name)
    {
        if (!root.TryGetProperty(name, out var value))
        {
            return 0;
        }

        if (value.ValueKind == JsonValueKind.Number && value.TryGetDouble(out var result))
        {
            return result;
        }

        if (value.ValueKind == JsonValueKind.String && double.TryParse(value.GetString(), out result))
        {
            return result;
        }

        return 0;
    }
}
