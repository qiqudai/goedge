using Cnn.Api.Services.Deletion;

namespace Cnn.Api.Services.Tasks.Workflow;

public sealed class DeleteRequestResult
{
    public bool Queued { get; init; }
    public string? ErrorCode { get; init; }
    public string? Message { get; init; }
    public TaskRequestResult? Task { get; init; }
    public IReadOnlyList<DeleteReferenceItem> References { get; init; } = Array.Empty<DeleteReferenceItem>();
}
