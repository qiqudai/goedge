namespace Cnn.Api.Services.Tasks.Workflow;

public sealed class TaskRequestResult
{
    public long TaskId { get; init; }
    public string TaskNo { get; init; } = default!;
    public string State { get; init; } = "pending";
}
