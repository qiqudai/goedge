using System.Globalization;
using System.Text.Json;
using Cnn.Common.Contracts;
using Cnn.Api.Services.Common;
using Microsoft.Extensions.Configuration;

namespace Cnn.Api.Services.Stats;

public interface IStatsService
{
    Task<ServiceResult<StatRankingResultDto>> GetRankingAsync(string rankType, string? keyword, StatsRange range, AccessScope scope, CancellationToken cancellationToken);
    Task<ServiceResult<StatLatencyResultDto>> GetLatencyRankingAsync(string? keyword, StatsRange range, AccessScope scope, CancellationToken cancellationToken);
    Task<ServiceResult<StatBasicResultDto>> GetBasicAsync(StatsRange range, AccessScope scope, CancellationToken cancellationToken);
    Task<ServiceResult<StatQualityResultDto>> GetQualityAsync(StatsRange range, AccessScope scope, CancellationToken cancellationToken);
    Task<ServiceResult<StatOriginResultDto>> GetOriginAsync(StatsRange range, AccessScope scope, CancellationToken cancellationToken);
    Task<ServiceResult<StatNodeTrafficDto>> GetNodeTrafficAsync(string? window, CancellationToken cancellationToken);
    Task<ServiceResult<StatNodeRankingResultDto>> GetNodeRankingAsync(string? metric, string? window, CancellationToken cancellationToken);
    Task<ServiceResult<StatNodeMetricsResultDto>> GetNodeMetricsAsync(string? metric, string? window, string? startRaw, string? endRaw, CancellationToken cancellationToken);
    Task<ServiceResult<UsageResultDto>> GetUsageAsync(string? rangeKey, AccessScope scope, CancellationToken cancellationToken);
}

public sealed class StatsService : IStatsService
{
    private const string TimeLayout = "yyyy-MM-dd HH:mm:ss";
    private readonly IAccessStatsService _accessStatsService;
    private readonly IRankingService _rankingService;
    private readonly IHostFilterResolver _hostFilterResolver;
    private readonly ISystemConfigService _systemConfigService;
    private readonly IConfiguration _configuration;

    public StatsService(
        IAccessStatsService accessStatsService,
        IRankingService rankingService,
        IHostFilterResolver hostFilterResolver,
        ISystemConfigService systemConfigService,
        IConfiguration configuration)
    {
        _accessStatsService = accessStatsService;
        _rankingService = rankingService;
        _hostFilterResolver = hostFilterResolver;
        _systemConfigService = systemConfigService;
        _configuration = configuration;
    }

    public async Task<ServiceResult<StatRankingResultDto>> GetRankingAsync(
        string rankType,
        string? keyword,
        StatsRange range,
        AccessScope scope,
        CancellationToken cancellationToken)
    {
        var hostFilter = await _hostFilterResolver.ResolveAsync(scope, cancellationToken);
        if (!scope.IsAdmin && hostFilter.Empty)
        {
            return ServiceResult<StatRankingResultDto>.Ok(new StatRankingResultDto());
        }

        var limit = await ResolveRankSizeAsync(cancellationToken);
        IReadOnlyList<RankItem> items;
        if (rankType is "country" or "province")
        {
            items = await _rankingService.QueryRegionRankingAsync(rankType, range.Start, range.End, hostFilter, keyword, limit, cancellationToken);
        }
        else
        {
            items = await _rankingService.QueryAccessRankingAsync(rankType, range.Start, range.End, hostFilter, keyword, limit, cancellationToken);
        }

        var list = new List<StatRankingItemDto>();
        var rank = 1;
        foreach (var item in items)
        {
            list.Add(new StatRankingItemDto
            {
                Rank = rank++,
                Item = item.Item,
                RequestCount = (int)item.RequestCount,
                OutTraffic = StatsFormat.FormatBytes(item.OutBytes),
                OriginTraffic = StatsFormat.FormatBytes(item.OriginBytes)
            });
        }

        return ServiceResult<StatRankingResultDto>.Ok(new StatRankingResultDto { List = list });
    }

    public async Task<ServiceResult<StatLatencyResultDto>> GetLatencyRankingAsync(
        string? keyword,
        StatsRange range,
        AccessScope scope,
        CancellationToken cancellationToken)
    {
        var hostFilter = await _hostFilterResolver.ResolveAsync(scope, cancellationToken);
        if (!scope.IsAdmin && hostFilter.Empty)
        {
            return ServiceResult<StatLatencyResultDto>.Ok(new StatLatencyResultDto());
        }

        var limit = await ResolveRankSizeAsync(cancellationToken);
        var items = await _rankingService.QueryLatencyRankingAsync(range.Start, range.End, hostFilter, keyword, limit, cancellationToken);
        var list = items.Select(item => new StatLatencyItemDto
        {
            Rank = item.Rank,
            Item = item.Item,
            RequestCount = item.RequestCount,
            AvgTime = item.AvgTime,
            MaxTime = item.MaxTime,
            MinTime = item.MinTime,
            P95Time = item.P95Time
        }).ToList();

        return ServiceResult<StatLatencyResultDto>.Ok(new StatLatencyResultDto { List = list });
    }

    public async Task<ServiceResult<StatBasicResultDto>> GetBasicAsync(
        StatsRange range,
        AccessScope scope,
        CancellationToken cancellationToken)
    {
        var hostFilter = await _hostFilterResolver.ResolveAsync(scope, cancellationToken);
        if (!scope.IsAdmin && hostFilter.Empty)
        {
            return ServiceResult<StatBasicResultDto>.Ok(new StatBasicResultDto());
        }

        var buckets = await _accessStatsService.QueryBucketsAsync(range, hostFilter, cancellationToken);
        var series = _accessStatsService.BuildSeries(range, buckets);

        var bandwidth = new List<double>();
        var traffic = new List<double>();
        var qps = new List<double>();
        var seconds = range.Bucket.TotalSeconds;
        for (var i = 0; i < series.Bytes.Count; i++)
        {
            bandwidth.Add(StatsFormat.RoundFloat(StatsFormat.BytesToMbps(series.Bytes[i], range.Bucket), 2));
            traffic.Add(StatsFormat.RoundFloat(StatsFormat.BytesToMB(series.Bytes[i]), 2));
            var qpsVal = seconds > 0 ? series.Requests[i] / seconds : 0;
            qps.Add(StatsFormat.RoundFloat(qpsVal, 2));
        }

        return ServiceResult<StatBasicResultDto>.Ok(new StatBasicResultDto
        {
            XAxis = series.XAxis,
            Bandwidth = bandwidth,
            Traffic = traffic,
            Qps = qps
        });
    }

    public async Task<ServiceResult<StatQualityResultDto>> GetQualityAsync(
        StatsRange range,
        AccessScope scope,
        CancellationToken cancellationToken)
    {
        var hostFilter = await _hostFilterResolver.ResolveAsync(scope, cancellationToken);
        if (!scope.IsAdmin && hostFilter.Empty)
        {
            return ServiceResult<StatQualityResultDto>.Ok(new StatQualityResultDto());
        }

        var buckets = await _accessStatsService.QueryBucketsAsync(range, hostFilter, cancellationToken);
        var series = _accessStatsService.BuildSeries(range, buckets);

        var hitRate = new List<double>();
        var status4xx = new List<double>();
        var status5xx = new List<double>();
        for (var i = 0; i < series.Requests.Count; i++)
        {
            var value = series.Requests[i] > 0
                ? (double)series.HitCount[i] / series.Requests[i] * 100
                : 0;
            hitRate.Add(StatsFormat.RoundFloat(value, 2));
            status4xx.Add(series.Status4xx[i]);
            status5xx.Add(series.Status5xx[i]);
        }

        return ServiceResult<StatQualityResultDto>.Ok(new StatQualityResultDto
        {
            XAxis = series.XAxis,
            HitRate = hitRate,
            Status4xx = status4xx,
            Status5xx = status5xx
        });
    }

    public async Task<ServiceResult<StatOriginResultDto>> GetOriginAsync(
        StatsRange range,
        AccessScope scope,
        CancellationToken cancellationToken)
    {
        var hostFilter = await _hostFilterResolver.ResolveAsync(scope, cancellationToken);
        if (!scope.IsAdmin && hostFilter.Empty)
        {
            return ServiceResult<StatOriginResultDto>.Ok(new StatOriginResultDto());
        }

        var buckets = await _accessStatsService.QueryBucketsAsync(range, hostFilter, cancellationToken);
        var series = _accessStatsService.BuildSeries(range, buckets);

        var bandwidth = new List<double>();
        var traffic = new List<double>();
        for (var i = 0; i < series.OriginBytes.Count; i++)
        {
            bandwidth.Add(StatsFormat.RoundFloat(StatsFormat.BytesToMbps(series.OriginBytes[i], range.Bucket), 2));
            traffic.Add(StatsFormat.RoundFloat(StatsFormat.BytesToMB(series.OriginBytes[i]), 2));
        }

        return ServiceResult<StatOriginResultDto>.Ok(new StatOriginResultDto
        {
            XAxis = series.XAxis,
            OriginBandwidth = bandwidth,
            OriginTraffic = traffic
        });
    }

    public Task<ServiceResult<StatNodeTrafficDto>> GetNodeTrafficAsync(string? window, CancellationToken cancellationToken)
    {
        return GetNodeTrafficCoreAsync(window, cancellationToken);
    }

    public Task<ServiceResult<StatNodeRankingResultDto>> GetNodeRankingAsync(string? metric, string? window, CancellationToken cancellationToken)
    {
        return GetNodeRankingCoreAsync(metric, window, cancellationToken);
    }

    public Task<ServiceResult<StatNodeMetricsResultDto>> GetNodeMetricsAsync(
        string? metric,
        string? window,
        string? startRaw,
        string? endRaw,
        CancellationToken cancellationToken)
    {
        return GetNodeMetricsCoreAsync(metric, window, startRaw, endRaw, cancellationToken);
    }

    public async Task<ServiceResult<UsageResultDto>> GetUsageAsync(string? rangeKey, AccessScope scope, CancellationToken cancellationToken)
    {
        var range = StatsRangeResolver.Resolve(rangeKey, null, null, DateTime.Now);
        var hostFilter = await _hostFilterResolver.ResolveAsync(scope, cancellationToken);
        if (hostFilter.Empty)
        {
            return ServiceResult<UsageResultDto>.Ok(new UsageResultDto
            {
                XAxis = Array.Empty<string>(),
                Values = Array.Empty<double>(),
                List = Array.Empty<UsagePointDto>(),
                Total = 0,
                Avg = 0,
                Peak = 0,
                Unit = "MB"
            });
        }

        var buckets = await _accessStatsService.QueryBucketsAsync(range, hostFilter, cancellationToken);
        var series = _accessStatsService.BuildSeries(range, buckets);
        var totals = await _accessStatsService.QueryTotalsAsync(range.Start, range.End, hostFilter, cancellationToken);

        var unit = "MB";
        var divider = 1024.0 * 1024.0;
        if (totals.Bytes >= 1024UL * 1024UL * 1024UL)
        {
            unit = "GB";
            divider = 1024.0 * 1024.0 * 1024.0;
        }

        var values = new List<double>();
        var list = new List<UsagePointDto>();
        double total = 0;
        double peak = 0;
        for (var i = 0; i < series.Bytes.Count; i++)
        {
            var val = series.Bytes[i] / divider;
            val = StatsFormat.RoundFloat(val, 2);
            values.Add(val);
            list.Add(new UsagePointDto { Time = series.XAxis[i], Value = val });
            total += val;
            if (val > peak)
            {
                peak = val;
            }
        }

        var avg = values.Count > 0 ? StatsFormat.RoundFloat(total / values.Count, 2) : 0;

        var result = new UsageResultDto
        {
            XAxis = series.XAxis,
            Values = values,
            List = list,
            Total = StatsFormat.RoundFloat(total, 2),
            Avg = avg,
            Peak = StatsFormat.RoundFloat(peak, 2),
            Unit = unit
        };

        return ServiceResult<UsageResultDto>.Ok(result);
    }

    private async Task<ServiceResult<StatNodeTrafficDto>> GetNodeTrafficCoreAsync(string? window, CancellationToken cancellationToken)
    {
        var resolved = string.IsNullOrWhiteSpace(window) ? "30d" : window.Trim().ToLowerInvariant();
        var (start, end, bucket, count, labelFormat, buckets) = ResolveNodeTrafficWindow(resolved);

        var trafficByBucket = new Dictionary<DateTime, (double InBytes, double OutBytes)>();
        var cfg = ClickHouseHttpHelper.ResolveConfig(_configuration);
        if (cfg != null)
        {
            var bucketExpr = BuildBucketExpression(bucket);
            var where =
                $"metric IN ('node_network_receive_bytes_total','node_network_transmit_bytes_total')" +
                $" AND ts >= toDateTime('{start:yyyy-MM-dd HH:mm:ss}')" +
                $" AND ts <= toDateTime('{end:yyyy-MM-dd HH:mm:ss}')" +
                " AND labels NOT LIKE '%device=\"lo\"%'";

            var query =
                $"SELECT {bucketExpr} AS bucket, metric, sum(delta) AS delta_bytes " +
                "FROM (" +
                $"SELECT {bucketExpr} AS bucket, metric, labels, greatest(argMax(value, ts) - argMin(value, ts), 0) AS delta " +
                $"FROM node_metrics WHERE {where} GROUP BY bucket, metric, labels" +
                ") GROUP BY bucket, metric ORDER BY bucket FORMAT JSONEachRow";

            var rows = await ClickHouseHttpHelper.QueryRowsAsync(cfg, query, cancellationToken);
            if (rows != null && rows.Length > 0)
            {
                foreach (var row in rows)
                {
                    if (string.IsNullOrWhiteSpace(row))
                    {
                        continue;
                    }

                    if (!TryParseBucketRow(row, out var bucketTime, out var metric, out var value))
                    {
                        continue;
                    }

                    if (!trafficByBucket.TryGetValue(bucketTime, out var item))
                    {
                        item = (0, 0);
                    }

                    if (string.Equals(metric, "node_network_receive_bytes_total", StringComparison.OrdinalIgnoreCase))
                    {
                        item.InBytes += value;
                    }
                    else if (string.Equals(metric, "node_network_transmit_bytes_total", StringComparison.OrdinalIgnoreCase))
                    {
                        item.OutBytes += value;
                    }

                    trafficByBucket[bucketTime] = item;
                }
            }
        }

        var xAxis = new List<string>(count);
        var inTraffic = new List<double>(count);
        var outTraffic = new List<double>(count);
        foreach (var bucketTime in buckets)
        {
            xAxis.Add(bucketTime.ToString(labelFormat, CultureInfo.InvariantCulture));
            if (trafficByBucket.TryGetValue(bucketTime, out var item))
            {
                inTraffic.Add(StatsFormat.RoundFloat(item.InBytes / (1024.0 * 1024.0), 2));
                outTraffic.Add(StatsFormat.RoundFloat(item.OutBytes / (1024.0 * 1024.0), 2));
            }
            else
            {
                inTraffic.Add(0);
                outTraffic.Add(0);
            }
        }

        return ServiceResult<StatNodeTrafficDto>.Ok(new StatNodeTrafficDto
        {
            XAxis = xAxis,
            InTraffic = inTraffic,
            OutTraffic = outTraffic
        });
    }

    private async Task<ServiceResult<StatNodeRankingResultDto>> GetNodeRankingCoreAsync(string? metric, string? window, CancellationToken cancellationToken)
    {
        var resolvedMetric = string.IsNullOrWhiteSpace(metric) ? "bandwidth" : metric.Trim().ToLowerInvariant();
        var resolvedWindow = string.IsNullOrWhiteSpace(window) ? "1m" : window.Trim().ToLowerInvariant();
        var duration = ResolveNodeRankingWindow(resolvedWindow);
        var end = DateTime.Now;
        var start = end.Add(-duration);

        var cfg = ClickHouseHttpHelper.ResolveConfig(_configuration);
        if (cfg == null)
        {
            return ServiceResult<StatNodeRankingResultDto>.Ok(new StatNodeRankingResultDto());
        }

        var list = resolvedMetric switch
        {
            "connection" => await QueryNodeRankingGaugeAsync(cfg, "node_netstat_Tcp_CurrEstab", " conn", "connection", start, end, cancellationToken),
            "load" => await QueryNodeRankingGaugeAsync(cfg, "node_load1", string.Empty, "load", start, end, cancellationToken),
            "disk" => await QueryNodeRankingDiskAsync(cfg, start, end, cancellationToken),
            _ => await QueryNodeRankingBandwidthAsync(cfg, duration, start, end, cancellationToken)
        };

        return ServiceResult<StatNodeRankingResultDto>.Ok(new StatNodeRankingResultDto { List = list });
    }

    private async Task<ServiceResult<StatNodeMetricsResultDto>> GetNodeMetricsCoreAsync(
        string? metric,
        string? window,
        string? startRaw,
        string? endRaw,
        CancellationToken cancellationToken)
    {
        var resolvedMetric = string.IsNullOrWhiteSpace(metric) ? "bandwidth" : metric.Trim().ToLowerInvariant();
        var resolvedWindow = string.IsNullOrWhiteSpace(window) ? "1h" : window.Trim().ToLowerInvariant();

        if (!TryResolveNodeMetricsRange(resolvedWindow, startRaw, endRaw, out var range))
        {
            return ServiceResult<StatNodeMetricsResultDto>.Ok(new StatNodeMetricsResultDto());
        }

        var cfg = ClickHouseHttpHelper.ResolveConfig(_configuration);
        if (cfg == null)
        {
            return ServiceResult<StatNodeMetricsResultDto>.Ok(new StatNodeMetricsResultDto());
        }

        var points = resolvedMetric switch
        {
            "connection" => await QueryNodeMetricGaugeAsync(cfg, "node_netstat_Tcp_CurrEstab", range, cancellationToken),
            "load" => await QueryNodeMetricGaugeAsync(cfg, "node_load1", range, cancellationToken),
            "disk" => await QueryNodeMetricDiskAsync(cfg, range, cancellationToken),
            _ => await QueryNodeMetricBandwidthAsync(cfg, range, cancellationToken)
        };

        return ServiceResult<StatNodeMetricsResultDto>.Ok(new StatNodeMetricsResultDto { List = points });
    }

    private async Task<IReadOnlyList<StatNodeRankingItemDto>> QueryNodeRankingBandwidthAsync(
        ClickHouseHttpConfig cfg,
        TimeSpan window,
        DateTime start,
        DateTime end,
        CancellationToken cancellationToken)
    {
        var deviceExpr = @"extract(labels, 'device=""([^""]+)""')";
        var where =
            $"metric IN ('node_network_receive_bytes_total','node_network_transmit_bytes_total')" +
            $" AND ts >= toDateTime('{start:yyyy-MM-dd HH:mm:ss}')" +
            $" AND ts <= toDateTime('{end:yyyy-MM-dd HH:mm:ss}')" +
            " AND labels NOT LIKE '%device=\"lo\"%'";

        var query =
            "SELECT node_id, device, metric, sum(delta) AS delta_bytes " +
            "FROM (" +
            $"SELECT node_id, metric, {deviceExpr} AS device, greatest(argMax(value, ts) - argMin(value, ts), 0) AS delta " +
            $"FROM node_metrics WHERE {where} GROUP BY node_id, metric, device" +
            ") GROUP BY node_id, device, metric FORMAT JSONEachRow";

        var rows = await ClickHouseHttpHelper.QueryRowsAsync(cfg, query, cancellationToken);
        if (rows == null || rows.Length == 0)
        {
            return Array.Empty<StatNodeRankingItemDto>();
        }

        var map = new Dictionary<string, Dictionary<string, (double Rx, double Tx)>>(StringComparer.OrdinalIgnoreCase);
        foreach (var row in rows)
        {
            if (string.IsNullOrWhiteSpace(row))
            {
                continue;
            }

            if (!TryParseNodeMetricRow(row, out var nodeId, out var metricName, out var device, out var value))
            {
                continue;
            }

            if (string.IsNullOrWhiteSpace(nodeId))
            {
                continue;
            }

            if (!map.TryGetValue(nodeId, out var deviceMap))
            {
                deviceMap = new Dictionary<string, (double Rx, double Tx)>(StringComparer.OrdinalIgnoreCase);
                map[nodeId] = deviceMap;
            }

            device = string.IsNullOrWhiteSpace(device) ? "-" : device;
            if (!deviceMap.TryGetValue(device, out var stats))
            {
                stats = (0, 0);
            }

            if (string.Equals(metricName, "node_network_receive_bytes_total", StringComparison.OrdinalIgnoreCase))
            {
                stats.Rx += value;
            }
            else if (string.Equals(metricName, "node_network_transmit_bytes_total", StringComparison.OrdinalIgnoreCase))
            {
                stats.Tx += value;
            }

            deviceMap[device] = stats;
        }

        var ranked = new List<(string NodeId, string Device, double Rx, double Tx, double Total)>();
        foreach (var (nodeId, deviceMap) in map)
        {
            var bestDevice = "-";
            var bestRx = 0d;
            var bestTx = 0d;
            var bestTotal = -1d;
            foreach (var (device, stats) in deviceMap)
            {
                var total = stats.Rx + stats.Tx;
                if (total > bestTotal)
                {
                    bestTotal = total;
                    bestDevice = device;
                    bestRx = stats.Rx;
                    bestTx = stats.Tx;
                }
            }

            if (bestTotal >= 0)
            {
                ranked.Add((nodeId, bestDevice, bestRx, bestTx, bestTotal));
            }
        }

        var top = ranked
            .OrderByDescending(item => item.Total)
            .ThenBy(item => item.NodeId, StringComparer.OrdinalIgnoreCase)
            .Take(10)
            .ToList();

        var list = new List<StatNodeRankingItemDto>(top.Count);
        var rank = 1;
        foreach (var item in top)
        {
            var outMbps = StatsFormat.BytesToMbps((ulong)Math.Max(0, item.Tx), window);
            var inMbps = StatsFormat.BytesToMbps((ulong)Math.Max(0, item.Rx), window);
            list.Add(new StatNodeRankingItemDto
            {
                Rank = rank++,
                Node = item.NodeId,
                Nic = item.Device,
                Out = StatsFormat.RoundFloat(outMbps, 1).ToString("F1", CultureInfo.InvariantCulture) + " Mbps",
                In = StatsFormat.RoundFloat(inMbps, 1).ToString("F1", CultureInfo.InvariantCulture) + " Mbps"
            });
        }

        return list;
    }

    private async Task<IReadOnlyList<StatNodeRankingItemDto>> QueryNodeRankingGaugeAsync(
        ClickHouseHttpConfig cfg,
        string metric,
        string unit,
        string metricType,
        DateTime start,
        DateTime end,
        CancellationToken cancellationToken)
    {
        var where =
            $"metric = {ClickHouseHttpHelper.QuoteString(metric)}" +
            $" AND ts >= toDateTime('{start:yyyy-MM-dd HH:mm:ss}')" +
            $" AND ts <= toDateTime('{end:yyyy-MM-dd HH:mm:ss}')";

        var query =
            "SELECT node_id, argMax(value, ts) AS value " +
            $"FROM node_metrics WHERE {where} GROUP BY node_id FORMAT JSONEachRow";

        var rows = await ClickHouseHttpHelper.QueryRowsAsync(cfg, query, cancellationToken);
        if (rows == null || rows.Length == 0)
        {
            return Array.Empty<StatNodeRankingItemDto>();
        }

        var list = new List<(string NodeId, double Value)>();
        foreach (var row in rows)
        {
            if (string.IsNullOrWhiteSpace(row))
            {
                continue;
            }

            if (!TryParseNodeGaugeRow(row, out var nodeId, out var value))
            {
                continue;
            }

            list.Add((nodeId, value));
        }

        var ranked = list
            .OrderByDescending(item => item.Value)
            .ThenBy(item => item.NodeId, StringComparer.OrdinalIgnoreCase)
            .Take(10)
            .ToList();

        var result = new List<StatNodeRankingItemDto>(ranked.Count);
        var rank = 1;
        foreach (var item in ranked)
        {
            var formatted = metricType switch
            {
                "connection" => ((int)Math.Round(item.Value)).ToString(CultureInfo.InvariantCulture),
                "disk" => ((int)Math.Round(item.Value)).ToString(CultureInfo.InvariantCulture),
                "load" => item.Value.ToString("F2", CultureInfo.InvariantCulture),
                _ => item.Value.ToString("F1", CultureInfo.InvariantCulture)
            };

            result.Add(new StatNodeRankingItemDto
            {
                Rank = rank++,
                Node = item.NodeId,
                Nic = "-",
                Out = formatted + unit,
                In = formatted + unit
            });
        }

        return result;
    }

    private async Task<IReadOnlyList<StatNodeRankingItemDto>> QueryNodeRankingDiskAsync(
        ClickHouseHttpConfig cfg,
        DateTime start,
        DateTime end,
        CancellationToken cancellationToken)
    {
        var where =
            "metric IN ('node_filesystem_size_bytes','node_filesystem_avail_bytes')" +
            $" AND ts >= toDateTime('{start:yyyy-MM-dd HH:mm:ss}')" +
            $" AND ts <= toDateTime('{end:yyyy-MM-dd HH:mm:ss}')" +
            " AND labels NOT LIKE '%fstype=\"tmpfs\"%'" +
            " AND labels NOT LIKE '%fstype=\"overlay\"%'" +
            " AND labels NOT LIKE '%fstype=\"squashfs\"%'" +
            " AND labels NOT LIKE '%fstype=\"nsfs\"%'";

        var query =
            "SELECT node_id, sumIf(val, metric='node_filesystem_size_bytes') AS total, " +
            "sumIf(val, metric='node_filesystem_avail_bytes') AS avail " +
            "FROM (" +
            "SELECT node_id, metric, labels, argMax(value, ts) AS val " +
            $"FROM node_metrics WHERE {where} GROUP BY node_id, metric, labels" +
            ") GROUP BY node_id FORMAT JSONEachRow";

        var rows = await ClickHouseHttpHelper.QueryRowsAsync(cfg, query, cancellationToken);
        if (rows == null || rows.Length == 0)
        {
            return Array.Empty<StatNodeRankingItemDto>();
        }

        var list = new List<(string NodeId, double Usage)>();
        foreach (var row in rows)
        {
            if (string.IsNullOrWhiteSpace(row))
            {
                continue;
            }

            if (!TryParseDiskUsageRow(row, out var nodeId, out var usage))
            {
                continue;
            }

            list.Add((nodeId, usage * 100));
        }

        var ranked = list
            .OrderByDescending(item => item.Usage)
            .ThenBy(item => item.NodeId, StringComparer.OrdinalIgnoreCase)
            .Take(10)
            .ToList();

        var result = new List<StatNodeRankingItemDto>(ranked.Count);
        var rank = 1;
        foreach (var item in ranked)
        {
            var value = Math.Round(item.Usage);
            var formatted = value.ToString(CultureInfo.InvariantCulture);
            result.Add(new StatNodeRankingItemDto
            {
                Rank = rank++,
                Node = item.NodeId,
                Nic = "-",
                Out = formatted + "%",
                In = formatted + "%"
            });
        }

        return result;
    }

    private async Task<IReadOnlyList<StatNodeMetricPointDto>> QueryNodeMetricBandwidthAsync(
        ClickHouseHttpConfig cfg,
        NodeMetricRange range,
        CancellationToken cancellationToken)
    {
        var bucketExpr = BuildBucketExpression(range.Bucket);
        var where =
            $"metric IN ('node_network_receive_bytes_total','node_network_transmit_bytes_total')" +
            $" AND ts >= toDateTime('{range.Start:yyyy-MM-dd HH:mm:ss}')" +
            $" AND ts <= toDateTime('{range.End:yyyy-MM-dd HH:mm:ss}')" +
            " AND labels NOT LIKE '%device=\"lo\"%'";

        var query =
            $"SELECT {bucketExpr} AS bucket, metric, sum(delta) AS delta_bytes " +
            "FROM (" +
            $"SELECT {bucketExpr} AS bucket, metric, labels, greatest(argMax(value, ts) - argMin(value, ts), 0) AS delta " +
            $"FROM node_metrics WHERE {where} GROUP BY bucket, metric, labels" +
            ") GROUP BY bucket, metric ORDER BY bucket FORMAT JSONEachRow";

        var rows = await ClickHouseHttpHelper.QueryRowsAsync(cfg, query, cancellationToken);
        var values = new Dictionary<DateTime, double>();
        if (rows != null && rows.Length > 0)
        {
            foreach (var row in rows)
            {
                if (string.IsNullOrWhiteSpace(row))
                {
                    continue;
                }

                if (!TryParseBucketRow(row, out var bucketTime, out var metricName, out var value))
                {
                    continue;
                }

                if (!values.TryGetValue(bucketTime, out var current))
                {
                    current = 0;
                }

                if (string.Equals(metricName, "node_network_receive_bytes_total", StringComparison.OrdinalIgnoreCase) ||
                    string.Equals(metricName, "node_network_transmit_bytes_total", StringComparison.OrdinalIgnoreCase))
                {
                    current += value;
                }

                values[bucketTime] = current;
            }
        }

        var list = new List<StatNodeMetricPointDto>(range.Count);
        foreach (var bucketTime in range.Buckets)
        {
            values.TryGetValue(bucketTime, out var bytes);
            var mbps = StatsFormat.BytesToMbps((ulong)Math.Max(0, bytes), range.Bucket);
            list.Add(new StatNodeMetricPointDto
            {
                Time = bucketTime.ToString(range.LabelFormat, CultureInfo.InvariantCulture),
                Value = StatsFormat.RoundFloat(mbps, 2)
            });
        }

        return list;
    }

    private async Task<IReadOnlyList<StatNodeMetricPointDto>> QueryNodeMetricGaugeAsync(
        ClickHouseHttpConfig cfg,
        string metric,
        NodeMetricRange range,
        CancellationToken cancellationToken)
    {
        var bucketExpr = BuildBucketExpression(range.Bucket);
        var where =
            $"metric = {ClickHouseHttpHelper.QuoteString(metric)}" +
            $" AND ts >= toDateTime('{range.Start:yyyy-MM-dd HH:mm:ss}')" +
            $" AND ts <= toDateTime('{range.End:yyyy-MM-dd HH:mm:ss}')";

        var query =
            $"SELECT {bucketExpr} AS bucket, avg(value) AS value " +
            $"FROM node_metrics WHERE {where} GROUP BY bucket ORDER BY bucket FORMAT JSONEachRow";

        var rows = await ClickHouseHttpHelper.QueryRowsAsync(cfg, query, cancellationToken);
        var values = new Dictionary<DateTime, double>();
        if (rows != null && rows.Length > 0)
        {
            foreach (var row in rows)
            {
                if (string.IsNullOrWhiteSpace(row))
                {
                    continue;
                }

                if (!TryParseBucketValueRow(row, out var bucketTime, out var value))
                {
                    continue;
                }

                values[bucketTime] = value;
            }
        }

        var list = new List<StatNodeMetricPointDto>(range.Count);
        foreach (var bucketTime in range.Buckets)
        {
            values.TryGetValue(bucketTime, out var value);
            list.Add(new StatNodeMetricPointDto
            {
                Time = bucketTime.ToString(range.LabelFormat, CultureInfo.InvariantCulture),
                Value = StatsFormat.RoundFloat(value, 2)
            });
        }

        return list;
    }

    private async Task<IReadOnlyList<StatNodeMetricPointDto>> QueryNodeMetricDiskAsync(
        ClickHouseHttpConfig cfg,
        NodeMetricRange range,
        CancellationToken cancellationToken)
    {
        var bucketExpr = BuildBucketExpression(range.Bucket);
        var where =
            "metric IN ('node_filesystem_size_bytes','node_filesystem_avail_bytes')" +
            $" AND ts >= toDateTime('{range.Start:yyyy-MM-dd HH:mm:ss}')" +
            $" AND ts <= toDateTime('{range.End:yyyy-MM-dd HH:mm:ss}')" +
            " AND labels NOT LIKE '%fstype=\"tmpfs\"%'" +
            " AND labels NOT LIKE '%fstype=\"overlay\"%'" +
            " AND labels NOT LIKE '%fstype=\"squashfs\"%'" +
            " AND labels NOT LIKE '%fstype=\"nsfs\"%'";

        var query =
            $"SELECT {bucketExpr} AS bucket, " +
            "sumIf(val, metric='node_filesystem_size_bytes') AS total, " +
            "sumIf(val, metric='node_filesystem_avail_bytes') AS avail " +
            "FROM (" +
            $"SELECT {bucketExpr} AS bucket, metric, labels, argMax(value, ts) AS val " +
            $"FROM node_metrics WHERE {where} GROUP BY bucket, metric, labels" +
            ") GROUP BY bucket ORDER BY bucket FORMAT JSONEachRow";

        var rows = await ClickHouseHttpHelper.QueryRowsAsync(cfg, query, cancellationToken);
        var values = new Dictionary<DateTime, double>();
        if (rows != null && rows.Length > 0)
        {
            foreach (var row in rows)
            {
                if (string.IsNullOrWhiteSpace(row))
                {
                    continue;
                }

                if (!TryParseDiskBucketRow(row, out var bucketTime, out var usage))
                {
                    continue;
                }

                values[bucketTime] = usage * 100;
            }
        }

        var list = new List<StatNodeMetricPointDto>(range.Count);
        foreach (var bucketTime in range.Buckets)
        {
            values.TryGetValue(bucketTime, out var value);
            list.Add(new StatNodeMetricPointDto
            {
                Time = bucketTime.ToString(range.LabelFormat, CultureInfo.InvariantCulture),
                Value = StatsFormat.RoundFloat(value, 2)
            });
        }

        return list;
    }

    private static string BuildBucketExpression(TimeSpan bucket)
    {
        if (bucket >= TimeSpan.FromDays(1))
        {
            return "toStartOfDay(ts)";
        }

        var seconds = (int)bucket.TotalSeconds;
        if (seconds <= 0)
        {
            seconds = 60;
        }

        return $"toStartOfInterval(ts, INTERVAL {seconds} SECOND)";
    }

    private static (DateTime Start, DateTime End, TimeSpan Bucket, int Count, string LabelFormat, List<DateTime> Buckets)
        ResolveNodeTrafficWindow(string resolved)
    {
        var count = 30;
        var labelFormat = "yyyy-MM-dd";
        var bucket = TimeSpan.FromDays(1);
        switch (resolved)
        {
            case "1d":
                count = 24;
                labelFormat = "HH:mm";
                bucket = TimeSpan.FromHours(1);
                break;
            case "7d":
                count = 7;
                labelFormat = "yyyy-MM-dd";
                bucket = TimeSpan.FromDays(1);
                break;
            case "30d":
                count = 30;
                labelFormat = "yyyy-MM-dd";
                bucket = TimeSpan.FromDays(1);
                break;
            case "custom":
                count = 12;
                labelFormat = "yyyy-MM-dd";
                bucket = TimeSpan.FromDays(1);
                break;
        }

        var end = DateTime.Now;
        var start = resolved == "1d" ? end.AddHours(-count) : end.AddDays(-count);
        var startAligned = AlignToBucket(start, bucket);
        var buckets = new List<DateTime>(count);
        var current = startAligned;
        for (var i = 0; i < count; i++)
        {
            buckets.Add(current);
            current = current.Add(bucket);
        }

        return (startAligned, end, bucket, count, labelFormat, buckets);
    }

    private static TimeSpan ResolveNodeRankingWindow(string window)
    {
        return window switch
        {
            "5m" => TimeSpan.FromMinutes(5),
            "30m" => TimeSpan.FromMinutes(30),
            "1h" => TimeSpan.FromHours(1),
            _ => TimeSpan.FromMinutes(1)
        };
    }

    private sealed record NodeMetricRange(
        DateTime Start,
        DateTime End,
        TimeSpan Bucket,
        int Count,
        string LabelFormat,
        List<DateTime> Buckets);

    private static bool TryResolveNodeMetricsRange(
        string window,
        string? startRaw,
        string? endRaw,
        out NodeMetricRange range)
    {
        range = null!;
        var now = DateTime.Now;
        var start = now.AddHours(-1);
        var end = now;
        var count = 12;
        var bucket = TimeSpan.FromMinutes(5);
        var labelFormat = "HH:mm";

        switch (window)
        {
            case "6h":
                start = now.AddHours(-6);
                count = 36;
                bucket = TimeSpan.FromMinutes(10);
                labelFormat = "HH:mm";
                break;
            case "12h":
                start = now.AddHours(-12);
                count = 72;
                bucket = TimeSpan.FromMinutes(10);
                labelFormat = "MM-dd HH:mm";
                break;
            case "custom":
                if (!DateTime.TryParseExact(startRaw, TimeLayout, null, DateTimeStyles.None, out var startParsed) ||
                    !DateTime.TryParseExact(endRaw, TimeLayout, null, DateTimeStyles.None, out var endParsed) ||
                    endParsed < startParsed)
                {
                    return false;
                }
                start = startParsed;
                end = endParsed;
                var total = end - start;
                if (total <= TimeSpan.Zero)
                {
                    return false;
                }
                count = 60;
                if (total < TimeSpan.FromMinutes(60))
                {
                    count = (int)Math.Max(10, total.TotalMinutes);
                }
                if (count > 200)
                {
                    count = 200;
                }
                bucket = TimeSpan.FromTicks(total.Ticks / count);
                labelFormat = "yyyy-MM-dd HH:mm";
                break;
        }

        var startAligned = AlignToBucket(start, bucket);
        var buckets = new List<DateTime>(count);
        var current = startAligned;
        for (var i = 0; i < count && current <= end; i++)
        {
            buckets.Add(current);
            current = current.Add(bucket);
        }

        range = new NodeMetricRange(startAligned, end, bucket, buckets.Count, labelFormat, buckets);
        return true;
    }

    private static DateTime AlignToBucket(DateTime value, TimeSpan bucket)
    {
        if (bucket <= TimeSpan.Zero)
        {
            return value;
        }

        if (bucket >= TimeSpan.FromDays(1))
        {
            return value.Date;
        }

        var ticks = value.Ticks - (value.Ticks % bucket.Ticks);
        return new DateTime(ticks, value.Kind);
    }

    private static bool TryParseBucketRow(string row, out DateTime bucket, out string metric, out double value)
    {
        bucket = default;
        metric = string.Empty;
        value = 0;

        try
        {
            using var doc = JsonDocument.Parse(row);
            var root = doc.RootElement;
            var bucketRaw = ReadString(root, "bucket");
            if (!DateTime.TryParseExact(bucketRaw, TimeLayout, null, DateTimeStyles.None, out bucket))
            {
                return false;
            }

            metric = ReadString(root, "metric");
            value = ReadDouble(root, "delta_bytes");
            if (value < 0)
            {
                value = 0;
            }

            return true;
        }
        catch
        {
            return false;
        }
    }

    private static bool TryParseBucketValueRow(string row, out DateTime bucket, out double value)
    {
        bucket = default;
        value = 0;
        try
        {
            using var doc = JsonDocument.Parse(row);
            var root = doc.RootElement;
            var bucketRaw = ReadString(root, "bucket");
            if (!DateTime.TryParseExact(bucketRaw, TimeLayout, null, DateTimeStyles.None, out bucket))
            {
                return false;
            }
            value = ReadDouble(root, "value");
            return true;
        }
        catch
        {
            return false;
        }
    }

    private static bool TryParseNodeMetricRow(string row, out string nodeId, out string metric, out string device, out double value)
    {
        nodeId = string.Empty;
        metric = string.Empty;
        device = string.Empty;
        value = 0;
        try
        {
            using var doc = JsonDocument.Parse(row);
            var root = doc.RootElement;
            nodeId = ReadString(root, "node_id");
            metric = ReadString(root, "metric");
            device = ReadString(root, "device");
            value = ReadDouble(root, "delta_bytes");
            if (value < 0)
            {
                value = 0;
            }
            return true;
        }
        catch
        {
            return false;
        }
    }

    private static bool TryParseNodeGaugeRow(string row, out string nodeId, out double value)
    {
        nodeId = string.Empty;
        value = 0;
        try
        {
            using var doc = JsonDocument.Parse(row);
            var root = doc.RootElement;
            nodeId = ReadString(root, "node_id");
            value = ReadDouble(root, "value");
            return !string.IsNullOrWhiteSpace(nodeId);
        }
        catch
        {
            return false;
        }
    }

    private static bool TryParseDiskUsageRow(string row, out string nodeId, out double usage)
    {
        nodeId = string.Empty;
        usage = 0;
        try
        {
            using var doc = JsonDocument.Parse(row);
            var root = doc.RootElement;
            nodeId = ReadString(root, "node_id");
            var total = ReadDouble(root, "total");
            var avail = ReadDouble(root, "avail");
            if (total <= 0)
            {
                return false;
            }
            usage = 1 - (avail / total);
            if (usage < 0)
            {
                usage = 0;
            }
            if (usage > 1)
            {
                usage = 1;
            }
            return !string.IsNullOrWhiteSpace(nodeId);
        }
        catch
        {
            return false;
        }
    }

    private static bool TryParseDiskBucketRow(string row, out DateTime bucket, out double usage)
    {
        bucket = default;
        usage = 0;
        try
        {
            using var doc = JsonDocument.Parse(row);
            var root = doc.RootElement;
            var bucketRaw = ReadString(root, "bucket");
            if (!DateTime.TryParseExact(bucketRaw, TimeLayout, null, DateTimeStyles.None, out bucket))
            {
                return false;
            }
            var total = ReadDouble(root, "total");
            var avail = ReadDouble(root, "avail");
            if (total <= 0)
            {
                return false;
            }
            usage = 1 - (avail / total);
            if (usage < 0)
            {
                usage = 0;
            }
            if (usage > 1)
            {
                usage = 1;
            }
            return true;
        }
        catch
        {
            return false;
        }
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

    private static double ReadDouble(JsonElement root, string key)
    {
        if (!root.TryGetProperty(key, out var value))
        {
            return 0;
        }

        if (value.ValueKind == JsonValueKind.Number)
        {
            if (value.TryGetDouble(out var parsed))
            {
                return parsed;
            }
        }

        if (value.ValueKind == JsonValueKind.String &&
            double.TryParse(value.GetString(), NumberStyles.Float, CultureInfo.InvariantCulture, out var parsedString))
        {
            return parsedString;
        }

        return 0;
    }



    private async Task<int> ResolveRankSizeAsync(CancellationToken cancellationToken)
    {
        var cfg = await _systemConfigService.LoadSystemConfigAsync(cancellationToken);
        if (!cfg.TryGetValue("res_rank_size", out var raw))
        {
            return 100;
        }

        raw = raw?.Trim();
        if (!int.TryParse(raw, out var size) || size <= 0)
        {
            return 100;
        }

        return size;
    }
}
