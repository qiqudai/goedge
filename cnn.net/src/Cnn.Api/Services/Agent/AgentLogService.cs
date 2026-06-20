using System.Globalization;
using System.IO;
using System.Net.Http.Headers;
using System.Text;
using System.Text.Json;
using System.Text.Json.Serialization;
using Cnn.Api.Services.Stats;
using Microsoft.Extensions.Configuration;

namespace Cnn.Api.Services.Agent;

public interface IAgentLogService
{
    Task<int> InsertAccessLogsAsync(string? nodeId, string? nodeIp, IReadOnlyList<string> lines, CancellationToken cancellationToken);
    Task<int> InsertStreamLogsAsync(string? nodeId, string? nodeIp, IReadOnlyList<string> lines, CancellationToken cancellationToken);
    Task<int> InsertMetricsAsync(string? nodeId, string? nodeIp, string? content, CancellationToken cancellationToken);
    Task<int> InsertEventLogsAsync(string? nodeId, string? nodeIp, string? eventType, IReadOnlyList<string> payloads, CancellationToken cancellationToken);
}

public sealed class AgentLogService : IAgentLogService
{
    private static readonly HttpClient HttpClient = new()
    {
        Timeout = TimeSpan.FromSeconds(5)
    };

    private static readonly JsonSerializerOptions JsonOptions = new(JsonSerializerDefaults.Web)
    {
        PropertyNameCaseInsensitive = true
    };

    private readonly IConfiguration _configuration;

    public AgentLogService(IConfiguration configuration)
    {
        _configuration = configuration;
    }

    public async Task<int> InsertAccessLogsAsync(string? nodeId, string? nodeIp, IReadOnlyList<string> lines, CancellationToken cancellationToken)
    {
        if (lines == null || lines.Count == 0)
        {
            return 0;
        }

        var cfg = ClickHouseHttpHelper.ResolveConfig(_configuration);
        if (cfg == null)
        {
            return 0;
        }

        var rows = new List<Dictionary<string, object?>>(lines.Count);
        foreach (var line in lines)
        {
            var rawLine = line?.Trim();
            if (string.IsNullOrWhiteSpace(rawLine))
            {
                continue;
            }

            RawAccessLog? raw;
            try
            {
                raw = JsonSerializer.Deserialize<RawAccessLog>(rawLine!, JsonOptions);
            }
            catch
            {
                continue;
            }
            if (raw == null)
            {
                continue;
            }

            var (method, uri) = ParseRequest(raw.Request);
            var ts = FormatTime(ParseIsoTime(raw.TimeIso8601));
            var upstreamRT = ParseFloatFirst(raw.UpstreamResponseTime);

            rows.Add(new Dictionary<string, object?>
            {
                ["ts"] = ts,
                ["node_id"] = nodeId ?? string.Empty,
                ["node_ip"] = nodeIp ?? string.Empty,
                ["remote_addr"] = raw.RemoteAddr,
                ["client_country"] = raw.ClientCountry,
                ["client_province"] = raw.ClientProvince,
                ["host"] = raw.Host,
                ["method"] = method,
                ["uri"] = uri,
                ["status"] = raw.Status,
                ["bytes"] = raw.BodyBytesSent,
                ["request_time"] = raw.RequestTime,
                ["upstream_addr"] = raw.UpstreamAddr,
                ["upstream_response_time"] = upstreamRT,
                ["upstream_cache_status"] = raw.UpstreamCacheStatus,
                ["http_referer"] = raw.HttpReferer,
                ["http_user_agent"] = raw.HttpUserAgent,
                ["scheme"] = raw.Scheme,
                ["ssl_protocol"] = raw.SslProtocol,
                ["ssl_cipher"] = raw.SslCipher,
                ["raw"] = rawLine
            });
        }

        return await InsertRowsAsync(cfg, "node_access_logs", rows, cancellationToken);
    }

    public async Task<int> InsertMetricsAsync(string? nodeId, string? nodeIp, string? content, CancellationToken cancellationToken)
    {
        if (string.IsNullOrWhiteSpace(content))
        {
            return 0;
        }

        var cfg = ClickHouseHttpHelper.ResolveConfig(_configuration);
        if (cfg == null)
        {
            return 0;
        }

        var rows = new List<Dictionary<string, object?>>();
        var now = FormatTime(DateTime.Now);
        using var reader = new StringReader(content);
        string? line;
        while ((line = reader.ReadLine()) != null)
        {
            line = line.Trim();
            if (line.Length == 0 || line.StartsWith("#", StringComparison.Ordinal))
            {
                continue;
            }

            if (!TryParseMetricLine(line, out var metric, out var labels, out var value))
            {
                continue;
            }

            rows.Add(new Dictionary<string, object?>
            {
                ["ts"] = now,
                ["node_id"] = nodeId ?? string.Empty,
                ["node_ip"] = nodeIp ?? string.Empty,
                ["metric"] = metric,
                ["labels"] = labels,
                ["value"] = value
            });
        }

        return await InsertRowsAsync(cfg, "node_metrics", rows, cancellationToken);
    }

    public async Task<int> InsertStreamLogsAsync(string? nodeId, string? nodeIp, IReadOnlyList<string> lines, CancellationToken cancellationToken)
    {
        if (lines == null || lines.Count == 0)
        {
            return 0;
        }

        var cfg = ClickHouseHttpHelper.ResolveConfig(_configuration);
        if (cfg == null)
        {
            return 0;
        }

        var rows = new List<Dictionary<string, object?>>(lines.Count);
        foreach (var line in lines)
        {
            var rawLine = line?.Trim();
            if (string.IsNullOrWhiteSpace(rawLine))
            {
                continue;
            }

            RawStreamLog? raw;
            try
            {
                raw = JsonSerializer.Deserialize<RawStreamLog>(rawLine!, JsonOptions);
            }
            catch
            {
                continue;
            }
            if (raw == null)
            {
                continue;
            }

            rows.Add(new Dictionary<string, object?>
            {
                ["ts"] = FormatTime(ParseIsoTime(raw.TimeIso8601)),
                ["node_id"] = nodeId ?? string.Empty,
                ["node_ip"] = nodeIp ?? string.Empty,
                ["remote_addr"] = raw.RemoteAddr,
                ["server_port"] = raw.ServerPort,
                ["protocol"] = raw.Protocol,
                ["status"] = raw.Status,
                ["bytes_sent"] = raw.BytesSent,
                ["bytes_received"] = raw.BytesReceived,
                ["session_time"] = ParseFloatFirst(raw.SessionTime),
                ["upstream_addr"] = raw.UpstreamAddr,
                ["upstream_bytes_sent"] = ParseInt64First(raw.UpstreamBytesSent),
                ["upstream_bytes_received"] = ParseInt64First(raw.UpstreamBytesReceived),
                ["upstream_connect_time"] = ParseFloatFirst(raw.UpstreamConnectTime),
                ["upstream_session_time"] = ParseFloatFirst(raw.UpstreamSessionTime),
                ["raw"] = rawLine
            });
        }

        return await InsertRowsAsync(cfg, "node_stream_logs", rows, cancellationToken);
    }

    public async Task<int> InsertEventLogsAsync(string? nodeId, string? nodeIp, string? eventType, IReadOnlyList<string> payloads, CancellationToken cancellationToken)
    {
        if (payloads == null || payloads.Count == 0)
        {
            return 0;
        }

        var cfg = ClickHouseHttpHelper.ResolveConfig(_configuration);
        if (cfg == null)
        {
            return 0;
        }

        var rows = new List<Dictionary<string, object?>>(payloads.Count);
        var now = FormatTime(DateTime.Now);
        var type = string.IsNullOrWhiteSpace(eventType) ? "event" : eventType.Trim();

        foreach (var payload in payloads)
        {
            var value = payload?.Trim();
            if (string.IsNullOrWhiteSpace(value))
            {
                continue;
            }

            rows.Add(new Dictionary<string, object?>
            {
                ["ts"] = now,
                ["node_id"] = nodeId ?? string.Empty,
                ["node_ip"] = nodeIp ?? string.Empty,
                ["event_type"] = type,
                ["payload"] = value
            });
        }

        return await InsertRowsAsync(cfg, "node_events", rows, cancellationToken);
    }

    private static async Task<int> InsertRowsAsync(
        ClickHouseHttpConfig config,
        string table,
        IReadOnlyList<Dictionary<string, object?>> rows,
        CancellationToken cancellationToken)
    {
        if (rows == null || rows.Count == 0)
        {
            return 0;
        }

        var query = $"INSERT INTO {table} FORMAT JSONEachRow";
        var endpoint = new StringBuilder();
        endpoint.Append(config.BaseUrl.TrimEnd('/'));
        endpoint.Append("/?query=").Append(Uri.EscapeDataString(query));
        if (!string.IsNullOrWhiteSpace(config.Database))
        {
            endpoint.Append("&database=").Append(Uri.EscapeDataString(config.Database));
        }

        var body = new StringBuilder(rows.Count * 256);
        foreach (var row in rows)
        {
            var json = JsonSerializer.Serialize(row, JsonOptions);
            body.Append(json).Append('\n');
        }

        using var request = new HttpRequestMessage(HttpMethod.Post, endpoint.ToString())
        {
            Content = new StringContent(body.ToString(), Encoding.UTF8, "text/plain")
        };

        if (!string.IsNullOrWhiteSpace(config.User))
        {
            var token = Convert.ToBase64String(Encoding.UTF8.GetBytes($"{config.User}:{config.Pass ?? string.Empty}"));
            request.Headers.Authorization = new AuthenticationHeaderValue("Basic", token);
        }

        using var response = await HttpClient.SendAsync(request, cancellationToken);
        if (!response.IsSuccessStatusCode)
        {
            return 0;
        }

        return rows.Count;
    }

    private static (string Method, string Uri) ParseRequest(string? request)
    {
        if (string.IsNullOrWhiteSpace(request))
        {
            return (string.Empty, string.Empty);
        }

        var parts = request.Trim().Split(' ', StringSplitOptions.RemoveEmptyEntries);
        if (parts.Length >= 2)
        {
            return (parts[0], parts[1]);
        }

        return (string.Empty, string.Empty);
    }

    private static DateTime ParseIsoTime(string? value)
    {
        if (string.IsNullOrWhiteSpace(value))
        {
            return DateTime.Now;
        }

        if (DateTime.TryParse(value, out var ts))
        {
            return ts;
        }

        return DateTime.Now;
    }

    private static string FormatTime(DateTime value)
    {
        if (value == DateTime.MinValue)
        {
            value = DateTime.Now;
        }

        return value.ToString("yyyy-MM-dd HH:mm:ss");
    }

    private static double ParseFloatFirst(string? value)
    {
        if (string.IsNullOrWhiteSpace(value))
        {
            return 0;
        }

        var raw = value.Trim();
        if (raw == "-")
        {
            return 0;
        }

        var idx = raw.IndexOf(',');
        if (idx >= 0)
        {
            raw = raw.Substring(0, idx);
        }

        if (double.TryParse(raw, NumberStyles.Float, CultureInfo.InvariantCulture, out var result))
        {
            return result;
        }

        return 0;
    }

    private static long ParseInt64First(string? value)
    {
        if (string.IsNullOrWhiteSpace(value))
        {
            return 0;
        }

        var raw = value.Trim();
        if (raw == "-")
        {
            return 0;
        }

        var idx = raw.IndexOf(',');
        if (idx >= 0)
        {
            raw = raw.Substring(0, idx);
        }

        if (long.TryParse(raw, NumberStyles.Integer, CultureInfo.InvariantCulture, out var result))
        {
            return result;
        }

        if (ulong.TryParse(raw, NumberStyles.Integer, CultureInfo.InvariantCulture, out var unsigned))
        {
            return unchecked((long)unsigned);
        }

        return 0;
    }

    private static bool TryParseMetricLine(string line, out string metric, out string labels, out double value)
    {
        metric = string.Empty;
        labels = string.Empty;
        value = 0;

        var parts = line.Split(' ', StringSplitOptions.RemoveEmptyEntries);
        if (parts.Length < 2)
        {
            return false;
        }

        var metricPart = parts[0];
        var valuePart = parts[1];
        var idx = metricPart.IndexOf('{');
        if (idx >= 0)
        {
            metric = metricPart.Substring(0, idx);
            labels = metricPart.TrimEnd('}').Substring(idx + 1);
        }
        else
        {
            metric = metricPart;
            labels = string.Empty;
        }

        return double.TryParse(valuePart, NumberStyles.Float, CultureInfo.InvariantCulture, out value);
    }

    private sealed class RawAccessLog
    {
        [JsonPropertyName("time_iso8601")]
        public string? TimeIso8601 { get; set; }

        [JsonPropertyName("remote_addr")]
        public string? RemoteAddr { get; set; }

        [JsonPropertyName("client_country")]
        public string? ClientCountry { get; set; }

        [JsonPropertyName("client_province")]
        public string? ClientProvince { get; set; }

        [JsonPropertyName("host")]
        public string? Host { get; set; }

        [JsonPropertyName("request")]
        public string? Request { get; set; }

        [JsonPropertyName("status")]
        public int Status { get; set; }

        [JsonPropertyName("body_bytes_sent")]
        public long BodyBytesSent { get; set; }

        [JsonPropertyName("request_time")]
        public double RequestTime { get; set; }

        [JsonPropertyName("upstream_addr")]
        public string? UpstreamAddr { get; set; }

        [JsonPropertyName("upstream_response_time")]
        public string? UpstreamResponseTime { get; set; }

        [JsonPropertyName("upstream_cache_status")]
        public string? UpstreamCacheStatus { get; set; }

        [JsonPropertyName("http_referer")]
        public string? HttpReferer { get; set; }

        [JsonPropertyName("http_user_agent")]
        public string? HttpUserAgent { get; set; }

        [JsonPropertyName("scheme")]
        public string? Scheme { get; set; }

        [JsonPropertyName("ssl_protocol")]
        public string? SslProtocol { get; set; }

        [JsonPropertyName("ssl_cipher")]
        public string? SslCipher { get; set; }
    }

    private sealed class RawStreamLog
    {
        [JsonPropertyName("time_iso8601")]
        public string? TimeIso8601 { get; set; }

        [JsonPropertyName("remote_addr")]
        public string? RemoteAddr { get; set; }

        [JsonPropertyName("protocol")]
        public string? Protocol { get; set; }

        [JsonPropertyName("status")]
        public int Status { get; set; }

        [JsonPropertyName("bytes_sent")]
        public long BytesSent { get; set; }

        [JsonPropertyName("bytes_received")]
        public long BytesReceived { get; set; }

        [JsonPropertyName("session_time")]
        public string? SessionTime { get; set; }

        [JsonPropertyName("upstream_addr")]
        public string? UpstreamAddr { get; set; }

        [JsonPropertyName("upstream_bytes_sent")]
        public string? UpstreamBytesSent { get; set; }

        [JsonPropertyName("upstream_bytes_received")]
        public string? UpstreamBytesReceived { get; set; }

        [JsonPropertyName("upstream_connect_time")]
        public string? UpstreamConnectTime { get; set; }

        [JsonPropertyName("upstream_session_time")]
        public string? UpstreamSessionTime { get; set; }

        [JsonPropertyName("server_port")]
        public int ServerPort { get; set; }
    }
}
