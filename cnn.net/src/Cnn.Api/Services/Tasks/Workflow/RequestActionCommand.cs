namespace Cnn.Api.Services.Tasks.Workflow;

public sealed class RequestActionCommand
{
    public string TaskType { get; init; } = default!;
    public string? ResourceType { get; init; }
    public long? ResourceId { get; init; }
    public long? OwnerUserId { get; init; }
    public long? OperatorUserId { get; init; }
    public string? DedupeKey { get; init; }
    public string PayloadJson { get; init; } = "{}";
}
