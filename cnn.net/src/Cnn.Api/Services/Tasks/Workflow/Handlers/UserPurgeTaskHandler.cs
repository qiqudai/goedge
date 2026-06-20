using System.Text.Json;
using System.Text.Json.Serialization;
using Cnn.Api.Services.Users;

namespace Cnn.Api.Services.Tasks.Workflow.Handlers;

public sealed class UserPurgeTaskHandler : ITaskHandler
{
    private readonly IUserPurgeExecutor _userPurgeExecutor;

    public UserPurgeTaskHandler(IUserPurgeExecutor userPurgeExecutor)
    {
        _userPurgeExecutor = userPurgeExecutor;
    }

    public string TaskType => AsyncTaskTypes.UserPurge;

    public Task HandleAsync(long taskId, string payloadJson, CancellationToken cancellationToken)
    {
        var payload = ParsePayload(payloadJson);
        if (payload.ResourceId <= 0)
        {
            throw new InvalidOperationException("user purge payload is missing resource_id");
        }

        return _userPurgeExecutor.ExecuteAsync(payload.ResourceId, cancellationToken);
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
