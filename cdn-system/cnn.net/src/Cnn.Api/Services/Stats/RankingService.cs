using System.Text.Json;
using Microsoft.Extensions.Configuration;

namespace Cnn.Api.Services.Stats;

public sealed record RankItem(string Item, ulong RequestCount, ulong OutBytes, ulong OriginBytes);

public sealed class LatencyRankItem
{
    public int Rank { get; set; }
    public string? Item { get; set; }
    public int RequestCount { get; set; }
    public double AvgTime { get; set; }
    public double MaxTime { get; set; }
    public double MinTime { get; set; }
    public double P95Time { get; set; }
}

public interface IRankingService
{
    Task<IReadOnlyList<RankItem>> QueryAccessRankingAsync(string rankType, DateTime start, DateTime end, HostFilter hostFilter, string? keyword, int limit, CancellationToken cancellationToken);
    Task<IReadOnlyList<RankItem>> QueryRegionRankingAsync(string regionType, DateTime start, DateTime end, HostFilter hostFilter, string? keyword, int limit, CancellationToken cancellationToken);
    Task<IReadOnlyList<LatencyRankItem>> QueryLatencyRankingAsync(DateTime start, DateTime end, HostFilter hostFilter, string? keyword, int limit, CancellationToken cancellationToken);
}

public sealed class RankingService : IRankingService
{
    private readonly IConfiguration _configuration;

    public RankingService(IConfiguration configuration)
    {
        _configuration = configuration;
    }

    public async Task<IReadOnlyList<RankItem>> QueryAccessRankingAsync(
        string rankType,
        DateTime start,
        DateTime end,
        HostFilter hostFilter,
        string? keyword,
        int limit,
        CancellationToken cancellationToken)
    {
        if (start == DateTime.MinValue || end == DateTime.MinValue || end < start)
        {
            return Array.Empty<RankItem>();
        }

        if (limit <= 0)
        {
            limit = 50;
        }

        var spec = ResolveRankingSpec(rankType);
        if (spec == null)
        {
            return Array.Empty<RankItem>();
        }

        var cfg = ClickHouseHttpHelper.ResolveConfig(_configuration);
        if (cfg == null)
        {
            return Array.Empty<RankItem>();
        }

        var conditions = new List<string>
        {
            $"ts >= toDateTime('{start:yyyy-MM-dd HH:mm:ss}') AND ts <= toDateTime('{end:yyyy-MM-dd HH:mm:ss}')"
        };

        var hostClause = hostFilter.BuildHttpCondition();
        if (!string.IsNullOrWhiteSpace(hostClause))
        {
            conditions.Add(hostClause);
        }

        keyword = keyword?.Trim();
        if (!string.IsNullOrWhiteSpace(keyword) && !string.IsNullOrWhiteSpace(spec.KeywordCondition))
        {
            conditions.Add(spec.KeywordCondition.Replace("{kw}", ClickHouseHttpHelper.QuoteString("%" + keyword + "%")));
        }

        var where = string.Join(" AND ", conditions);
        var query =
            $"SELECT {spec.ItemExpr} AS item," +
            " count() AS request_count," +
            " sum(\"bytes\") AS out_traffic," +
            " sumIf(\"bytes\", upstream_cache_status != 'HIT') AS origin_traffic" +
            $" FROM node_access_logs WHERE {where} GROUP BY {spec.GroupBy} ORDER BY request_count DESC LIMIT {limit} FORMAT JSONEachRow";

        var rows = await ClickHouseHttpHelper.QueryRowsAsync(cfg, query, cancellationToken);
        if (rows == null || rows.Length == 0)
        {
            return Array.Empty<RankItem>();
        }

        var list = new List<RankItem>();
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
                var item = ReadString(root, "item");
                if (string.IsNullOrWhiteSpace(item))
                {
                    item = spec.NormalizeNil ? "-" : string.Empty;
                }

                list.Add(new RankItem(
                    item,
                    ReadUInt64(root, "request_count"),
                    ReadUInt64(root, "out_traffic"),
                    ReadUInt64(root, "origin_traffic")
                ));
            }
            catch
            {
                // ignore row
            }
        }

        return list;
    }

    public async Task<IReadOnlyList<RankItem>> QueryRegionRankingAsync(
        string regionType,
        DateTime start,
        DateTime end,
        HostFilter hostFilter,
        string? keyword,
        int limit,
        CancellationToken cancellationToken)
    {
        if (start == DateTime.MinValue || end == DateTime.MinValue || end < start)
        {
            return Array.Empty<RankItem>();
        }

        var cfg = ClickHouseHttpHelper.ResolveConfig(_configuration);
        if (cfg == null)
        {
            return Array.Empty<RankItem>();
        }

        var conditions = new List<string>
        {
            $"ts >= toDateTime('{start:yyyy-MM-dd HH:mm:ss}') AND ts <= toDateTime('{end:yyyy-MM-dd HH:mm:ss}')"
        };

        var hostClause = hostFilter.BuildHttpCondition();
        if (!string.IsNullOrWhiteSpace(hostClause))
        {
            conditions.Add(hostClause);
        }

        var where = string.Join(" AND ", conditions);
        var itemExpr = regionType.Trim().ToLowerInvariant() switch
        {
            "country" => AccessLogGeoExpressions.ClientCountryExpr(),
            "province" => AccessLogGeoExpressions.ClientProvinceExpr(),
            _ => string.Empty
        };
        if (string.IsNullOrWhiteSpace(itemExpr))
        {
            return Array.Empty<RankItem>();
        }

        keyword = keyword?.Trim();
        if (!string.IsNullOrWhiteSpace(keyword))
        {
            conditions.Add($"{itemExpr} LIKE {ClickHouseHttpHelper.QuoteString("%" + keyword + "%")}");
            where = string.Join(" AND ", conditions);
        }

        var query =
            $"SELECT {itemExpr} AS item," +
            " count() AS request_count," +
            " sum(\"bytes\") AS out_traffic," +
            " sumIf(\"bytes\", upstream_cache_status != 'HIT') AS origin_traffic" +
            $" FROM node_access_logs WHERE {where} GROUP BY {itemExpr} ORDER BY request_count DESC LIMIT {limit} FORMAT JSONEachRow";

        var rows = await ClickHouseHttpHelper.QueryRowsAsync(cfg, query, cancellationToken);
        if (rows == null || rows.Length == 0)
        {
            return Array.Empty<RankItem>();
        }

        var list = new List<RankItem>();
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
                var item = ReadString(root, "item");
                if (string.IsNullOrWhiteSpace(item))
                {
                    item = "-";
                }

                list.Add(new RankItem(
                    item,
                    ReadUInt64(root, "request_count"),
                    ReadUInt64(root, "out_traffic"),
                    ReadUInt64(root, "origin_traffic")
                ));
            }
            catch
            {
                // ignore
            }
        }

        return list;
    }

    public async Task<IReadOnlyList<LatencyRankItem>> QueryLatencyRankingAsync(
        DateTime start,
        DateTime end,
        HostFilter hostFilter,
        string? keyword,
        int limit,
        CancellationToken cancellationToken)
    {
        if (start == DateTime.MinValue || end == DateTime.MinValue || end < start)
        {
            return Array.Empty<LatencyRankItem>();
        }

        if (limit <= 0)
        {
            limit = 50;
        }

        var cfg = ClickHouseHttpHelper.ResolveConfig(_configuration);
        if (cfg == null)
        {
            return Array.Empty<LatencyRankItem>();
        }

        var conditions = new List<string>
        {
            $"ts >= toDateTime('{start:yyyy-MM-dd HH:mm:ss}') AND ts <= toDateTime('{end:yyyy-MM-dd HH:mm:ss}') AND request_time > 0"
        };

        var hostClause = hostFilter.BuildHttpCondition();
        if (!string.IsNullOrWhiteSpace(hostClause))
        {
            conditions.Add(hostClause);
        }

        keyword = keyword?.Trim();
        if (!string.IsNullOrWhiteSpace(keyword))
        {
            var kw = ClickHouseHttpHelper.QuoteString("%" + keyword + "%");
            conditions.Add($"(host LIKE {kw} OR uri LIKE {kw})");
        }

        var where = string.Join(" AND ", conditions);
        var query =
            "SELECT host, uri, count() AS request_count," +
            " avg(request_time) AS avg_time," +
            " max(request_time) AS max_time," +
            " min(request_time) AS min_time," +
            " quantile(0.95)(request_time) AS p95_time" +
            $" FROM node_access_logs WHERE {where} GROUP BY host, uri ORDER BY avg_time DESC LIMIT {limit} FORMAT JSONEachRow";

        var rows = await ClickHouseHttpHelper.QueryRowsAsync(cfg, query, cancellationToken);
        if (rows == null || rows.Length == 0)
        {
            return Array.Empty<LatencyRankItem>();
        }

        var list = new List<LatencyRankItem>();
        var rank = 1;
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
                var uri = ReadString(root, "uri");
                var item = string.IsNullOrWhiteSpace(uri) ? host : host + uri;

                list.Add(new LatencyRankItem
                {
                    Rank = rank++,
                    Item = item,
                    RequestCount = (int)ReadUInt64(root, "request_count"),
                    AvgTime = StatsFormat.RoundFloat(ReadDouble(root, "avg_time"), 3),
                    MaxTime = StatsFormat.RoundFloat(ReadDouble(root, "max_time"), 3),
                    MinTime = StatsFormat.RoundFloat(ReadDouble(root, "min_time"), 3),
                    P95Time = StatsFormat.RoundFloat(ReadDouble(root, "p95_time"), 3)
                });
            }
            catch
            {
                // ignore
            }
        }

        return list;
    }

    private static RankingSpec? ResolveRankingSpec(string rankType)
    {
        rankType = (rankType ?? string.Empty).Trim().ToLowerInvariant();
        return rankType switch
        {
            "domain" => new RankingSpec("host", "host", "host LIKE {kw}", false),
            "url" => new RankingSpec("concat(host, uri)", "host, uri", "(host LIKE {kw} OR uri LIKE {kw})", false),
            "ip" => new RankingSpec("remote_addr", "remote_addr", "remote_addr LIKE {kw}", false),
            "referer" => new RankingSpec("http_referer", "http_referer", "http_referer LIKE {kw}", true),
            _ => null
        };
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
            _ => value.ToString() ?? string.Empty
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

    private static double ReadDouble(JsonElement root, string key)
    {
        if (!root.TryGetProperty(key, out var value))
        {
            return 0;
        }

        if (value.ValueKind == JsonValueKind.Number && value.TryGetDouble(out var num))
        {
            return num;
        }

        if (value.ValueKind == JsonValueKind.String && double.TryParse(value.GetString(), out var parsed))
        {
            return parsed;
        }

        return 0;
    }

    private sealed record RankingSpec(string ItemExpr, string GroupBy, string KeywordCondition, bool NormalizeNil);
}
