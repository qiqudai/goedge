using System.Globalization;
using System.Net;
using System.Security.Cryptography;
using System.Text;
using System.Text.Json;

namespace Cnn.Api.Services.Common.Dns.Providers;

internal sealed class AliyunDnsProvider : IDnsRecordProvider
{
    private readonly string _accessKeyId;
    private readonly string _accessKeySecret;

    private AliyunDnsProvider(string accessKeyId, string accessKeySecret)
    {
        _accessKeyId = accessKeyId;
        _accessKeySecret = accessKeySecret;
    }

    public static IDnsRecordProvider? TryCreate(string credentials)
    {
        if (string.IsNullOrWhiteSpace(credentials))
        {
            return null;
        }

        try
        {
            using var doc = JsonDocument.Parse(credentials);
            var root = doc.RootElement;
            var id = root.TryGetProperty("access_key_id", out var idProp) ? idProp.GetString() : null;
            var secret = root.TryGetProperty("access_key_secret", out var secProp) ? secProp.GetString() : null;
            return new AliyunDnsProvider((id ?? string.Empty).Trim(), (secret ?? string.Empty).Trim());
        }
        catch
        {
            return null;
        }
    }

    public Task<IReadOnlyList<DnsRecord>> GetRecordsAsync(string domain)
    {
        return Task.FromResult<IReadOnlyList<DnsRecord>>(Array.Empty<DnsRecord>());
    }

    public async Task AddRecordAsync(string domain, DnsRecord record)
    {
        var parameters = NewParams("AddDomainRecord");
        parameters["DomainName"] = domain;
        parameters["RR"] = record.Name;
        parameters["Type"] = record.Type;
        parameters["Value"] = record.Value;
        if (record.TTL > 0)
        {
            parameters["TTL"] = record.TTL.ToString(CultureInfo.InvariantCulture);
        }
        if (!string.IsNullOrWhiteSpace(record.Line))
        {
            parameters["Line"] = record.Line;
        }

        var body = await DoRequestAsync(parameters);
        using var doc = JsonDocument.Parse(body);
        var root = doc.RootElement;
        if (root.TryGetProperty("Code", out var codeProp))
        {
            var code = codeProp.GetString() ?? string.Empty;
            if (!string.IsNullOrWhiteSpace(code))
            {
                if (string.Equals(code, "DomainRecordDuplicate", StringComparison.OrdinalIgnoreCase))
                {
                    return;
                }

                var message = root.TryGetProperty("Message", out var msgProp) ? msgProp.GetString() : null;
                throw new InvalidOperationException($"aliyun error: {code} - {message}");
            }
        }
    }

    public async Task DeleteRecordAsync(string domain, DnsRecord record)
    {
        var recordId = await FindRecordIdAsync(domain, record);
        if (string.IsNullOrWhiteSpace(recordId))
        {
            return;
        }

        var parameters = NewParams("DeleteDomainRecord");
        parameters["RecordId"] = recordId;

        var body = await DoRequestAsync(parameters);
        using var doc = JsonDocument.Parse(body);
        var root = doc.RootElement;
        if (root.TryGetProperty("Code", out var codeProp))
        {
            var code = codeProp.GetString() ?? string.Empty;
            if (!string.IsNullOrWhiteSpace(code))
            {
                var message = root.TryGetProperty("Message", out var msgProp) ? msgProp.GetString() : null;
                throw new InvalidOperationException($"aliyun error: {code} - {message}");
            }
        }
    }

    private async Task<string> FindRecordIdAsync(string domain, DnsRecord record)
    {
        var parameters = NewParams("DescribeDomainRecords");
        parameters["DomainName"] = domain;
        parameters["RRKeyWord"] = record.Name;
        parameters["TypeKeyWord"] = record.Type;
        parameters["PageSize"] = "500";

        var body = await DoRequestAsync(parameters);
        using var doc = JsonDocument.Parse(body);
        if (!doc.RootElement.TryGetProperty("DomainRecords", out var recordsElement))
        {
            return string.Empty;
        }
        if (!recordsElement.TryGetProperty("Record", out var recordList) || recordList.ValueKind != JsonValueKind.Array)
        {
            return string.Empty;
        }

        foreach (var item in recordList.EnumerateArray())
        {
            var rr = item.TryGetProperty("RR", out var rrProp) ? rrProp.GetString() : null;
            var type = item.TryGetProperty("Type", out var typeProp) ? typeProp.GetString() : null;
            var value = item.TryGetProperty("Value", out var valueProp) ? valueProp.GetString() : null;
            if (string.Equals(rr, record.Name, StringComparison.OrdinalIgnoreCase) &&
                string.Equals(type, record.Type, StringComparison.OrdinalIgnoreCase) &&
                string.Equals(value, record.Value, StringComparison.OrdinalIgnoreCase))
            {
                return item.TryGetProperty("RecordId", out var idProp) ? (idProp.GetString() ?? string.Empty) : string.Empty;
            }
        }

        return string.Empty;
    }

    private Dictionary<string, string> NewParams(string action)
    {
        return new Dictionary<string, string>(StringComparer.Ordinal)
        {
            ["Action"] = action,
            ["Format"] = "JSON",
            ["Version"] = "2015-01-09",
            ["AccessKeyId"] = _accessKeyId,
            ["SignatureMethod"] = "HMAC-SHA1",
            ["Timestamp"] = DateTime.UtcNow.ToString("yyyy-MM-ddTHH:mm:ssZ", CultureInfo.InvariantCulture),
            ["SignatureVersion"] = "1.0",
            ["SignatureNonce"] = Guid.NewGuid().ToString()
        };
    }

    private async Task<string> DoRequestAsync(Dictionary<string, string> parameters)
    {
        var signature = Sign(parameters);
        parameters["Signature"] = signature;

        var query = await new FormUrlEncodedContent(parameters).ReadAsStringAsync();
        var url = "https://alidns.aliyuncs.com/?" + query;
        var req = new HttpRequestMessage(HttpMethod.Get, url);
        var (_, body) = await DnsHttp.SendAsync(req, TimeSpan.FromSeconds(30));
        return body;
    }

    private string Sign(Dictionary<string, string> parameters)
    {
        var keys = parameters.Keys.OrderBy(k => k, StringComparer.Ordinal).ToList();
        var canonical = new StringBuilder();
        foreach (var key in keys)
        {
            if (canonical.Length > 0)
            {
                canonical.Append('&');
            }
            canonical.Append(PercentEncode(key));
            canonical.Append('=');
            canonical.Append(PercentEncode(parameters[key]));
        }

        var stringToSign = "GET&" + PercentEncode("/") + "&" + PercentEncode(canonical.ToString());
        using var hmac = new HMACSHA1(Encoding.UTF8.GetBytes(_accessKeySecret + "&"));
        var hash = hmac.ComputeHash(Encoding.UTF8.GetBytes(stringToSign));
        return Convert.ToBase64String(hash);
    }

    private static string PercentEncode(string value)
    {
        var encoded = Uri.EscapeDataString(value ?? string.Empty);
        encoded = encoded.Replace("+", "%20", StringComparison.Ordinal)
            .Replace("*", "%2A", StringComparison.Ordinal)
            .Replace("%7E", "~", StringComparison.OrdinalIgnoreCase);
        return encoded;
    }
}
