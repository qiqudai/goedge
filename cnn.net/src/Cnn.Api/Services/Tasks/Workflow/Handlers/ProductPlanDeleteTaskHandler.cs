using System.Text.Json;
using System.Text.Json.Serialization;
using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;

namespace Cnn.Api.Services.Tasks.Workflow.Handlers;

public sealed class ProductPlanDeleteTaskHandler : ITaskHandler
{
    private readonly ISqlSugarClient _db;

    public ProductPlanDeleteTaskHandler(ISqlSugarClient db)
    {
        _db = db;
    }

    public string TaskType => AsyncTaskTypes.ProductPlanDelete;

    public async Task HandleAsync(long taskId, string payloadJson, CancellationToken cancellationToken)
    {
        var payload = ParsePayload(payloadJson);
        if (payload.ResourceId <= 0)
        {
            throw new InvalidOperationException("product plan delete payload is missing resource_id");
        }

        var planId = (int)payload.ResourceId;
        var plan = await _db.Queryable<Package>()
            .Where(x => x.Id == planId)
            .FirstAsync();
        if (plan == null)
        {
            return;
        }

        var soldRefs = await _db.Queryable<UserPackage>()
            .Where(x => x.Package == planId)
            .CountAsync();
        if (soldRefs > 0)
        {
            throw new InvalidOperationException("product plan is still referenced by sold packages");
        }

        await _db.Deleteable<MergePackageGroup>()
            .Where(x => x.PackageId == planId)
            .ExecuteCommandAsync();

        await _db.Deleteable<Package>()
            .Where(x => x.Id == planId)
            .ExecuteCommandAsync();
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
