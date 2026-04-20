using System.Text.Json;

namespace Cnn.Api.Services.Tasks.Workflow.Handlers;

public sealed class ConfigSyncTaskHandler : ITaskHandler
{
    public string TaskType => AsyncTaskTypes.ConfigSync;

    public Task HandleAsync(long taskId, string payloadJson, CancellationToken cancellationToken)
    {
        if (string.IsNullOrWhiteSpace(payloadJson))
        {
            return Task.CompletedTask;
        }

        using var document = JsonDocument.Parse(payloadJson);
        _ = document.RootElement;
        return Task.CompletedTask;
    }
}
