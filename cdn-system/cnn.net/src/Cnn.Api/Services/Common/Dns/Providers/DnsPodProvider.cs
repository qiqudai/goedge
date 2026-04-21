using System.Security.Cryptography;
using System.Text;
using System.Text.Json;

namespace Cnn.Api.Services.Common.Dns.Providers;

internal sealed class DnsPodProvider : IDnsRecordProvider, IDnsRecordValueReplacer, IDnsLineRecordDeleter
{
    private const string InternationalRegion = "international";

    private readonly string _id;
    private readonly string _token;
    private readonly string _secretId;
    private readonly string _secretKey;
    private readonly string _apiType;
    private readonly string _region;

    private DnsPodProvider(string id, string token, string secretId, string secretKey, string apiType, string region)
    {
        _id = id;
        _token = token;
        _secretId = secretId;
        _secretKey = secretKey;
        _apiType = apiType;
        _region = region;
    }

    public static IDnsRecordProvider? TryCreate(string credentials, bool forceInternational)
    {
        if (string.IsNullOrWhiteSpace(credentials))
        {
            return null;
        }

        try
        {
            using var doc = JsonDocument.Parse(credentials);
            var root = doc.RootElement;
            var id = GetString(root, "id") ?? string.Empty;
            var token = GetString(root, "token") ?? string.Empty;
            var secretId = GetString(root, "secret_id") ?? string.Empty;
            var secretKey = GetString(root, "secret_key") ?? string.Empty;
            var apiType = GetString(root, "apiType") ?? string.Empty;
            var region = GetString(root, "region") ?? string.Empty;
            if (forceInternational)
            {
                region = InternationalRegion;
            }

            return new DnsPodProvider(id.Trim(), token.Trim(), secretId.Trim(), secretKey.Trim(), apiType.Trim(), region.Trim());
        }
        catch
        {
            return null;
        }
    }

    public async Task<IReadOnlyList<DnsRecord>> GetRecordsAsync(string domain)
    {
        return UseTc3() ? await GetRecordsTc3Async(domain) : await GetRecordsV2Async(domain);
    }

    public async Task AddRecordAsync(string domain, DnsRecord record)
    {
        if (UseTc3())
        {
            await AddRecordTc3Async(domain, record);
            return;
        }

        await AddRecordV2Async(domain, record);
    }

    public async Task DeleteRecordAsync(string domain, DnsRecord record)
    {
        if (UseTc3())
        {
            await DeleteRecordTc3Async(domain, record);
            return;
        }

        await DeleteRecordV2Async(domain, record);
    }

    public async Task ReplaceRecordValueAsync(string domain, DnsRecord record, string newValue)
    {
        if (UseTc3())
        {
            await ReplaceRecordTc3Async(domain, record, newValue);
            return;
        }

        await ReplaceRecordV2Async(domain, record, newValue);
    }

    public async Task DeleteRecordsByLineAsync(string domain, DnsRecord record)
    {
        if (UseTc3())
        {
            await DeleteRecordsByLineTc3Async(domain, record);
            return;
        }

        await DeleteRecordsByLineV2Async(domain, record);
    }

    private bool UseTc3()
    {
        if (string.Equals(_apiType, "tencentDNS", StringComparison.OrdinalIgnoreCase))
        {
            return true;
        }

        return !string.IsNullOrWhiteSpace(_secretId) && !string.IsNullOrWhiteSpace(_secretKey);
    }

    private async Task AddRecordV2Async(string domain, DnsRecord record)
    {
        if (string.IsNullOrWhiteSpace(_id) || string.IsNullOrWhiteSpace(_token))
        {
            throw new InvalidOperationException("dnspod id/token required");
        }

        var values = new Dictionary<string, string>
        {
            ["domain"] = domain,
            ["sub_domain"] = record.Name,
            ["record_type"] = record.Type,
            ["record_line"] = string.IsNullOrWhiteSpace(record.Line) ? "Default" : record.Line,
            ["value"] = record.Value,
            ["ttl"] = record.TTL.ToString()
        };
        if (record.Weight > 0)
        {
            values["weight"] = record.Weight.ToString();
        }

        var body = await SendRequestV2Async("Record.Create", values);
        var (code, message) = ParseV2Status(body);
        if (code != "1")
        {
            if (message.Contains("exists", StringComparison.OrdinalIgnoreCase) || message.Contains("already exists", StringComparison.OrdinalIgnoreCase))
            {
                return;
            }

            throw new InvalidOperationException($"dnspod error code: {code} message: {message}");
        }
    }

    private async Task ReplaceRecordV2Async(string domain, DnsRecord record, string newValue)
    {
        if (string.IsNullOrWhiteSpace(_id) || string.IsNullOrWhiteSpace(_token))
        {
            throw new InvalidOperationException("dnspod id/token required");
        }

        var recordId = await FindRecordIdByNameV2Async(domain, record);
        if (!string.IsNullOrWhiteSpace(record.Value))
        {
            var exact = await FindRecordIdV2Async(domain, record);
            if (!string.IsNullOrWhiteSpace(exact))
            {
                recordId = exact;
            }
        }

        if (string.IsNullOrWhiteSpace(recordId))
        {
            throw new InvalidOperationException("record not found");
        }

        var values = new Dictionary<string, string>
        {
            ["domain"] = domain,
            ["record_id"] = recordId,
            ["sub_domain"] = record.Name,
            ["record_type"] = record.Type,
            ["record_line"] = string.IsNullOrWhiteSpace(record.Line) ? "Default" : record.Line,
            ["value"] = newValue,
            ["ttl"] = record.TTL.ToString()
        };
        if (record.Weight > 0)
        {
            values["weight"] = record.Weight.ToString();
        }

        var body = await SendRequestV2Async("Record.Modify", values);
        var (code, message) = ParseV2Status(body);
        if (code != "1")
        {
            if (message.Contains("exists", StringComparison.OrdinalIgnoreCase) || message.Contains("already exists", StringComparison.OrdinalIgnoreCase))
            {
                return;
            }

            throw new InvalidOperationException($"dnspod error code: {code} message: {message}");
        }
    }

    private async Task DeleteRecordV2Async(string domain, DnsRecord record)
    {
        if (string.IsNullOrWhiteSpace(_id) || string.IsNullOrWhiteSpace(_token))
        {
            throw new InvalidOperationException("dnspod id/token required");
        }

        var recordId = await FindRecordIdV2Async(domain, record);
        if (string.IsNullOrWhiteSpace(recordId))
        {
            return;
        }

        var values = new Dictionary<string, string>
        {
            ["domain"] = domain,
            ["record_id"] = recordId
        };

        var body = await SendRequestV2Async("Record.Remove", values);
        var (code, _) = ParseV2Status(body);
        if (code != "1" && code != "10")
        {
            throw new InvalidOperationException($"dnspod error code: {code}");
        }
    }

    private async Task DeleteRecordsByLineV2Async(string domain, DnsRecord record)
    {
        var records = await ListRecordsV2Async(domain);
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

            var values = new Dictionary<string, string>
            {
                ["domain"] = domain,
                ["record_id"] = item.Id
            };
            await SendRequestV2Async("Record.Remove", values);
        }
    }

    private async Task<string> FindRecordIdByNameV2Async(string domain, DnsRecord record)
    {
        var records = await ListRecordsV2Async(domain);
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

            return item.Id;
        }

        return string.Empty;
    }

    private async Task<string> FindRecordIdV2Async(string domain, DnsRecord record)
    {
        var records = await ListRecordsV2Async(domain);
        foreach (var item in records)
        {
            if (!string.Equals(item.Type, record.Type, StringComparison.OrdinalIgnoreCase))
            {
                continue;
            }
            if (!string.Equals(item.Value, record.Value, StringComparison.OrdinalIgnoreCase))
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

            return item.Id;
        }

        return string.Empty;
    }

    private async Task<List<DnsRecord>> GetRecordsV2Async(string domain)
    {
        var records = await ListRecordsV2Async(domain);
        return records.Select(item => new DnsRecord
        {
            Type = item.Type,
            Name = item.Name,
            Value = item.Value,
            Line = item.Line,
            TTL = item.Ttl == 0 ? 600 : item.Ttl,
            Weight = item.Weight
        }).ToList();
    }

    private async Task<List<V2Record>> ListRecordsV2Async(string domain)
    {
        var values = new Dictionary<string, string>
        {
            ["domain"] = domain,
            ["length"] = "3000"
        };

        var body = await SendRequestV2Async("Record.List", values);
        using var doc = JsonDocument.Parse(body);
        var status = doc.RootElement.GetProperty("status");
        var code = status.GetProperty("code").GetString();
        if (code != "1")
        {
            if (code == "10")
            {
                return new List<V2Record>();
            }
            throw new InvalidOperationException($"dnspod api error code: {code}");
        }

        var result = new List<V2Record>();
        if (!doc.RootElement.TryGetProperty("records", out var recordsElement) || recordsElement.ValueKind != JsonValueKind.Array)
        {
            return result;
        }

        foreach (var item in recordsElement.EnumerateArray())
        {
            var ttlRaw = item.TryGetProperty("ttl", out var ttlProp) ? ttlProp.GetString() : null;
            var weightRaw = item.TryGetProperty("weight", out var weightProp) ? weightProp.GetString() : null;
            _ = int.TryParse(ttlRaw, out var ttl);
            _ = int.TryParse(weightRaw, out var weight);
            result.Add(new V2Record
            {
                Id = item.GetProperty("id").GetString() ?? string.Empty,
                Name = item.GetProperty("name").GetString() ?? string.Empty,
                Type = item.GetProperty("type").GetString() ?? string.Empty,
                Value = item.GetProperty("value").GetString() ?? string.Empty,
                Line = item.GetProperty("line").GetString() ?? string.Empty,
                Ttl = ttl,
                Weight = weight
            });
        }

        return result;
    }

    private async Task AddRecordTc3Async(string domain, DnsRecord record)
    {
        var line = string.IsNullOrWhiteSpace(record.Line) ? "Default" : record.Line.Trim();
        var payload = new Dictionary<string, object?>
        {
            ["Domain"] = domain,
            ["SubDomain"] = record.Name,
            ["RecordType"] = record.Type,
            ["Value"] = record.Value,
            ["TTL"] = record.TTL
        };
        SetRecordLinePayload(payload, line);
        if (record.Weight > 0)
        {
            payload["Weight"] = record.Weight;
        }

        var body = await SendRequestTc3Async("CreateRecord", payload);
        var error = ExtractTc3Error(body);
        if (error != null && !IsIgnorableTc3(error.Value.Code, error.Value.Message))
        {
            throw new InvalidOperationException($"dnspod tc3 error: {error.Value.Code} - {error.Value.Message}");
        }
    }

    private async Task ReplaceRecordTc3Async(string domain, DnsRecord record, string newValue)
    {
        var line = string.IsNullOrWhiteSpace(record.Line) ? "Default" : record.Line.Trim();
        var recordId = await FindRecordIdByNameTc3Async(domain, record);
        if (!string.IsNullOrWhiteSpace(record.Value))
        {
            var exact = await FindRecordIdTc3Async(domain, record);
            if (exact != 0)
            {
                recordId = exact;
            }
        }

        if (recordId == 0)
        {
            throw new InvalidOperationException("record not found");
        }

        var payload = new Dictionary<string, object?>
        {
            ["Domain"] = domain,
            ["RecordId"] = recordId,
            ["SubDomain"] = record.Name,
            ["RecordType"] = record.Type,
            ["Value"] = newValue,
            ["TTL"] = record.TTL
        };
        SetRecordLinePayload(payload, line);
        if (record.Weight > 0)
        {
            payload["Weight"] = record.Weight;
        }

        var body = await SendRequestTc3Async("ModifyRecord", payload);
        var error = ExtractTc3Error(body);
        if (error != null && !IsIgnorableTc3(error.Value.Code, error.Value.Message))
        {
            throw new InvalidOperationException($"dnspod tc3 error: {error.Value.Code} - {error.Value.Message}");
        }
    }

    private async Task DeleteRecordTc3Async(string domain, DnsRecord record)
    {
        var recordId = await FindRecordIdTc3Async(domain, record);
        if (recordId == 0)
        {
            return;
        }

        var payload = new Dictionary<string, object?>
        {
            ["Domain"] = domain,
            ["RecordId"] = recordId
        };

        var body = await SendRequestTc3Async("DeleteRecord", payload);
        var error = ExtractTc3Error(body);
        if (error != null && !IsIgnorableTc3(error.Value.Code, error.Value.Message))
        {
            throw new InvalidOperationException($"dnspod tc3 error: {error.Value.Code} - {error.Value.Message}");
        }
    }

    private async Task DeleteRecordsByLineTc3Async(string domain, DnsRecord record)
    {
        var payload = new Dictionary<string, object?>
        {
            ["Domain"] = domain,
            ["Subdomain"] = record.Name,
            ["RecordType"] = record.Type,
            ["Limit"] = 3000
        };

        var body = await SendRequestTc3Async("DescribeRecordList", payload);
        var error = ExtractTc3Error(body);
        if (error != null)
        {
            if (error.Value.Code == "ResourceNotFound.NoDataOfRecord")
            {
                return;
            }
            throw new InvalidOperationException($"dnspod tc3 error: {error.Value.Code}");
        }

        using var doc = JsonDocument.Parse(body);
        if (!doc.RootElement.TryGetProperty("Response", out var response) ||
            !response.TryGetProperty("RecordList", out var list) || list.ValueKind != JsonValueKind.Array)
        {
            return;
        }

        foreach (var item in list.EnumerateArray())
        {
            if (!LineMatches(item, record.Line))
            {
                continue;
            }

            var recordId = ReadUInt64Property(item, "RecordId");
            if (recordId == 0)
            {
                continue;
            }

            var delPayload = new Dictionary<string, object?>
            {
                ["Domain"] = domain,
                ["RecordId"] = recordId
            };

            await SendRequestTc3Async("DeleteRecord", delPayload);
        }
    }

    private async Task<ulong> FindRecordIdByNameTc3Async(string domain, DnsRecord record)
    {
        var payload = new Dictionary<string, object?>
        {
            ["Domain"] = domain,
            ["Subdomain"] = record.Name,
            ["RecordType"] = record.Type,
            ["Limit"] = 3000
        };

        var body = await SendRequestTc3Async("DescribeRecordList", payload);
        var error = ExtractTc3Error(body);
        if (error != null)
        {
            return 0;
        }

        using var doc = JsonDocument.Parse(body);
        if (!doc.RootElement.TryGetProperty("Response", out var response) ||
            !response.TryGetProperty("RecordList", out var list) || list.ValueKind != JsonValueKind.Array)
        {
            return 0;
        }

        foreach (var item in list.EnumerateArray())
        {
            var type = item.TryGetProperty("Type", out var typeProp) ? typeProp.GetString() : null;
            if (!string.Equals(type, record.Type, StringComparison.OrdinalIgnoreCase))
            {
                continue;
            }
            if (!LineMatches(item, record.Line))
            {
                continue;
            }

            if (item.TryGetProperty("RecordId", out var idProp) && idProp.TryGetUInt64(out var id))
            {
                return id;
            }
        }

        return 0;
    }

    private async Task<ulong> FindRecordIdTc3Async(string domain, DnsRecord record)
    {
        var payload = new Dictionary<string, object?>
        {
            ["Domain"] = domain,
            ["Subdomain"] = record.Name,
            ["RecordType"] = record.Type,
            ["Limit"] = 3000
        };

        var body = await SendRequestTc3Async("DescribeRecordList", payload);
        var error = ExtractTc3Error(body);
        if (error != null)
        {
            return 0;
        }

        using var doc = JsonDocument.Parse(body);
        if (!doc.RootElement.TryGetProperty("Response", out var response) ||
            !response.TryGetProperty("RecordList", out var list) || list.ValueKind != JsonValueKind.Array)
        {
            return 0;
        }

        foreach (var item in list.EnumerateArray())
        {
            var type = item.TryGetProperty("Type", out var typeProp) ? typeProp.GetString() : null;
            var value = item.TryGetProperty("Value", out var valueProp) ? valueProp.GetString() : null;
            if (!string.Equals(type, record.Type, StringComparison.OrdinalIgnoreCase))
            {
                continue;
            }
            if (!string.Equals(value, record.Value, StringComparison.OrdinalIgnoreCase))
            {
                continue;
            }
            if (!LineMatches(item, record.Line))
            {
                continue;
            }

            if (item.TryGetProperty("RecordId", out var idProp) && idProp.TryGetUInt64(out var id))
            {
                return id;
            }
        }

        return 0;
    }

    private async Task<IReadOnlyList<DnsRecord>> GetRecordsTc3Async(string domain)
    {
        var payload = new Dictionary<string, object?>
        {
            ["Domain"] = domain,
            ["Limit"] = 3000
        };

        var body = await SendRequestTc3Async("DescribeRecordList", payload);
        var error = ExtractTc3Error(body);
        if (error != null)
        {
            if (error.Value.Code == "ResourceNotFound.NoDataOfRecord")
            {
                return Array.Empty<DnsRecord>();
            }
            throw new InvalidOperationException($"dnspod tc3 error: {error.Value.Code}");
        }

        using var doc = JsonDocument.Parse(body);
        var results = new List<DnsRecord>();
        if (!doc.RootElement.TryGetProperty("Response", out var response) ||
            !response.TryGetProperty("RecordList", out var list) || list.ValueKind != JsonValueKind.Array)
        {
            return results;
        }

        foreach (var item in list.EnumerateArray())
        {
            var ttl = ReadIntProperty(item, "TTL", 600);
            var weight = ReadIntProperty(item, "Weight", 0);
            results.Add(new DnsRecord
            {
                Type = item.TryGetProperty("Type", out var typeProp) ? (typeProp.GetString() ?? string.Empty) : string.Empty,
                Name = item.TryGetProperty("Name", out var nameProp) ? (nameProp.GetString() ?? string.Empty) : string.Empty,
                Value = item.TryGetProperty("Value", out var valueProp) ? (valueProp.GetString() ?? string.Empty) : string.Empty,
                Line = item.TryGetProperty("Line", out var lineProp) ? (lineProp.GetString() ?? string.Empty) : string.Empty,
                TTL = ttl,
                Weight = weight
            });
        }

        return results;
    }

    private async Task<string> SendRequestV2Async(string action, Dictionary<string, string> values)
    {
        var apiHost = "https://dnsapi.cn";
        var lang = "cn";
        if (string.Equals(_region.Trim(), InternationalRegion, StringComparison.OrdinalIgnoreCase))
        {
            apiHost = "https://api.dnspod.com";
            lang = "en";
        }

        values["login_token"] = _id + "," + _token;
        values["format"] = "json";
        values["lang"] = lang;
        values["error_on_empty"] = "no";

        var content = new FormUrlEncodedContent(values);
        var req = new HttpRequestMessage(HttpMethod.Post, apiHost + "/" + action)
        {
            Content = content
        };

        var (_, body) = await DnsHttp.SendAsync(req, TimeSpan.FromSeconds(30));
        return body;
    }

    private async Task<string> SendRequestTc3Async(string action, Dictionary<string, object?> payload)
    {
        const string host = "dnspod.tencentcloudapi.com";
        const string version = "2021-03-23";
        const string service = "dnspod";

        if (string.IsNullOrWhiteSpace(_secretId) || string.IsNullOrWhiteSpace(_secretKey))
        {
            throw new InvalidOperationException("dnspod secret_id/secret_key required");
        }

        var bodyJson = JsonSerializer.Serialize(payload);
        var timestamp = DateTimeOffset.UtcNow.ToUnixTimeSeconds();
        var date = DateTimeOffset.FromUnixTimeSeconds(timestamp).UtcDateTime.ToString("yyyy-MM-dd");

        var canonicalHeaders = "content-type:application/json; charset=utf-8\n" + "host:" + host + "\n";
        var signedHeaders = "content-type;host";
        var hashedPayload = Sha256Hex(bodyJson);
        var canonicalRequest = string.Join("\n", new[]
        {
            "POST",
            "/",
            string.Empty,
            canonicalHeaders,
            signedHeaders,
            hashedPayload
        });

        var credentialScope = date + "/" + service + "/tc3_request";
        var stringToSign = string.Join("\n", new[]
        {
            "TC3-HMAC-SHA256",
            timestamp.ToString(),
            credentialScope,
            Sha256Hex(canonicalRequest)
        });

        var signingKey = HmacSha256("TC3" + _secretKey, date);
        signingKey = HmacSha256(signingKey, service);
        signingKey = HmacSha256(signingKey, "tc3_request");
        var signature = ToHex(HmacSha256(signingKey, stringToSign));

        var authHeader = $"TC3-HMAC-SHA256 Credential={_secretId}/{credentialScope}, SignedHeaders={signedHeaders}, Signature={signature}";

        var req = new HttpRequestMessage(HttpMethod.Post, "https://" + host)
        {
            Content = new StringContent(bodyJson, Encoding.UTF8, "application/json")
        };
        req.Headers.TryAddWithoutValidation("Host", host);
        req.Headers.TryAddWithoutValidation("X-TC-Action", action);
        req.Headers.TryAddWithoutValidation("X-TC-Timestamp", timestamp.ToString());
        req.Headers.TryAddWithoutValidation("X-TC-Version", version);
        if (!string.IsNullOrWhiteSpace(_region) && !string.Equals(_region, InternationalRegion, StringComparison.OrdinalIgnoreCase))
        {
            req.Headers.TryAddWithoutValidation("X-TC-Region", _region);
        }
        req.Headers.TryAddWithoutValidation("Authorization", authHeader);

        var (_, body) = await DnsHttp.SendAsync(req, TimeSpan.FromSeconds(30));
        return body;
    }

    private static (string Code, string Message) ParseV2Status(string body)
    {
        using var doc = JsonDocument.Parse(body);
        var status = doc.RootElement.GetProperty("status");
        var code = status.GetProperty("code").GetString() ?? string.Empty;
        var message = status.TryGetProperty("message", out var msgProp) ? (msgProp.GetString() ?? string.Empty) : string.Empty;
        return (code, message);
    }

    private static (string Code, string Message)? ExtractTc3Error(string body)
    {
        using var doc = JsonDocument.Parse(body);
        if (!doc.RootElement.TryGetProperty("Response", out var response))
        {
            return null;
        }
        if (!response.TryGetProperty("Error", out var error) || error.ValueKind == JsonValueKind.Null)
        {
            return null;
        }
        var code = error.TryGetProperty("Code", out var codeProp) ? codeProp.GetString() ?? string.Empty : string.Empty;
        var message = error.TryGetProperty("Message", out var msgProp) ? msgProp.GetString() ?? string.Empty : string.Empty;
        return (code, message);
    }

    private static bool IsIgnorableTc3(string code, string message)
    {
        code = (code ?? string.Empty).Trim();
        return code is "InvalidParameter.DomainRecordExist" or "ResourceNotFound.NoDataOfRecord";
    }

    private static int ReadIntProperty(JsonElement root, string propertyName, int fallback)
    {
        if (!root.TryGetProperty(propertyName, out var value))
        {
            return fallback;
        }

        return value.ValueKind switch
        {
            JsonValueKind.Number when value.TryGetInt32(out var number) => number,
            JsonValueKind.Number when value.TryGetUInt64(out var unsigned) => (int)Math.Min(unsigned, int.MaxValue),
            JsonValueKind.String when int.TryParse(value.GetString(), out var parsed) => parsed,
            _ => fallback
        };
    }

    private static void SetRecordLinePayload(Dictionary<string, object?> payload, string line)
    {
        if (string.IsNullOrWhiteSpace(line))
        {
            payload["RecordLine"] = "Default";
            return;
        }

        var trimmed = line.Trim();
        if (trimmed.Contains('='))
        {
            payload["RecordLine"] = "Default";
            payload["RecordLineId"] = trimmed;
            return;
        }

        payload["RecordLine"] = trimmed;
    }

    private static bool LineMatches(JsonElement item, string? expectedLine)
    {
        if (string.IsNullOrWhiteSpace(expectedLine))
        {
            return true;
        }

        var expected = expectedLine.Trim();
        var line = item.TryGetProperty("Line", out var lineProp) ? lineProp.GetString() : null;
        if (!string.IsNullOrWhiteSpace(line) && string.Equals(line, expected, StringComparison.OrdinalIgnoreCase))
        {
            return true;
        }

        var lineId = item.TryGetProperty("LineId", out var lineIdProp) ? lineIdProp.GetString() : null;
        return !string.IsNullOrWhiteSpace(lineId) && string.Equals(lineId, expected, StringComparison.OrdinalIgnoreCase);
    }

    private static ulong ReadUInt64Property(JsonElement root, string propertyName)
    {
        if (!root.TryGetProperty(propertyName, out var value))
        {
            return 0;
        }

        return value.ValueKind switch
        {
            JsonValueKind.Number when value.TryGetUInt64(out var unsigned) => unsigned,
            JsonValueKind.Number when value.TryGetInt64(out var signed) && signed > 0 => (ulong)signed,
            JsonValueKind.String when ulong.TryParse(value.GetString(), out var parsed) => parsed,
            _ => 0
        };
    }

    private static string Sha256Hex(string data)
    {
        var hash = SHA256.HashData(Encoding.UTF8.GetBytes(data));
        return ToHex(hash);
    }

    private static byte[] HmacSha256(string key, string msg)
    {
        using var hmac = new HMACSHA256(Encoding.UTF8.GetBytes(key));
        return hmac.ComputeHash(Encoding.UTF8.GetBytes(msg));
    }

    private static byte[] HmacSha256(byte[] key, string msg)
    {
        using var hmac = new HMACSHA256(key);
        return hmac.ComputeHash(Encoding.UTF8.GetBytes(msg));
    }

    private static string ToHex(byte[] buffer)
    {
        var sb = new StringBuilder(buffer.Length * 2);
        foreach (var b in buffer)
        {
            sb.Append(b.ToString("x2"));
        }
        return sb.ToString();
    }

    private static string? GetString(JsonElement root, string name)
    {
        return root.TryGetProperty(name, out var prop) ? prop.GetString() : null;
    }

    private sealed class V2Record
    {
        public string Id { get; set; } = string.Empty;
        public string Name { get; set; } = string.Empty;
        public string Type { get; set; } = string.Empty;
        public string Value { get; set; } = string.Empty;
        public string Line { get; set; } = string.Empty;
        public int Ttl { get; set; }
        public int Weight { get; set; }
    }
}
