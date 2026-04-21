using System.Text.Json;
using Cnn.Common.Contracts.Admin;
using Cnn.Api.Services.Admin;

namespace Cnn.Api.Services.Tasks.Workflow.Handlers;

public sealed class SiteCreateTaskHandler : ITaskHandler
{
    private readonly ISiteService _siteService;

    public SiteCreateTaskHandler(ISiteService siteService)
    {
        _siteService = siteService;
    }

    public string TaskType => "site_create";

    public async Task HandleAsync(long taskId, string payloadJson, CancellationToken cancellationToken)
    {
        var payload = ParsePayload(payloadJson);
        if (payload.UserId <= 0 || string.IsNullOrWhiteSpace(payload.Domain))
        {
            throw new InvalidOperationException("site create payload is invalid");
        }

        var request = new SiteCreateRequest
        {
            UserId = payload.UserId,
            UserPackageId = payload.UserPackageId,
            DnsProviderId = payload.DnsProviderId,
            GroupId = payload.GroupId,
            GroupIds = payload.GroupId > 0 ? new[] { payload.GroupId } : null,
            NodeGroupId = payload.NodeGroupId,
            Domains = new[] { payload.Domain.Trim() },
            Backends = payload.Backends?.Where(item => !string.IsNullOrWhiteSpace(item)).Select(item => item.Trim()).ToList()
        };

        var result = await _siteService.CreateAsync(request, payload.UserId, true, cancellationToken);
        if (!result.Success)
        {
            throw new InvalidOperationException(result.MessageKey ?? $"site_create_failed:{result.ErrorCode}");
        }
    }

    private static SiteCreatePayload ParsePayload(string payloadJson)
    {
        if (string.IsNullOrWhiteSpace(payloadJson))
        {
            return new SiteCreatePayload();
        }

        try
        {
            var parsed = JsonSerializer.Deserialize<SiteCreatePayload>(payloadJson, new JsonSerializerOptions(JsonSerializerDefaults.Web)
            {
                PropertyNameCaseInsensitive = true
            });
            if (parsed != null && parsed.UserId > 0 && !string.IsNullOrWhiteSpace(parsed.Domain))
            {
                return parsed;
            }

            using var doc = JsonDocument.Parse(payloadJson);
            var root = doc.RootElement;
            return new SiteCreatePayload
            {
                UserId = ReadLong(root, "user_id", "userId"),
                UserPackageId = ReadLong(root, "user_package_id", "userPackageId"),
                DnsProviderId = ReadLong(root, "dns_provider_id", "dnsProviderId"),
                NodeGroupId = ReadLong(root, "node_group_id", "nodeGroupId"),
                GroupId = ReadLong(root, "group_id", "groupId"),
                Domain = ReadString(root, "domain") ?? string.Empty,
                Backends = ReadStringList(root, "backends")
            };
        }
        catch
        {
            return new SiteCreatePayload();
        }
    }

    private sealed class SiteCreatePayload
    {
        public long UserId { get; init; }
        public long UserPackageId { get; init; }
        public long DnsProviderId { get; init; }
        public long NodeGroupId { get; init; }
        public long GroupId { get; init; }
        public string Domain { get; init; } = string.Empty;
        public List<string> Backends { get; init; } = new();
    }

    private static long ReadLong(JsonElement root, params string[] names)
    {
        foreach (var name in names)
        {
            if (!root.TryGetProperty(name, out var value))
            {
                continue;
            }

            switch (value.ValueKind)
            {
                case JsonValueKind.Number when value.TryGetInt64(out var number):
                    return number;
                case JsonValueKind.String when long.TryParse(value.GetString(), out var parsed):
                    return parsed;
            }
        }

        return 0;
    }

    private static string? ReadString(JsonElement root, params string[] names)
    {
        foreach (var name in names)
        {
            if (root.TryGetProperty(name, out var value) && value.ValueKind == JsonValueKind.String)
            {
                return value.GetString();
            }
        }

        return null;
    }

    private static List<string> ReadStringList(JsonElement root, params string[] names)
    {
        foreach (var name in names)
        {
            if (!root.TryGetProperty(name, out var value) || value.ValueKind != JsonValueKind.Array)
            {
                continue;
            }

            var result = new List<string>();
            foreach (var item in value.EnumerateArray())
            {
                if (item.ValueKind != JsonValueKind.String)
                {
                    continue;
                }

                var text = item.GetString()?.Trim();
                if (!string.IsNullOrWhiteSpace(text))
                {
                    result.Add(text);
                }
            }

            return result;
        }

        return new List<string>();
    }
}
