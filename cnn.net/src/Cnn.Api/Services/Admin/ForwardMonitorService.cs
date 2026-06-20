using System.Globalization;
using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Common;
using Cnn.Domain.Entities;
using Microsoft.Extensions.Configuration;
using SqlSugar;
using Stream = Cnn.Domain.Entities.Stream;

namespace Cnn.Api.Services.Admin;

public sealed class ForwardMonitorService : IForwardMonitorService
{
    private static readonly HttpClient HttpClient = new()
    {
        Timeout = TimeSpan.FromSeconds(5)
    };

    private readonly ISqlSugarClient _db;
    private readonly ISystemConfigService _systemConfigService;
    private readonly IConfiguration _configuration;

    public ForwardMonitorService(ISqlSugarClient db, ISystemConfigService systemConfigService, IConfiguration configuration)
    {
        _db = db;
        _systemConfigService = systemConfigService;
        _configuration = configuration;
    }

    public async Task<ServiceResult<ForwardTrafficResult>> GetTrafficAsync(
        string? range,
        string? keyword,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        var rangeKey = string.IsNullOrWhiteSpace(range) ? "1h" : range.Trim().ToLowerInvariant();
        var (start, end, step, bucketMinutes, labelFormat) = ResolveForwardRange(rangeKey);
        var (port, protocol) = ParseForwardKeyword(keyword);
        var allowedPorts = new List<int>();

        if (!isAdmin)
        {
            if (!userId.HasValue || userId <= 0)
            {
                return ServiceResult<ForwardTrafficResult>.Ok(new ForwardTrafficResult());
            }

            var (portMap, ports) = await LoadUserForwardPortMapAsync(userId.Value);
            if (ports.Count == 0)
            {
                return ServiceResult<ForwardTrafficResult>.Ok(new ForwardTrafficResult());
            }

            if (port > 0)
            {
                if (!PortAllowed(portMap, port, protocol))
                {
                    return ServiceResult<ForwardTrafficResult>.Ok(new ForwardTrafficResult());
                }
                allowedPorts.Add(port);
            }
            else if (!string.IsNullOrWhiteSpace(protocol))
            {
                var filtered = FilterPortsByProtocol(portMap, protocol);
                if (filtered.Count == 0)
                {
                    return ServiceResult<ForwardTrafficResult>.Ok(new ForwardTrafficResult());
                }
                allowedPorts.AddRange(filtered);
            }
            else
            {
                allowedPorts.AddRange(ports);
            }
        }

        var buckets = await QueryForwardTrafficBucketsWithPortsAsync(start, end, bucketMinutes, port, protocol, allowedPorts, cancellationToken);
        var factor = await ResolveForwardTrafficFactorAsync(cancellationToken);

        var bucketMap = new Dictionary<DateTime, ulong>();
        foreach (var bucket in buckets)
        {
            var key = bucket.Bucket.Truncate(step);
            bucketMap[key] = bucket.TotalBytes;
        }

        var times = new List<string>();
        var bandwidth = new List<double>();
        var traffic = new List<double>();

        var cur = start;
        while (cur <= end)
        {
            bucketMap.TryGetValue(cur, out var totalBytes);
            var adjusted = totalBytes;
            if (Math.Abs(factor - 1.0) > 0.00001)
            {
                adjusted = (ulong)(totalBytes * factor);
            }

            var trafficGb = adjusted / (1024.0 * 1024.0 * 1024.0);
            var bandwidthMbps = 0.0;
            if (step.TotalSeconds > 0)
            {
                bandwidthMbps = adjusted * 8 / (step.TotalSeconds * 1000 * 1000.0);
            }

            times.Add(cur.ToString(labelFormat));
            bandwidth.Add(Math.Round(bandwidthMbps, 3));
            traffic.Add(Math.Round(trafficGb, 3));

            cur = cur.Add(step);
        }

        return ServiceResult<ForwardTrafficResult>.Ok(new ForwardTrafficResult
        {
            XAxis = times,
            Bandwidth = bandwidth,
            Traffic = traffic
        });
    }

    public async Task<ServiceResult<ForwardRankingResult>> GetRankingAsync(
        string? range,
        long? userId,
        bool isAdmin,
        CancellationToken cancellationToken)
    {
        var rangeKey = string.IsNullOrWhiteSpace(range) ? "1h" : range.Trim().ToLowerInvariant();
        var (start, end, _, _, _) = ResolveForwardRange(rangeKey);
        var allowedPorts = new List<int>();

        if (!isAdmin)
        {
            if (!userId.HasValue || userId <= 0)
            {
                return ServiceResult<ForwardRankingResult>.Ok(new ForwardRankingResult(Array.Empty<ForwardRankingItemDto>()));
            }

            var (_, ports) = await LoadUserForwardPortMapAsync(userId.Value);
            if (ports.Count == 0)
            {
                return ServiceResult<ForwardRankingResult>.Ok(new ForwardRankingResult(Array.Empty<ForwardRankingItemDto>()));
            }
            allowedPorts.AddRange(ports);
        }

        var list = await QueryForwardPortRankingWithPortsAsync(start, end, 50, allowedPorts, cancellationToken);
        var factor = await ResolveForwardTrafficFactorAsync(cancellationToken);

        var result = new List<ForwardRankingItemDto>();
        foreach (var item in list)
        {
            var adjusted = item.TotalBytes;
            if (Math.Abs(factor - 1.0) > 0.00001)
            {
                adjusted = (ulong)(item.TotalBytes * factor);
            }
            var trafficGb = adjusted / (1024.0 * 1024.0 * 1024.0);
            var protocol = string.IsNullOrWhiteSpace(item.Protocol) ? "TCP" : item.Protocol.Trim().ToUpperInvariant();
            if (protocol is not "TCP" and not "UDP")
            {
                protocol = "TCP";
            }
            var label = $"{item.Port}/{protocol}";
            result.Add(new ForwardRankingItemDto
            {
                Port = label,
                Connections = item.Connections,
                Traffic = $"{trafficGb:F2} GB"
            });
        }

        return ServiceResult<ForwardRankingResult>.Ok(new ForwardRankingResult(result));
    }

    private static (DateTime Start, DateTime End, TimeSpan Step, int BucketMinutes, string LabelFormat) ResolveForwardRange(string rangeKey)
    {
        var end = DateTime.Now;
        switch (rangeKey)
        {
            case "6h":
                {
                    var step = TimeSpan.FromMinutes(5);
                    var start = end.AddHours(-6).Truncate(step);
                    return (start, end.Truncate(step), step, 5, "HH:mm");
                }
            case "24h":
                {
                    var step = TimeSpan.FromMinutes(30);
                    var start = end.AddHours(-24).Truncate(step);
                    return (start, end.Truncate(step), step, 30, "MM-dd HH:mm");
                }
            default:
                {
                    var step = TimeSpan.FromMinutes(1);
                    var start = end.AddHours(-1).Truncate(step);
                    return (start, end.Truncate(step), step, 1, "HH:mm");
                }
        }
    }

    private static (int Port, string Protocol) ParseForwardKeyword(string? keyword)
    {
        if (string.IsNullOrWhiteSpace(keyword))
        {
            return (0, string.Empty);
        }

        var raw = keyword.Trim();
        var portPart = raw;
        var protocol = string.Empty;

        if (raw.Contains('/'))
        {
            var parts = raw.Split('/', 2);
            portPart = parts[0].Trim();
            protocol = parts[1].Trim();
        }

        var lower = raw.ToLowerInvariant();
        if (string.IsNullOrWhiteSpace(protocol))
        {
            if (lower.Contains("tcp"))
            {
                protocol = "TCP";
            }
            else if (lower.Contains("udp"))
            {
                protocol = "UDP";
            }
        }

        protocol = protocol.Trim().ToUpperInvariant();
        if (protocol is not "TCP" and not "UDP")
        {
            protocol = string.Empty;
        }

        var digits = new string(portPart.Where(char.IsDigit).ToArray());
        if (string.IsNullOrWhiteSpace(digits) || !int.TryParse(digits, out var port) || port <= 0)
        {
            return (0, protocol);
        }

        return (port, protocol);
    }

    private async Task<double> ResolveForwardTrafficFactorAsync(CancellationToken cancellationToken)
    {
        var cfg = await _systemConfigService.LoadSystemConfigAsync(cancellationToken);
        if (!cfg.TryGetValue("tcp_traffic_factor", out var raw))
        {
            return 1.0;
        }

        raw = raw.Trim();
        if (string.IsNullOrWhiteSpace(raw) || !double.TryParse(raw, out var factor) || factor <= 0)
        {
            return 1.0;
        }

        return factor;
    }

    private async Task<(Dictionary<int, HashSet<string>> PortMap, List<int> Ports)> LoadUserForwardPortMapAsync(long userId)
    {
        var forwards = await _db.Queryable<Stream>()
            .Where(s => s.Uid == (int)userId)
            .Select(s => new { s.Listen })
            .ToListAsync();

        var map = new Dictionary<int, HashSet<string>>();
        foreach (var forward in forwards)
        {
            var entries = SplitFieldsFromRaw(forward.Listen);
            foreach (var entry in entries)
            {
                var (port, protocol) = ParseForwardListenPort(entry);
                if (port <= 0)
                {
                    continue;
                }

                if (!map.TryGetValue(port, out var protocols))
                {
                    protocols = new HashSet<string>(StringComparer.OrdinalIgnoreCase);
                    map[port] = protocols;
                }

                if (string.IsNullOrWhiteSpace(protocol))
                {
                    protocols.Add("TCP");
                }
                else
                {
                    protocols.Add(protocol);
                }
            }
        }

        var ports = map.Keys.OrderBy(p => p).ToList();
        return (map, ports);
    }

    private static (int Port, string Protocol) ParseForwardListenPort(string raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return (0, string.Empty);
        }

        var text = raw.Trim();
        var protocol = string.Empty;
        if (text.Contains('/'))
        {
            var parts = text.Split('/', 2);
            text = parts[0].Trim();
            protocol = parts[1].Trim().ToUpperInvariant();
        }

        if (protocol is not "TCP" and not "UDP")
        {
            protocol = string.Empty;
        }

        var digits = new string(text.Where(char.IsDigit).ToArray());
        if (string.IsNullOrWhiteSpace(digits) || !int.TryParse(digits, out var port) || port <= 0)
        {
            return (0, protocol);
        }

        return (port, protocol);
    }

    private static bool PortAllowed(Dictionary<int, HashSet<string>> portMap, int port, string protocol)
    {
        if (!portMap.TryGetValue(port, out var protocols))
        {
            return false;
        }

        if (string.IsNullOrWhiteSpace(protocol))
        {
            return true;
        }

        return protocols.Contains(protocol);
    }

    private static List<int> FilterPortsByProtocol(Dictionary<int, HashSet<string>> portMap, string protocol)
    {
        protocol = protocol.Trim().ToUpperInvariant();
        var list = new List<int>();
        foreach (var (port, protocols) in portMap)
        {
            if (protocols.Contains(protocol))
            {
                list.Add(port);
            }
        }
        list.Sort();
        return list;
    }

    private static List<string> SplitFieldsFromRaw(string? raw)
    {
        if (string.IsNullOrWhiteSpace(raw))
        {
            return new List<string>();
        }

        var trimmed = raw.Trim();
        if (trimmed.StartsWith("["))
        {
            try
            {
                var list = System.Text.Json.JsonSerializer.Deserialize<List<string>>(trimmed);
                if (list != null)
                {
                    return list.Select(item => item?.Trim() ?? string.Empty).Where(item => !string.IsNullOrWhiteSpace(item)).ToList();
                }
            }
            catch
            {
            }
        }

        var normalized = trimmed.Replace(",", " ").Replace(";", " ").Replace("\n", " ").Replace("\r", " ");
        return normalized.Split(' ', StringSplitOptions.RemoveEmptyEntries)
            .Select(part => part.Trim())
            .Where(part => !string.IsNullOrWhiteSpace(part))
            .ToList();
    }

    private async Task<List<ForwardTrafficBucket>> QueryForwardTrafficBucketsWithPortsAsync(
        DateTime start,
        DateTime end,
        int bucketMinutes,
        int port,
        string protocol,
        IReadOnlyList<int> allowedPorts,
        CancellationToken cancellationToken)
    {
        var httpCfg = BuildClickHouseHttpConfig();
        if (httpCfg == null)
        {
            return new List<ForwardTrafficBucket>();
        }

        var conditions = BuildForwardConditions(start, end, port, protocol, allowedPorts);
        var query = $"SELECT toStartOfInterval(ts, INTERVAL {bucketMinutes} MINUTE) AS bucket, " +
                    $"sum(bytes_sent + bytes_received) AS total_bytes " +
                    $"FROM node_stream_logs WHERE {string.Join(" AND ", conditions)} " +
                    "GROUP BY bucket ORDER BY bucket FORMAT JSONEachRow";

        var rows = await QueryClickHouseRowsAsync(httpCfg, query, cancellationToken);
        if (rows == null || rows.Length == 0)
        {
            return new List<ForwardTrafficBucket>();
        }

        var list = new List<ForwardTrafficBucket>();
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
                var bucketRaw = ReadString(root, "bucket");
                if (string.IsNullOrWhiteSpace(bucketRaw))
                {
                    continue;
                }

                if (!DateTime.TryParseExact(bucketRaw, "yyyy-MM-dd HH:mm:ss", CultureInfo.InvariantCulture, DateTimeStyles.None, out var bucket))
                {
                    continue;
                }

                var total = ReadUInt64(root, "total_bytes");
                list.Add(new ForwardTrafficBucket(bucket, total));
            }
            catch
            {
                // ignore invalid rows
            }
        }

        return list;
    }

    private async Task<List<ForwardPortRankingItem>> QueryForwardPortRankingWithPortsAsync(
        DateTime start,
        DateTime end,
        int limit,
        IReadOnlyList<int> allowedPorts,
        CancellationToken cancellationToken)
    {
        var httpCfg = BuildClickHouseHttpConfig();
        if (httpCfg == null)
        {
            return new List<ForwardPortRankingItem>();
        }

        var conditions = BuildForwardConditions(start, end, 0, string.Empty, allowedPorts);
        var query = $"SELECT server_port, protocol, count() AS connections, " +
                    $"sum(bytes_sent + bytes_received) AS total_bytes " +
                    $"FROM node_stream_logs WHERE {string.Join(" AND ", conditions)} " +
                    "GROUP BY server_port, protocol " +
                    $"ORDER BY total_bytes DESC LIMIT {limit} FORMAT JSONEachRow";

        var rows = await QueryClickHouseRowsAsync(httpCfg, query, cancellationToken);
        if (rows == null || rows.Length == 0)
        {
            return new List<ForwardPortRankingItem>();
        }

        var list = new List<ForwardPortRankingItem>();
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
                var port = ReadInt(root, "server_port");
                if (port <= 0)
                {
                    continue;
                }

                var protocol = ReadString(root, "protocol");
                var total = ReadUInt64(root, "total_bytes");
                var connections = ReadUInt64(root, "connections");
                list.Add(new ForwardPortRankingItem(port, protocol, total, connections));
            }
            catch
            {
                // ignore invalid rows
            }
        }

        return list;
    }

    private List<string> BuildForwardConditions(
        DateTime start,
        DateTime end,
        int port,
        string protocol,
        IReadOnlyList<int> allowedPorts)
    {
        var conditions = new List<string>
        {
            $"ts >= toDateTime('{start:yyyy-MM-dd HH:mm:ss}') AND ts <= toDateTime('{end:yyyy-MM-dd HH:mm:ss}')"
        };

        if (port > 0)
        {
            conditions.Add($"server_port = {port}");
        }
        else if (allowedPorts.Count > 0)
        {
            var values = string.Join(",", allowedPorts);
            conditions.Add($"server_port IN ({values})");
        }

        if (!string.IsNullOrWhiteSpace(protocol))
        {
            var normalized = protocol.Trim().ToUpperInvariant();
            var escaped = EscapeClickHouseString(normalized);
            conditions.Add($"protocol = '{escaped}'");
        }

        return conditions;
    }

    private ClickHouseHttpConfig? BuildClickHouseHttpConfig()
    {
        var dsn = _configuration["ClickHouse:Dsn"]
            ?? _configuration["ClickHouse:DSN"]
            ?? _configuration["ClickHouse:HttpDsn"]
            ?? _configuration["ClickHouse:HttpDSN"];
        if (string.IsNullOrWhiteSpace(dsn))
        {
            return null;
        }

        if (!Uri.TryCreate(dsn.Trim(), UriKind.Absolute, out var uri))
        {
            return null;
        }

        if (!string.Equals(uri.Scheme, "http", StringComparison.OrdinalIgnoreCase) &&
            !string.Equals(uri.Scheme, "https", StringComparison.OrdinalIgnoreCase))
        {
            return null;
        }

        var database = uri.AbsolutePath.Trim('/');
        if (string.IsNullOrWhiteSpace(database))
        {
            var query = uri.Query.TrimStart('?');
            if (!string.IsNullOrWhiteSpace(query))
            {
                foreach (var pair in query.Split('&', StringSplitOptions.RemoveEmptyEntries))
                {
                    var kv = pair.Split('=', 2);
                    if (kv.Length == 2 && string.Equals(kv[0], "database", StringComparison.OrdinalIgnoreCase))
                    {
                        database = Uri.UnescapeDataString(kv[1]);
                        break;
                    }
                }
            }
        }

        string? user = null;
        string? pass = null;
        if (!string.IsNullOrWhiteSpace(uri.UserInfo))
        {
            var parts = uri.UserInfo.Split(':', 2);
            user = Uri.UnescapeDataString(parts[0]);
            if (parts.Length > 1)
            {
                pass = Uri.UnescapeDataString(parts[1]);
            }
        }

        var baseUrl = uri.GetLeftPart(UriPartial.Authority);
        return new ClickHouseHttpConfig(baseUrl, user, pass, string.IsNullOrWhiteSpace(database) ? null : database);
    }

    private async Task<string[]?> QueryClickHouseRowsAsync(
        ClickHouseHttpConfig cfg,
        string query,
        CancellationToken cancellationToken)
    {
        var builder = new StringBuilder();
        builder.Append(cfg.BaseUrl.TrimEnd('/'));
        builder.Append("/?query=").Append(Uri.EscapeDataString(query));
        if (!string.IsNullOrWhiteSpace(cfg.Database))
        {
            builder.Append("&database=").Append(Uri.EscapeDataString(cfg.Database));
        }

        using var request = new HttpRequestMessage(HttpMethod.Post, builder.ToString());
        if (!string.IsNullOrWhiteSpace(cfg.User))
        {
            var token = Convert.ToBase64String(Encoding.UTF8.GetBytes($"{cfg.User}:{cfg.Pass ?? string.Empty}"));
            request.Headers.Authorization = new AuthenticationHeaderValue("Basic", token);
        }

        using var response = await HttpClient.SendAsync(request, cancellationToken);
        if (!response.IsSuccessStatusCode)
        {
            return null;
        }

        var body = await response.Content.ReadAsStringAsync(cancellationToken);
        if (string.IsNullOrWhiteSpace(body))
        {
            return Array.Empty<string>();
        }

        return body.Split('\n', StringSplitOptions.RemoveEmptyEntries);
    }

    private static string EscapeClickHouseString(string value)
    {
        if (string.IsNullOrEmpty(value))
        {
            return string.Empty;
        }

        return value.Replace("\\", "\\\\").Replace("'", "\\'");
    }

    private static string ReadString(JsonElement root, string key)
    {
        if (!root.TryGetProperty(key, out var value))
        {
            return string.Empty;
        }

        return value.ValueKind switch
        {
            JsonValueKind.String => value.GetString() ?? string.Empty,
            _ => value.ToString()
        };
    }

    private static int ReadInt(JsonElement root, string key)
    {
        if (!root.TryGetProperty(key, out var value))
        {
            return 0;
        }

        return value.ValueKind switch
        {
            JsonValueKind.Number when value.TryGetInt32(out var parsed) => parsed,
            JsonValueKind.String when int.TryParse(value.GetString(), out var parsed) => parsed,
            _ => 0
        };
    }

    private static ulong ReadUInt64(JsonElement root, string key)
    {
        if (!root.TryGetProperty(key, out var value))
        {
            return 0;
        }

        if (value.ValueKind == JsonValueKind.Number)
        {
            if (value.TryGetUInt64(out var parsed))
            {
                return parsed;
            }
            if (value.TryGetInt64(out var signed) && signed > 0)
            {
                return (ulong)signed;
            }
        }

        if (value.ValueKind == JsonValueKind.String && ulong.TryParse(value.GetString(), out var strValue))
        {
            return strValue;
        }

        return 0;
    }

    private sealed record ClickHouseHttpConfig(string BaseUrl, string? User, string? Pass, string? Database);

    private sealed record ForwardTrafficBucket(DateTime Bucket, ulong TotalBytes);

    private sealed record ForwardPortRankingItem(int Port, string Protocol, ulong TotalBytes, ulong Connections);
}

internal static class DateTimeForwardExtensions
{
    public static DateTime Truncate(this DateTime value, TimeSpan step)
    {
        if (step == TimeSpan.Zero)
        {
            return value;
        }

        var ticks = value.Ticks / step.Ticks * step.Ticks;
        return new DateTime(ticks, value.Kind);
    }
}
