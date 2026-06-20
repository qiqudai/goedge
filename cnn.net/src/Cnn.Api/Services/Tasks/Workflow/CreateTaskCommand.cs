namespace Cnn.Api.Services.Tasks.Workflow;

public sealed class CreateTaskCommand
{
    public string TaskType { get; init; } = default!;
    public long? OwnerUserId { get; init; }
    public long? OperatorUserId { get; init; }
    public string? ResourceType { get; init; }
    public long? ResourceId { get; init; }
    public string? DedupeKey { get; init; }
    public string PayloadJson { get; init; } = "{}";
    public DateTime? ScheduledAt { get; init; }
}
