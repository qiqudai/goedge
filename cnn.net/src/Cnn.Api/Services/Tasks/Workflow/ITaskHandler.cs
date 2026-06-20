namespace Cnn.Api.Services.Tasks.Workflow;

public interface ITaskHandler
{
    string TaskType { get; }

    bool CanHandle(string taskType) => string.Equals(TaskType, taskType, StringComparison.OrdinalIgnoreCase);

    Task HandleAsync(long taskId, string payloadJson, CancellationToken cancellationToken);
}
