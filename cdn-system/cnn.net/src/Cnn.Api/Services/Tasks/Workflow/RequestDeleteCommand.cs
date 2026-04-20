namespace Cnn.Api.Services.Tasks.Workflow;

public sealed class RequestDeleteCommand
{
    public string ResourceType { get; init; } = default!;
    public long ResourceId { get; init; }
    public long? OwnerUserId { get; init; }
    public long? OperatorUserId { get; init; }
    public string? DedupeKey { get; init; }
    public string? RequestedReason { get; init; }
}
