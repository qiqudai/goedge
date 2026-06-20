using Cnn.Domain.Entities;
using SqlSugar;
using Cnn.Api.Services.Deletion;
using System.Text.Json;
using System.Text.Json.Serialization;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Tasks.Workflow.Handlers;

public sealed class CertificateDeleteTaskHandler : ITaskHandler
{
    private readonly ISqlSugarClient _db;

    public CertificateDeleteTaskHandler(ISqlSugarClient db)
    {
        _db = db;
    }

    public string TaskType => AsyncTaskTypes.CertificateDelete;

    public async Task HandleAsync(long taskId, string payloadJson, CancellationToken cancellationToken)
    {
        var payload = ParsePayload(payloadJson);
        if (payload.ResourceId <= 0)
        {
            throw new InvalidOperationException("certificate delete payload is missing resource_id");
        }

        var certId = (int)payload.ResourceId;
        var cert = await _db.Queryable<Cert>().Where(c => c.Id == certId).FirstAsync();
        if (cert == null)
        {
            return;
        }

        var sites = await CertificateUsageInspector.FindSiteUsagesAsync(_db, certId, cancellationToken);
        if (sites.Count > 0)
        {
            throw new InvalidOperationException("certificate is still referenced by sites");
        }

        await _db.Deleteable<Cert>().Where(c => c.Id == certId).ExecuteCommandAsync();
    }

    private static DeletePayload ParsePayload(string payloadJson)
    {
        if (string.IsNullOrWhiteSpace(payloadJson))
        {
            return new DeletePayload();
        }

        try
        {
            return JsonSerializer.Deserialize<DeletePayload>(payloadJson, new JsonSerializerOptions(JsonSerializerDefaults.Web))
                ?? new DeletePayload();
        }
        catch
        {
            return new DeletePayload();
        }
    }

    private sealed class DeletePayload
    {
        [JsonPropertyName("resource_id")]
        public long ResourceId { get; init; }
    }
}
