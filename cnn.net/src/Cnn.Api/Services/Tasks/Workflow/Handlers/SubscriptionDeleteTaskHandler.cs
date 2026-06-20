using System.Text.Json;
using System.Text.Json.Serialization;
using Cnn.Domain.Entities;
using SqlSugar;
using Task = System.Threading.Tasks.Task;
using StreamEntity = Cnn.Domain.Entities.Stream;

namespace Cnn.Api.Services.Tasks.Workflow.Handlers;

public sealed class SubscriptionDeleteTaskHandler : ITaskHandler
{
    private readonly ISqlSugarClient _db;

    public SubscriptionDeleteTaskHandler(ISqlSugarClient db)
    {
        _db = db;
    }

    public string TaskType => AsyncTaskTypes.SubscriptionDelete;

    public async Task HandleAsync(long taskId, string payloadJson, CancellationToken cancellationToken)
    {
        var payload = ParsePayload(payloadJson);
        if (payload.ResourceId <= 0)
        {
            throw new InvalidOperationException("subscription delete payload is missing resource_id");
        }

        var subscriptionId = (int)payload.ResourceId;
        var subscription = await _db.Queryable<UserPackage>()
            .Where(x => x.Id == subscriptionId)
            .FirstAsync();
        if (subscription == null)
        {
            return;
        }

        var siteRefs = await _db.Queryable<Site>()
            .Where(x => x.UserPackage == subscriptionId)
            .CountAsync();
        if (siteRefs > 0)
        {
            throw new InvalidOperationException("subscription is still referenced by sites");
        }

        var streamRefs = await _db.Queryable<StreamEntity>()
            .Where(x => x.UserPackage == subscriptionId)
            .CountAsync();
        if (streamRefs > 0)
        {
            throw new InvalidOperationException("subscription is still referenced by streams");
        }

        await _db.Deleteable<UserPackage>()
            .Where(x => x.Id == subscriptionId)
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
