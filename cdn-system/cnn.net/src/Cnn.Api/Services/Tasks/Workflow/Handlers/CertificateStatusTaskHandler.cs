using System.Text.Json;
using System.Text.Json.Serialization;
using Cnn.Api.Services.Deletion;
using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Tasks.Workflow.Handlers;

public sealed class CertificateStatusTaskHandler : ITaskHandler
{
    private readonly ISqlSugarClient _db;
    private readonly IConfigVersionService _configVersionService;

    public CertificateStatusTaskHandler(
        ISqlSugarClient db,
        IConfigVersionService configVersionService)
    {
        _db = db;
        _configVersionService = configVersionService;
    }

    public string TaskType => throw new NotSupportedException("Resolve task type via CanHandle.");

    public bool CanHandle(string taskType)
    {
        return string.Equals(taskType, AsyncTaskTypes.CertificateEnable, StringComparison.OrdinalIgnoreCase)
               || string.Equals(taskType, AsyncTaskTypes.CertificateDisable, StringComparison.OrdinalIgnoreCase);
    }

    public async Task HandleAsync(long taskId, string payloadJson, CancellationToken cancellationToken)
    {
        var payload = ParsePayload(payloadJson);
        var certIds = NormalizeIds(payload.ResourceIds);
        if (certIds.Count == 0 || payload.Enable == null)
        {
            throw new InvalidOperationException("certificate status payload is invalid");
        }

        if (!payload.Enable.Value)
        {
            foreach (var certId in certIds)
            {
                var usages = await CertificateUsageInspector.FindSiteUsagesAsync(_db, certId, cancellationToken);
                if (usages.Count > 0)
                {
                    throw new InvalidOperationException("certificate is still referenced by sites");
                }
            }
        }

        if (payload.Enable.Value)
        {
            await _db.Updateable<Cert>()
                .SetColumns(c => new Cert { Enable = true })
                .Where(c => certIds.Contains(c.Id))
                .ExecuteCommandAsync();
        }
        else if (payload.DisableAutoRenew)
        {
            await _db.Updateable(new Cert
                {
                    Enable = false,
                    AutoRenew = false
                })
                .UpdateColumns(c => new { c.Enable, c.AutoRenew })
                .Where(c => certIds.Contains(c.Id))
                .ExecuteCommandAsync();
        }
        else
        {
            await _db.Updateable(new Cert
                {
                    Enable = false
                })
                .UpdateColumns(c => new { c.Enable })
                .Where(c => certIds.Contains(c.Id))
                .ExecuteCommandAsync();
        }

        await _configVersionService.BumpAsync("cert", certIds, cancellationToken);
    }

    private static CertificateStatusPayload ParsePayload(string payloadJson)
    {
        if (string.IsNullOrWhiteSpace(payloadJson))
        {
            return new CertificateStatusPayload();
        }

        try
        {
            return JsonSerializer.Deserialize<CertificateStatusPayload>(payloadJson, new JsonSerializerOptions(JsonSerializerDefaults.Web))
                   ?? new CertificateStatusPayload();
        }
        catch
        {
            return new CertificateStatusPayload();
        }
    }

    private static List<long> NormalizeIds(IReadOnlyList<long>? ids)
    {
        return ids?
            .Where(id => id > 0)
            .Distinct()
            .OrderBy(id => id)
            .ToList()
               ?? new List<long>();
    }

    private sealed class CertificateStatusPayload
    {
        [JsonPropertyName("resource_ids")]
        public IReadOnlyList<long>? ResourceIds { get; init; }
        [JsonPropertyName("enable")]
        public bool? Enable { get; init; }
        [JsonPropertyName("disable_auto_renew")]
        public bool DisableAutoRenew { get; init; }
    }
}
