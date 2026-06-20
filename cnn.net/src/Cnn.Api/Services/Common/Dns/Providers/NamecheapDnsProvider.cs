using System.Text.Json;
using System.Xml.Linq;

namespace Cnn.Api.Services.Common.Dns.Providers;

internal sealed class NamecheapDnsProvider : IDnsRecordProvider
{
    private readonly string _user;
    private readonly string _apiKey;
    private readonly string _clientIp;
    private readonly string _endpoint;

    private NamecheapDnsProvider(string user, string apiKey, string clientIp, bool sandbox)
    {
        _user = user;
        _apiKey = apiKey;
        _clientIp = clientIp;
        _endpoint = sandbox ? "https://api.sandbox.namecheap.com/xml.response" : "https://api.namecheap.com/xml.response";
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
            var user = GetString(root, "user") ?? GetString(root, "username") ?? string.Empty;
            var apiKey = GetString(root, "api_key") ?? string.Empty;
            var ip = GetString(root, "ip") ?? GetString(root, "client_ip") ?? string.Empty;
            var sandbox = GetBool(root, "sandbox");

            return new NamecheapDnsProvider(user.Trim(), apiKey.Trim(), ip.Trim(), sandbox);
        }
        catch
        {
            return null;
        }
    }

    public async Task<IReadOnlyList<DnsRecord>> GetRecordsAsync(string domain)
    {
        var (records, _) = await GetHostsAsync(domain);
        var output = new List<DnsRecord>(records.Count);
        foreach (var item in records)
        {
            output.Add(new DnsRecord
            {
                Type = item.RecordType,
                Name = item.HostName,
                Value = item.Address,
                TTL = item.TTL
            });
        }

        return output;
    }

    public async Task AddRecordAsync(string domain, DnsRecord record)
    {
        var (records, split) = await GetHostsAsync(domain);
        var host = NormalizeHost(record.Name);
        var ttl = record.TTL <= 0 ? 300 : record.TTL;

        foreach (var item in records)
        {
            if (string.Equals(item.HostName, host, StringComparison.OrdinalIgnoreCase) &&
                string.Equals(item.RecordType, record.Type, StringComparison.OrdinalIgnoreCase) &&
                string.Equals(item.Address, record.Value, StringComparison.OrdinalIgnoreCase))
            {
                return;
            }
        }

        records.Add(new NamecheapHostRecord(host, record.Type, record.Value, ttl, string.Empty));
        await SetHostsAsync(split, records);
    }

    public async Task DeleteRecordAsync(string domain, DnsRecord record)
    {
        var (records, split) = await GetHostsAsync(domain);
        var host = NormalizeHost(record.Name);
        var removed = records.RemoveAll(item =>
            string.Equals(item.HostName, host, StringComparison.OrdinalIgnoreCase) &&
            string.Equals(item.RecordType, record.Type, StringComparison.OrdinalIgnoreCase) &&
            string.Equals(item.Address, record.Value, StringComparison.OrdinalIgnoreCase));

        if (removed > 0)
        {
            await SetHostsAsync(split, records);
        }
    }

    private async Task<(List<NamecheapHostRecord> Records, DomainSplit Split)> GetHostsAsync(string domain)
    {
        var candidates = GetDomainSplits(domain);
        foreach (var split in candidates)
        {
            var response = await SendRequestAsync(BuildParams(split, "namecheap.domains.dns.getHosts"));
            var errors = response.Errors;
            if (errors.Count == 0)
            {
                return (response.Records, split);
            }

            if (errors.Any(IsDomainNotFound))
            {
                continue;
            }

            throw new InvalidOperationException("namecheap error: " + string.Join("; ", errors));
        }

        throw new InvalidOperationException("namecheap error: domain not found");
    }

    private async Task SetHostsAsync(DomainSplit split, List<NamecheapHostRecord> records)
    {
        var parameters = BuildParams(split, "namecheap.domains.dns.setHosts");
        var index = 1;
        foreach (var record in records)
        {
            parameters[$"HostName{index}"] = record.HostName;
            parameters[$"RecordType{index}"] = record.RecordType;
            parameters[$"Address{index}"] = record.Address;
            parameters[$"TTL{index}"] = record.TTL <= 0 ? "300" : record.TTL.ToString();
            if (!string.IsNullOrWhiteSpace(record.MXPref))
            {
                parameters[$"MXPref{index}"] = record.MXPref;
            }
            index++;
        }

        var response = await SendRequestAsync(parameters);
        if (response.Errors.Count > 0)
        {
            throw new InvalidOperationException("namecheap error: " + string.Join("; ", response.Errors));
        }
    }

    private async Task<NamecheapResponse> SendRequestAsync(Dictionary<string, string> parameters)
    {
        if (string.IsNullOrWhiteSpace(_user) || string.IsNullOrWhiteSpace(_apiKey) || string.IsNullOrWhiteSpace(_clientIp))
        {
            throw new InvalidOperationException("namecheap credentials missing user/api_key/ip");
        }

        var req = new HttpRequestMessage(HttpMethod.Post, _endpoint)
        {
            Content = new FormUrlEncodedContent(parameters)
        };

        var (_, body) = await DnsHttp.SendAsync(req, TimeSpan.FromSeconds(30));
        if (string.IsNullOrWhiteSpace(body))
        {
            throw new InvalidOperationException("namecheap empty response");
        }

        return ParseResponse(body);
    }

    private Dictionary<string, string> BuildParams(DomainSplit split, string command)
    {
        return new Dictionary<string, string>
        {
            ["ApiUser"] = _user,
            ["ApiKey"] = _apiKey,
            ["UserName"] = _user,
            ["Command"] = command,
            ["ClientIp"] = _clientIp,
            ["SLD"] = split.Sld,
            ["TLD"] = split.Tld
        };
    }

    private static NamecheapResponse ParseResponse(string body)
    {
        var doc = XDocument.Parse(body);
        var root = doc.Root;
        if (root == null)
        {
            return NamecheapResponse.Empty;
        }

        var ns = root.Name.Namespace;
        var errors = root.Element(ns + "Errors")?.Elements(ns + "Error")
            .Select(e => (e.Value ?? string.Empty).Trim())
            .Where(e => !string.IsNullOrWhiteSpace(e))
            .ToList() ?? new List<string>();

        var records = new List<NamecheapHostRecord>();
        var result = root.Element(ns + "CommandResponse")?.Element(ns + "DomainDNSGetHostsResult");
        if (result != null)
        {
            foreach (var host in result.Elements(ns + "host"))
            {
                var name = host.Attribute("Name")?.Value ?? string.Empty;
                var type = host.Attribute("Type")?.Value ?? string.Empty;
                var address = host.Attribute("Address")?.Value ?? string.Empty;
                var ttlRaw = host.Attribute("TTL")?.Value ?? string.Empty;
                var mx = host.Attribute("MXPref")?.Value ?? string.Empty;
                _ = int.TryParse(ttlRaw, out var ttl);
                records.Add(new NamecheapHostRecord(NormalizeHost(name), type, address, ttl, mx));
            }
        }

        return new NamecheapResponse(records, errors);
    }

    private static bool IsDomainNotFound(string error)
    {
        if (string.IsNullOrWhiteSpace(error))
        {
            return false;
        }

        return error.Contains("not found", StringComparison.OrdinalIgnoreCase) ||
               error.Contains("does not exist", StringComparison.OrdinalIgnoreCase) ||
               error.Contains("no such domain", StringComparison.OrdinalIgnoreCase);
    }

    private static IEnumerable<DomainSplit> GetDomainSplits(string domain)
    {
        var normalized = NormalizeDomain(domain);
        var parts = normalized.Split('.', StringSplitOptions.RemoveEmptyEntries);
        if (parts.Length < 2)
        {
            yield break;
        }

        yield return new DomainSplit(parts[^2], parts[^1]);
        if (parts.Length >= 3)
        {
            var tld = parts[^2] + "." + parts[^1];
            yield return new DomainSplit(parts[^3], tld);
        }
    }

    private static string NormalizeHost(string? input)
    {
        var host = (input ?? string.Empty).Trim();
        if (string.IsNullOrWhiteSpace(host))
        {
            return "@";
        }

        return host;
    }

    private static string NormalizeDomain(string? input)
    {
        var value = (input ?? string.Empty).Trim().TrimEnd('.');
        return value.ToLowerInvariant();
    }

    private static string? GetString(JsonElement root, string name)
    {
        return root.TryGetProperty(name, out var prop) ? prop.GetString() : null;
    }

    private static bool GetBool(JsonElement root, string name)
    {
        if (!root.TryGetProperty(name, out var prop))
        {
            return false;
        }

        if (prop.ValueKind == JsonValueKind.True)
        {
            return true;
        }

        if (prop.ValueKind == JsonValueKind.String &&
            bool.TryParse(prop.GetString(), out var value))
        {
            return value;
        }

        return false;
    }

    private sealed record DomainSplit(string Sld, string Tld);

    private sealed record NamecheapHostRecord(string HostName, string RecordType, string Address, int TTL, string MXPref);

    private sealed record NamecheapResponse(List<NamecheapHostRecord> Records, List<string> Errors)
    {
        public static NamecheapResponse Empty { get; } = new(new List<NamecheapHostRecord>(), new List<string>());
    }
}
